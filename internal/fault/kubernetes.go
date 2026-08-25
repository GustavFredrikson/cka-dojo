package fault

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"

	"github.com/gustavfredrikson/cka-dojo/internal/environment"
)

func init() {
	register(func() Fault { return &Apply{} })
	register(func() Fault { return &Patch{} })
	register(func() Fault { return &Delete{} })
	register(func() Fault { return &Scale{} })
}

// Apply applies a manifest from the lab directory. Used for the parts of a
// scenario that are broken from birth rather than broken afterwards.
type Apply struct {
	Manifest string `yaml:"manifest"`
}

func (a *Apply) Type() string { return "kubernetesApply" }

func (a *Apply) Validate() error {
	if a.Manifest == "" {
		return fmt.Errorf("manifest is required")
	}
	return nil
}

func (a *Apply) Describe() string { return "apply " + a.Manifest }

func (a *Apply) Inject(ctx context.Context, env *environment.Manager, lc *Context) error {
	data, err := fs.ReadFile(lc.Files, a.Manifest)
	if err != nil {
		return fmt.Errorf("read %s: %w", a.Manifest, err)
	}
	return env.Apply(ctx, data)
}

func (a *Apply) Repair(ctx context.Context, env *environment.Manager, lc *Context) error {
	data, err := fs.ReadFile(lc.Files, a.Manifest)
	if err != nil {
		return nil
	}
	return env.DeleteManifest(ctx, data)
}

// Patch mutates a live object. This is the workhorse for "someone changed one
// field and broke it" scenarios.
type Patch struct {
	Resource  string `yaml:"resource"`
	Namespace string `yaml:"namespace"`
	// Patch is a strategic-merge patch written inline in the lab file.
	Patch map[string]any `yaml:"patch"`
	// PatchType overrides the default strategic merge, e.g. json or merge.
	PatchType string `yaml:"patchType"`
}

func (p *Patch) Type() string { return "kubernetesPatch" }

func (p *Patch) Validate() error {
	if p.Resource == "" {
		return fmt.Errorf("resource is required, e.g. service/web")
	}
	if len(p.Patch) == 0 {
		return fmt.Errorf("patch is required")
	}
	switch p.PatchType {
	case "", "strategic", "merge", "json":
	default:
		return fmt.Errorf("unknown patchType %q", p.PatchType)
	}
	return nil
}

func (p *Patch) Describe() string { return "patch " + p.Resource }

func (p *Patch) Inject(ctx context.Context, env *environment.Manager, lc *Context) error {
	body, err := json.Marshal(normalize(p.Patch))
	if err != nil {
		return err
	}
	args := []string{"patch", p.Resource, "-p", string(body)}
	if p.PatchType != "" {
		args = append(args, "--type", p.PatchType)
	}
	if p.Namespace != "" {
		args = append(args, "-n", p.Namespace)
	}
	_, err = env.Kubectl(ctx, args...)
	return err
}

// Repair is a no-op: reset re-creates the baseline objects outright, which is
// both simpler and more reliable than computing an inverse patch.
func (p *Patch) Repair(ctx context.Context, env *environment.Manager, lc *Context) error { return nil }

// normalize converts the map[any]any that YAML can produce into shapes
// encoding/json accepts.
func normalize(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			out[k] = normalize(val)
		}
		return out
	case map[any]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			out[fmt.Sprint(k)] = normalize(val)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, val := range t {
			out[i] = normalize(val)
		}
		return out
	default:
		return v
	}
}

// Delete removes a live object.
type Delete struct {
	Resource  string `yaml:"resource"`
	Namespace string `yaml:"namespace"`
}

func (d *Delete) Type() string { return "kubernetesDelete" }

func (d *Delete) Validate() error {
	if d.Resource == "" {
		return fmt.Errorf("resource is required, e.g. deployment/web")
	}
	return nil
}

func (d *Delete) Describe() string { return "delete " + d.Resource }

func (d *Delete) Inject(ctx context.Context, env *environment.Manager, lc *Context) error {
	args := []string{"delete", d.Resource, "--ignore-not-found"}
	if d.Namespace != "" {
		args = append(args, "-n", d.Namespace)
	}
	_, err := env.Kubectl(ctx, args...)
	return err
}

func (d *Delete) Repair(ctx context.Context, env *environment.Manager, lc *Context) error { return nil }

// Scale changes a workload's replica count.
type Scale struct {
	Resource  string `yaml:"resource"`
	Namespace string `yaml:"namespace"`
	Replicas  *int   `yaml:"replicas"`
}

func (s *Scale) Type() string { return "kubernetesScale" }

func (s *Scale) Validate() error {
	if s.Resource == "" {
		return fmt.Errorf("resource is required, e.g. deployment/web")
	}
	if s.Replicas == nil {
		return fmt.Errorf("replicas is required")
	}
	return nil
}

func (s *Scale) Describe() string { return fmt.Sprintf("scale %s to %d", s.Resource, *s.Replicas) }

func (s *Scale) Inject(ctx context.Context, env *environment.Manager, lc *Context) error {
	args := []string{"scale", s.Resource, fmt.Sprintf("--replicas=%d", *s.Replicas)}
	if s.Namespace != "" {
		args = append(args, "-n", s.Namespace)
	}
	_, err := env.Kubectl(ctx, args...)
	return err
}

func (s *Scale) Repair(ctx context.Context, env *environment.Manager, lc *Context) error { return nil }
