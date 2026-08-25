package grader

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/gustavfredrikson/cka-dojo/internal/environment"
)

func init() {
	register(func() Checker { return &ObjectExists{} })
	register(func() Checker { return &JSONPath{} })
	register(func() Checker { return &DeploymentAvailable{} })
	register(func() Checker { return &ServiceHasEndpoints{} })
	register(func() Checker { return &AuthCanI{} })
	register(func() Checker { return &HTTPService{} })
}

func ns(n string) string {
	if n == "" {
		return "default"
	}
	return n
}

func nsArgs(args []string, namespace string) []string {
	if namespace != "" {
		return append(args, "-n", namespace)
	}
	return args
}

// ObjectExists checks that a named object is present, or deliberately absent.
type ObjectExists struct {
	Kind      string `yaml:"kind"`
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace"`
	// Absent inverts the check.
	Absent bool `yaml:"absent"`
}

func (o *ObjectExists) Type() string { return "objectExists" }

func (o *ObjectExists) Validate() error {
	if o.Kind == "" || o.Name == "" {
		return fmt.Errorf("kind and name are required")
	}
	return nil
}

func (o *ObjectExists) Describe() string {
	verb := "exists"
	if o.Absent {
		verb = "does not exist"
	}
	return fmt.Sprintf("%s/%s in %s %s", o.Kind, o.Name, ns(o.Namespace), verb)
}

func (o *ObjectExists) Check(ctx context.Context, env *environment.Manager) Result {
	res, err := env.KubectlRaw(ctx, nsArgs([]string{"get", o.Kind, o.Name, "-o", "name"}, o.Namespace)...)
	if err != nil {
		return broken(o.Describe(), err)
	}
	found := res.ExitCode == 0
	if found == !o.Absent {
		return pass(o.Describe())
	}
	if o.Absent {
		return fail(o.Describe(), "it is still there")
	}
	return fail(o.Describe(), "not found")
}

// Match is the comparison shared by the text-producing checkers.
type Match struct {
	Equals   *string `yaml:"equals"`
	Contains string  `yaml:"contains"`
	Matches  string  `yaml:"matches"`
	// NotEquals is useful for "the value changed" checks.
	NotEquals *string `yaml:"notEquals"`
}

func (m *Match) empty() bool {
	return m.Equals == nil && m.Contains == "" && m.Matches == "" && m.NotEquals == nil
}

func (m *Match) validate() error {
	if m.Matches != "" {
		if _, err := regexp.Compile(m.Matches); err != nil {
			return fmt.Errorf("matches is not a valid regexp: %w", err)
		}
	}
	return nil
}

// describe renders the expectation in words.
func (m *Match) describe() string {
	switch {
	case m.Equals != nil:
		return "is " + strconv.Quote(*m.Equals)
	case m.NotEquals != nil:
		return "is not " + strconv.Quote(*m.NotEquals)
	case m.Contains != "":
		return "contains " + strconv.Quote(m.Contains)
	case m.Matches != "":
		return "matches /" + m.Matches + "/"
	default:
		return "is non-empty"
	}
}

func (m *Match) ok(got string) bool {
	got = strings.TrimSpace(got)
	switch {
	case m.Equals != nil:
		return got == *m.Equals
	case m.NotEquals != nil:
		return got != *m.NotEquals
	case m.Contains != "":
		return strings.Contains(got, m.Contains)
	case m.Matches != "":
		re, err := regexp.Compile(m.Matches)
		return err == nil && re.MatchString(got)
	default:
		return got != ""
	}
}

// JSONPath evaluates a jsonpath expression against one object. This is the
// general-purpose check that keeps the grader vocabulary small.
type JSONPath struct {
	Kind      string `yaml:"kind"`
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace"`
	Path      string `yaml:"path"`
	Match     `yaml:",inline"`
}

func (j *JSONPath) Type() string { return "jsonPath" }

func (j *JSONPath) Validate() error {
	if j.Kind == "" || j.Name == "" || j.Path == "" {
		return fmt.Errorf("kind, name and path are required")
	}
	return j.Match.validate()
}

func (j *JSONPath) Describe() string {
	return fmt.Sprintf("%s/%s in %s: %s %s", j.Kind, j.Name, ns(j.Namespace), j.Path, j.Match.describe())
}

func (j *JSONPath) Check(ctx context.Context, env *environment.Manager) Result {
	args := nsArgs([]string{"get", j.Kind, j.Name, "-o", "jsonpath=" + j.Path}, j.Namespace)
	res, err := env.KubectlRaw(ctx, args...)
	if err != nil {
		return broken(j.Describe(), err)
	}
	if res.ExitCode != 0 {
		return fail(j.Describe(), "cannot read the object: %s", firstLine(res.Stderr))
	}
	got := strings.TrimSpace(res.Stdout)
	if j.Match.ok(got) {
		return pass(j.Describe())
	}
	if got == "" {
		return fail(j.Describe(), "the field is empty or missing")
	}
	return fail(j.Describe(), "found %q", got)
}

