package environment

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/gustavfredrikson/cka-dojo/internal/provider"
)

// stagingDir is where the engine drops manifests inside the control plane.
// It is on tmpfs and files are deleted immediately after use, so lab answers
// do not sit around where a learner can read them.
const stagingDir = "/run/dojo"

// shellQuote makes a string safe as a single POSIX shell word.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// KubectlResult carries both streams so graders can explain a failure.
type KubectlResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// KubectlRaw runs kubectl on the control plane with the admin kubeconfig and
// returns the result without treating a non-zero exit as a Go error. Graders
// need to distinguish "command failed" from "answer is no".
func (m *Manager) KubectlRaw(ctx context.Context, args ...string) (KubectlResult, error) {
	cp := m.Profile.ControlPlane()
	if cp == nil {
		return KubectlResult{}, fmt.Errorf("profile %s has no control plane", m.Profile.ID)
	}
	quoted := make([]string, len(args))
	for i, a := range args {
		quoted[i] = shellQuote(a)
	}
	script := fmt.Sprintf("export KUBECONFIG=%s\nkubectl %s\n", AdminKubeconfig, strings.Join(quoted, " "))
	res, err := m.Exec(ctx, cp.Name, script, "root")
	out := KubectlResult{Stdout: res.Stdout, Stderr: res.Stderr, ExitCode: res.ExitCode}
	var exitErr *provider.ExitError
	if err != nil {
		if asExit(err, &exitErr) {
			return out, nil
		}
		return out, err
	}
	return out, nil
}

func asExit(err error, target **provider.ExitError) bool {
	e, ok := err.(*provider.ExitError)
	if ok {
		*target = e
	}
	return ok
}

// Kubectl runs kubectl and fails on a non-zero exit.
func (m *Manager) Kubectl(ctx context.Context, args ...string) (string, error) {
	res, err := m.KubectlRaw(ctx, args...)
	if err != nil {
		return "", err
	}
	if res.ExitCode != 0 {
		msg := strings.TrimSpace(res.Stderr)
		if msg == "" {
			msg = strings.TrimSpace(res.Stdout)
		}
		return res.Stdout, fmt.Errorf("kubectl %s: %s", strings.Join(args, " "), msg)
	}
	return res.Stdout, nil
}

// stage writes a manifest into the control plane and returns its path plus a
// cleanup function.
func (m *Manager) stage(ctx context.Context, data []byte, suffix string) (string, func(), error) {
	cp := m.Profile.ControlPlane()
	if cp == nil {
		return "", func() {}, fmt.Errorf("profile %s has no control plane", m.Profile.ID)
	}
	var buf [8]byte
	rand.Read(buf[:])
	remote := fmt.Sprintf("%s/%s%s", stagingDir, hex.EncodeToString(buf[:]), suffix)
	vm := m.VMName(cp.Name)
	if err := m.Prov.WriteFile(ctx, vm, remote, data, 0o600); err != nil {
		return "", func() {}, err
	}
	cleanup := func() {
		m.Run(context.WithoutCancel(ctx), cp.Name, fmt.Sprintf("rm -f %s\n", shellQuote(remote)))
	}
	return remote, cleanup, nil
}

// Apply applies a manifest from bytes.
func (m *Manager) Apply(ctx context.Context, data []byte) error {
	remote, cleanup, err := m.stage(ctx, data, ".yaml")
	if err != nil {
		return err
	}
	defer cleanup()
	_, err = m.Kubectl(ctx, "apply", "-f", remote)
	return err
}

// DeleteManifest deletes the objects in a manifest, ignoring absent ones.
func (m *Manager) DeleteManifest(ctx context.Context, data []byte) error {
	remote, cleanup, err := m.stage(ctx, data, ".yaml")
	if err != nil {
		return err
	}
	defer cleanup()
	_, err = m.Kubectl(ctx, "delete", "-f", remote, "--ignore-not-found", "--wait=false")
	return err
}

// JSONPath returns a single jsonpath expression evaluated against an object.
func (m *Manager) JSONPath(ctx context.Context, kind, name, namespace, expr string) (string, error) {
	args := []string{"get", kind, name, "-o", "jsonpath=" + expr}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}
	out, err := m.Kubectl(ctx, args...)
	return strings.TrimSpace(out), err
}
