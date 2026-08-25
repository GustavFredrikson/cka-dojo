package spec

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestSpecCapturesTypeAndBody(t *testing.T) {
	var s Spec
	if err := yaml.Unmarshal([]byte("type: systemdStop\nnode: worker1\nunit: kubelet\n"), &s); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if s.Type != "systemdStop" {
		t.Errorf("type = %q", s.Type)
	}
	var body struct {
		Node string `yaml:"node"`
		Unit string `yaml:"unit"`
	}
	if err := s.Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Node != "worker1" || body.Unit != "kubelet" {
		t.Errorf("body = %+v", body)
	}
}

func TestSpecRequiresType(t *testing.T) {
	var s Spec
	err := yaml.Unmarshal([]byte("node: worker1\n"), &s)
	if err == nil || !strings.Contains(err.Error(), "missing `type`") {
		t.Fatalf("err = %v, want a complaint about the missing type", err)
	}
}