// DeploymentAvailable checks that a Deployment has its replicas available,
// which is the honest definition of "the app is running".
type DeploymentAvailable struct {
	Namespace string `yaml:"namespace"`
	Name      string `yaml:"name"`
	// MinReplicas defaults to 1.
	MinReplicas int `yaml:"minReplicas"`
}

func (d *DeploymentAvailable) Type() string { return "deploymentAvailable" }

func (d *DeploymentAvailable) Validate() error {
	if d.Name == "" {
		return fmt.Errorf("name is required")
	}
	if d.MinReplicas < 0 {
		return fmt.Errorf("minReplicas cannot be negative")
	}
	return nil
}

func (d *DeploymentAvailable) want() int {
	if d.MinReplicas == 0 {
		return 1
	}
	return d.MinReplicas
}

func (d *DeploymentAvailable) Describe() string {
	return fmt.Sprintf("deployment %s in %s has at least %d available replica(s)", d.Name, ns(d.Namespace), d.want())
}

func (d *DeploymentAvailable) Check(ctx context.Context, env *environment.Manager) Result {
	args := nsArgs([]string{"get", "deployment", d.Name, "-o", "jsonpath={.status.availableReplicas}"}, d.Namespace)
	res, err := env.KubectlRaw(ctx, args...)
	if err != nil {
		return broken(d.Describe(), err)
	}
	if res.ExitCode != 0 {
		return fail(d.Describe(), "the deployment was not found")
	}
	got := strings.TrimSpace(res.Stdout)
	n, _ := strconv.Atoi(got)
	if n >= d.want() {
		return pass(d.Describe())
	}
	return fail(d.Describe(), "%d available", n)
}

// ServiceHasEndpoints checks that a Service actually resolves to backends.
// A Service with no EndpointSlice addresses is the single most common
// networking failure on the exam.
type ServiceHasEndpoints struct {
	Namespace string `yaml:"namespace"`
	Name      string `yaml:"name"`
	// MinEndpoints defaults to 1.
	MinEndpoints int `yaml:"minEndpoints"`
	// Port, when set, additionally requires the Service to expose it.
	Port int `yaml:"port"`
}

func (s *ServiceHasEndpoints) Type() string { return "serviceHasEndpoints" }

func (s *ServiceHasEndpoints) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}

func (s *ServiceHasEndpoints) want() int {
	if s.MinEndpoints == 0 {
		return 1
	}
	return s.MinEndpoints
}

func (s *ServiceHasEndpoints) Describe() string {
	if s.Port > 0 {
		return fmt.Sprintf("service %s in %s exposes port %d and has at least %d ready endpoint(s)",
			s.Name, ns(s.Namespace), s.Port, s.want())
	}
	return fmt.Sprintf("service %s in %s has at least %d ready endpoint(s)", s.Name, ns(s.Namespace), s.want())
}

func (s *ServiceHasEndpoints) Check(ctx context.Context, env *environment.Manager) Result {
	if s.Port > 0 {
		args := nsArgs([]string{"get", "service", s.Name, "-o", "jsonpath={.spec.ports[*].port}"}, s.Namespace)
		res, err := env.KubectlRaw(ctx, args...)
		if err != nil {
			return broken(s.Describe(), err)
		}
		if res.ExitCode != 0 {
			return fail(s.Describe(), "the service was not found")
		}
		if !containsField(res.Stdout, strconv.Itoa(s.Port)) {
			return fail(s.Describe(), "the service exposes port(s) %s", strings.TrimSpace(res.Stdout))
		}
	}

	// EndpointSlices, not the deprecated Endpoints object: this is what the
	// current control plane and the current curriculum both use.
	args := nsArgs([]string{
		"get", "endpointslice",
		"-l", "kubernetes.io/service-name=" + s.Name,
		"-o", "jsonpath={range .items[*]}{range .endpoints[?(@.conditions.ready==true)]}{.addresses[0]}{\"\\n\"}{end}{end}",
	}, s.Namespace)
	res, err := env.KubectlRaw(ctx, args...)
	if err != nil {
		return broken(s.Describe(), err)
	}
	if res.ExitCode != 0 {
		return fail(s.Describe(), "cannot read endpoint slices: %s", firstLine(res.Stderr))
	}
	count := len(nonEmptyLines(res.Stdout))
	if count >= s.want() {
		return pass(s.Describe())
	}
	return fail(s.Describe(), "%d ready endpoint(s)", count)
}

// AuthCanI asks the API server itself whether a subject may do something.
// Grading RBAC any other way would be grading the YAML, not the permission.
type AuthCanI struct {
	User           string `yaml:"user"`
	ServiceAccount string `yaml:"serviceAccount"`
	Namespace      string `yaml:"namespace"`
	Verb           string `yaml:"verb"`
	Resource       string `yaml:"resource"`
	// Expect is the required answer.
	Expect *bool `yaml:"expect"`
}

