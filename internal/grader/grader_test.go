package grader

import (
	"strings"
	"testing"

	"github.com/gustavfredrikson/cka-dojo/internal/spec"
	"gopkg.in/yaml.v3"
)

func buildFrom(t *testing.T, doc string) (Checker, error) {
	t.Helper()
	var s spec.Spec
	if err := yaml.Unmarshal([]byte(doc), &s); err != nil {
		t.Fatalf("parse spec: %v", err)
	}
	return Build(s)
}

func TestUnknownGraderIsRejected(t *testing.T) {
	_, err := buildFrom(t, "type: doesNotExist\n")
	if err == nil || !strings.Contains(err.Error(), "unknown grader type") {
		t.Fatalf("err = %v, want an unknown-type complaint", err)
	}
}

// TestAuthCanIRequiresExplicitExpectation matters because a defaulted
// expectation silently turns a negative permission check into a positive one.
func TestAuthCanIRequiresExplicitExpectation(t *testing.T) {
	_, err := buildFrom(t, `type: authCanI
user: alice
namespace: backend
verb: get
resource: pods
`)
	if err == nil || !strings.Contains(err.Error(), "expect is required") {
		t.Fatalf("err = %v, want a complaint about expect", err)
	}
}

func TestAuthCanISubject(t *testing.T) {
	yes := true
	for _, tc := range []struct {
		name string
		in   AuthCanI
		want string
	}{
		{"user", AuthCanI{User: "alice", Verb: "get", Resource: "pods", Expect: &yes}, "alice"},
		{"sa with namespace", AuthCanI{ServiceAccount: "builder", Namespace: "ci", Verb: "get", Resource: "pods", Expect: &yes},
			"system:serviceaccount:ci:builder"},
		{"qualified sa", AuthCanI{ServiceAccount: "ci/builder", Verb: "get", Resource: "pods", Expect: &yes},
			"system:serviceaccount:ci:builder"},
	} {
		if got := tc.in.subject(); got != tc.want {
			t.Errorf("%s: subject = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestMatchSemantics(t *testing.T) {
	eq := "v1.35.8"
	for _, tc := range []struct {
		name string
		m    Match
		got  string
		want bool
	}{
		{"equals hit", Match{Equals: &eq}, "v1.35.8", true},
		{"equals trims whitespace", Match{Equals: &eq}, "  v1.35.8\n", true},
		{"equals miss", Match{Equals: &eq}, "v1.34.0", false},
		{"contains", Match{Contains: "Ready"}, "node1 Ready", true},
		{"matches", Match{Matches: `^v1\.35\.\d+$`}, "v1.35.2", true},
		{"matches miss", Match{Matches: `^v1\.35\.\d+$`}, "v1.36.0", false},
		{"empty means non-empty", Match{}, "anything", true},
		{"empty rejects blank", Match{}, "   ", false},
	} {
		if got := tc.m.ok(tc.got); got != tc.want {
			t.Errorf("%s: ok(%q) = %v, want %v", tc.name, tc.got, got, tc.want)
		}
	}
}

func TestMatchRejectsBadRegexp(t *testing.T) {
	m := Match{Matches: "([unclosed"}
	if err := m.validate(); err == nil {
		t.Fatal("a broken regexp passed validation")
	}
}

func TestNodeServiceDefaultsToActive(t *testing.T) {
	c, err := buildFrom(t, `type: nodeService
node: worker1
unit: kubelet
`)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if !strings.Contains(c.Describe(), "is active") {
		t.Errorf("describe = %q, want the default active state", c.Describe())
	}
}

func TestPodScheduledRejectsNodeWhenExpectingPending(t *testing.T) {
	_, err := buildFrom(t, `type: podScheduled
name: web
scheduled: false
node: worker2
`)
	if err == nil || !strings.Contains(err.Error(), "node cannot be required") {
		t.Fatalf("err = %v, want incompatible expectation rejected", err)
	}
}

func TestEveryRegisteredGraderValidatesItsInput(t *testing.T) {
	// An empty spec must never validate: a grader that accepts nothing would
	// silently pass a lab.
	for _, name := range Types() {
		var s spec.Spec
		if err := yaml.Unmarshal([]byte("type: "+name+"\n"), &s); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if _, err := Build(s); err == nil {
			t.Errorf("grader %q accepted an empty definition", name)
		}
	}
}

// TestHTTPServiceFlagsPrecedeTheSeparator guards a bug that sent `-n <ns>` to
// wget instead of kubectl, so the probe silently ran in the wrong namespace.
func TestHTTPServiceFlagsPrecedeTheSeparator(t *testing.T) {
	h := &HTTPService{Namespace: "shop", Service: "web", Port: 80}
	args := h.probeArgs("http://web.shop.svc.cluster.local:80")

	sep := -1
	for i, a := range args {
		if a == "--" {
			sep = i
			break
		}
	}
	if sep < 0 {
		t.Fatalf("no `--` separator in %v", args)
	}
	nsFlag := -1
	for i, a := range args {
		if a == "-n" {
			nsFlag = i
			break
		}
	}
	if nsFlag < 0 {
		t.Fatalf("no namespace flag in %v", args)
	}
	if nsFlag > sep {
		t.Errorf("namespace flag lands after `--`, so wget receives it: %v", args)
	}
	if args[sep+1] != "wget" {
		t.Errorf("first argument after `--` is %q, want wget", args[sep+1])
	}
}

// TestSplitFieldsKeepsPositionsWhenAValueIsAbsent pins the defect that a
// whitespace split hid: kubectl renders a missing status field as the empty
// string, so `{.status.availableReplicas}|{.status.updatedReplicas}` on a
// Deployment with nothing available yet returns "|2". Splitting that on
// whitespace collapses it to a single field and reports 2 replicas *available*
// when in truth 2 are merely updated and none are available -- which sends a
// learner looking at a rollout instead of at the scheduler.
func TestSplitFieldsKeepsPositionsWhenAValueIsAbsent(t *testing.T) {
	for _, tc := range []struct {
		name                       string
		out                        string
		wantAvailable, wantUpdated int
	}{
		{"both present", "2|2", 2, 2},
		{"available absent", "|2", 0, 2},
		{"updated absent", "2|", 2, 0},
		{"both absent", "|", 0, 0},
		{"empty output", "", 0, 0},
		{"surrounding whitespace", " 3 | 3 \n", 3, 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fields := splitFields(tc.out, 2)
			if len(fields) != 2 {
				t.Fatalf("splitFields(%q) = %#v, want two fields", tc.out, fields)
			}
			available, updated := parseCount(fields[0]), parseCount(fields[1])
			if available != tc.wantAvailable || updated != tc.wantUpdated {
				t.Errorf("splitFields(%q) = %d available, %d updated; want %d, %d",
					tc.out, available, updated, tc.wantAvailable, tc.wantUpdated)
			}
		})
	}
}

func TestParseCountTreatsNonNumericAsZero(t *testing.T) {
	for _, in := range []string{"", "  ", "<none>", "null"} {
		if got := parseCount(in); got != 0 {
			t.Errorf("parseCount(%q) = %d, want 0", in, got)
		}
	}
}
