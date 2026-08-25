package fault

import (
	"strings"
	"testing"

	"github.com/gustavfredrikson/cka-dojo/internal/spec"
	"gopkg.in/yaml.v3"
)

func buildFrom(t *testing.T, doc string) (Fault, error) {
	t.Helper()
	var s spec.Spec
	if err := yaml.Unmarshal([]byte(doc), &s); err != nil {
		t.Fatalf("parse spec: %v", err)
	}
	return Build(s)
}

func TestUnknownFaultIsRejected(t *testing.T) {
	_, err := buildFrom(t, "type: setTheClusterOnFire\n")
	if err == nil || !strings.Contains(err.Error(), "unknown fault type") {
		t.Fatalf("err = %v, want an unknown-type complaint", err)
	}
}

// TestNodeExecDemandsAnUndo keeps the escape hatch from becoming a way to
// author labs that `dojo reset` cannot recover from.
func TestNodeExecDemandsAnUndo(t *testing.T) {
	_, err := buildFrom(t, `type: nodeExec
node: worker1
script: rm -f /etc/kubernetes/manifests/kube-apiserver.yaml
`)
	if err == nil || !strings.Contains(err.Error(), "undo is required") {
		t.Fatalf("err = %v, want a complaint about undo", err)
	}
}

func TestFileReplaceNeedsExactlyOneSource(t *testing.T) {
	if _, err := buildFrom(t, "type: fileReplace\nnode: worker1\npath: /etc/x\n"); err == nil {
		t.Error("fileReplace accepted no content at all")
	}
	if _, err := buildFrom(t, "type: fileReplace\nnode: worker1\npath: /etc/x\nsource: a\ncontent: b\n"); err == nil {
		t.Error("fileReplace accepted both source and content")
	}
	if _, err := buildFrom(t, "type: fileReplace\nnode: worker1\npath: /etc/x\ncontent: b\n"); err != nil {
		t.Errorf("fileReplace rejected valid inline content: %v", err)
	}
}

func TestFileReplaceBackupPathIsFlat(t *testing.T) {
	f := &FileReplace{Path: "/etc/containerd/config.toml"}
	want := backupDir + "/etc_containerd_config.toml"
	if got := f.backupPath(); got != want {
		t.Errorf("backupPath = %q, want %q", got, want)
	}
}

func TestNormalizeConvertsYAMLMaps(t *testing.T) {
	// yaml.v3 decodes nested mappings into map[string]any for typed targets
	// but tests the map[any]any path used by older documents too.
	in := map[any]any{"spec": map[any]any{"selector": map[any]any{"app": "web"}}}
	out, ok := normalize(in).(map[string]any)
	if !ok {
		t.Fatalf("normalize returned %T", normalize(in))
	}
	spec, ok := out["spec"].(map[string]any)
	if !ok {
		t.Fatalf("spec is %T", out["spec"])
	}
	if _, ok := spec["selector"].(map[string]any); !ok {
		t.Errorf("selector is %T, want map[string]any", spec["selector"])
	}
}

func TestSystemdStopDescribesDisable(t *testing.T) {
	f, err := buildFrom(t, "type: systemdStop\nnode: worker1\nunit: kubelet\ndisable: true\n")
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if !strings.Contains(f.Describe(), "disable") {
		t.Errorf("describe = %q, want it to mention disabling", f.Describe())
	}
}