func (a *AuthCanI) Type() string { return "authCanI" }

func (a *AuthCanI) Validate() error {
	if (a.User == "") == (a.ServiceAccount == "") {
		return fmt.Errorf("exactly one of user or serviceAccount is required")
	}
	if a.Verb == "" || a.Resource == "" {
		return fmt.Errorf("verb and resource are required")
	}
	if a.ServiceAccount != "" && !strings.Contains(a.ServiceAccount, "/") && a.Namespace == "" {
		return fmt.Errorf("serviceAccount needs a namespace, or the form namespace/name")
	}
	if a.Expect == nil {
		return fmt.Errorf("expect is required; say so explicitly rather than defaulting")
	}
	return nil
}

func (a *AuthCanI) subject() string {
	if a.User != "" {
		return a.User
	}
	if strings.Contains(a.ServiceAccount, "/") {
		parts := strings.SplitN(a.ServiceAccount, "/", 2)
		return "system:serviceaccount:" + parts[0] + ":" + parts[1]
	}
	return "system:serviceaccount:" + ns(a.Namespace) + ":" + a.ServiceAccount
}

func (a *AuthCanI) Describe() string {
	scope := "cluster-wide"
	if a.Namespace != "" {
		scope = "in " + a.Namespace
	}
	verb := "can"
	if !*a.Expect {
		verb = "cannot"
	}
	return fmt.Sprintf("%s %s %s %s %s", a.subject(), verb, a.Verb, a.Resource, scope)
}

func (a *AuthCanI) Check(ctx context.Context, env *environment.Manager) Result {
	args := []string{"auth", "can-i", a.Verb, a.Resource, "--as", a.subject()}
	args = nsArgs(args, a.Namespace)
	res, err := env.KubectlRaw(ctx, args...)
	if err != nil {
		return broken(a.Describe(), err)
	}
	answer := strings.TrimSpace(res.Stdout)
	// `auth can-i` says yes/no on stdout and mirrors it in the exit code.
	got := strings.HasPrefix(answer, "yes")
	if got == *a.Expect {
		return pass(a.Describe())
	}
	return fail(a.Describe(), "the API server says %q", answer)
}

// HTTPService checks that a Service answers over the network, from inside the
// cluster. This catches the failures that object inspection alone does not:
// wrong targetPort, a NetworkPolicy that drops the traffic, a broken CNI.
type HTTPService struct {
	Namespace string `yaml:"namespace"`
	Service   string `yaml:"service"`
	Port      int    `yaml:"port"`
	Path      string `yaml:"path"`
	// ExpectStatus defaults to 200.
	ExpectStatus int `yaml:"expectStatus"`
	// Image is the client image; it must contain wget or curl.
	Image string `yaml:"image"`
}

func (h *HTTPService) Type() string { return "httpService" }

func (h *HTTPService) Validate() error {
	if h.Service == "" {
		return fmt.Errorf("service is required")
	}
	if h.Port == 0 {
		return fmt.Errorf("port is required")
	}
	return nil
}

func (h *HTTPService) status() int {
	if h.ExpectStatus == 0 {
		return 200
	}
	return h.ExpectStatus
}

func (h *HTTPService) image() string {
	if h.Image == "" {
		return "busybox:1.36"
	}
	return h.Image
}

func (h *HTTPService) Describe() string {
	return fmt.Sprintf("http://%s.%s:%d%s answers %d from inside the cluster",
		h.Service, ns(h.Namespace), h.Port, h.Path, h.status())
}

func (h *HTTPService) Check(ctx context.Context, env *environment.Manager) Result {
	url := fmt.Sprintf("http://%s.%s.svc.cluster.local:%d%s", h.Service, ns(h.Namespace), h.Port, h.Path)
	name := fmt.Sprintf("dojo-probe-%d", nowNano()%100000)
	args := []string{
		"run", name,
		"--image=" + h.image(),
		"--restart=Never", "--rm", "-i", "--quiet",
		"--command", "--",
		"wget", "-q", "-O-", "-T", "10", url,
	}
	args = nsArgs(args, h.Namespace)
	res, err := env.KubectlRaw(ctx, args...)
	if err != nil {
		return broken(h.Describe(), err)
	}
	if res.ExitCode == 0 {
		return pass(h.Describe())
	}
	return fail(h.Describe(), "the request failed: %s", firstLine(res.Stderr+res.Stdout))
}

func containsField(s, want string) bool {
	for _, f := range strings.Fields(s) {
		if f == want {
			return true
		}
	}
	return false
}

func nonEmptyLines(s string) []string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if strings.TrimSpace(l) != "" {
			out = append(out, l)
		}
	}
	return out
}

func firstLine(s string) string {
	for _, l := range strings.Split(s, "\n") {
		l = strings.TrimSpace(l)
		if l != "" {
			if len(l) > 200 {
				return l[:200] + "..."
			}
			return l
		}
	}
	return "no output"
}
