// Package lab is the declarative unit of training: what to build, what to
// break, and what state counts as fixed.
package lab

import (
	"fmt"
	"hash/fnv"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/gustavfredrikson/cka-dojo/internal/content"
	"github.com/gustavfredrikson/cka-dojo/internal/fault"
	"github.com/gustavfredrikson/cka-dojo/internal/grader"
	"github.com/gustavfredrikson/cka-dojo/internal/spec"
	"gopkg.in/yaml.v3"
)

// Setup is what a lab builds before the learner sees it.
type Setup struct {
	// Apply lists manifests in the lab directory, applied in order.
	Apply []string `yaml:"apply"`
	// Faults are what makes the scenario a scenario.
	Faults []spec.Spec `yaml:"faults"`
	// WaitReady, when set, waits for these workloads to settle before the
	// faults land, so the learner does not race the setup.
	WaitReady []WaitTarget `yaml:"waitReady"`
}

// WaitTarget is a workload to wait for during setup.
type WaitTarget struct {
	Namespace string `yaml:"namespace"`
	Kind      string `yaml:"kind"`
	Name      string `yaml:"name"`
}

// Grading is the set of requirements a solution must satisfy.
type Grading struct {
	// All must every pass.
	All []spec.Spec `yaml:"all"`
	// Any needs at least one pass, for genuinely equivalent solutions.
	Any []spec.Spec `yaml:"any"`
}

func (g Grading) empty() bool { return len(g.All) == 0 && len(g.Any) == 0 }

// Answer is an accepted response to an observation checkpoint. Answers are
// deliberately small and factual; cluster state remains the source of truth
// for configuration and repair checkpoints.
type Answer struct {
	Accepted        []string `yaml:"accepted"`
	CaseInsensitive bool     `yaml:"caseInsensitive"`
}

// Matches compares a learner answer with the accepted forms.
func (a *Answer) Matches(got string) bool {
	got = strings.TrimSpace(got)
	for _, want := range a.Accepted {
		want = strings.TrimSpace(want)
		if a.CaseInsensitive {
			if strings.EqualFold(got, want) {
				return true
			}
		} else if got == want {
			return true
		}
	}
	return false
}

// Checkpoint is one step in an interactive exercise. It may ask for a short
// observation, validate cluster state, or require both.
type Checkpoint struct {
	ID      string  `yaml:"id"`
	Task    string  `yaml:"task"`
	Answer  *Answer `yaml:"answer"`
	Grading Grading `yaml:"grading"`
}

// Variant is one randomised form of a lab. Variants exist so that repeating a
// lab trains diagnosis rather than recall.
type Variant struct {
	ID     string      `yaml:"id"`
	Apply  []string    `yaml:"apply"`
	Faults []spec.Spec `yaml:"faults"`
	// Grading replaces the lab's grading when set.
	Grading *Grading `yaml:"grading"`
	// Hints replace the lab's hints when set.
	Hints []string `yaml:"hints"`
}

// Variants declares the alternatives a lab can present.
type Variants struct {
	Options []Variant `yaml:"options"`
}

// EnvSpec selects the environment a lab needs.
type EnvSpec struct {
	Profile string `yaml:"profile"`
}

// ExamSpec controls mock-exam eligibility.
type ExamSpec struct {
	Eligible bool `yaml:"eligible"`
}

// Lab is one exercise, loaded from labs/<id>/lab.yaml.
type Lab struct {
	SchemaVersion int           `yaml:"schemaVersion"`
	ID            string        `yaml:"id"`
	Title         string        `yaml:"title"`
	Domains       []string      `yaml:"domain"`
	Skills        []string      `yaml:"skills"`
	LearningStage LearningStage `yaml:"learningStage"`
	Difficulty    int           `yaml:"difficulty"`
	TargetMinutes int           `yaml:"targetMinutes"`
	Environment   EnvSpec       `yaml:"environment"`
	Setup         Setup         `yaml:"setup"`
	Variants      *Variants
	Grading       Grading      `yaml:"grading"`
	Hints         []string     `yaml:"hints"`
	Checkpoints   []Checkpoint `yaml:"checkpoints"`
	// Prerequisites need one pass. MasteryPrerequisites need the full mastery
	// rule and are intended for challenge/exam gates. Both can be bypassed
	// explicitly; the path is guidance, not a prison.
	Prerequisites        []string `yaml:"prerequisites"`
	MasteryPrerequisites []string `yaml:"masteryPrerequisites"`
	Exam                 ExamSpec `yaml:"exam"`
	// Conflicts names resources a lab monopolises, so the exam generator can
	// avoid pairing two labs that would fight.
	Conflicts []string `yaml:"conflicts"`

	// Dir is the lab's path inside the content root.
	Dir string `yaml:"-"`
	// Module is the module directory this lab belongs to.
	Module string `yaml:"-"`
	// Files is the lab directory as a filesystem.
	Files fs.FS `yaml:"-"`
}

// Profile is the environment profile this lab runs on.
func (l *Lab) Profile(def string) string {
	if l.Environment.Profile != "" {
		return l.Environment.Profile
	}
	return def
}

// Task returns the rendered task text shown to the learner.
func (l *Lab) Task() (string, error) {
	b, err := fs.ReadFile(l.Files, "task.md")
	if err != nil {
		return "", fmt.Errorf("lab %s has no task.md", l.ID)
	}
	return string(b), nil
}

// Solution returns the worked solution, which never leaves the host.
func (l *Lab) Solution() (string, error) {
	b, err := fs.ReadFile(l.Files, "solution.md")
	if err != nil {
		return "", fmt.Errorf("lab %s has no solution.md", l.ID)
	}
	return string(b), nil
}

// PickVariant chooses a variant deterministically from a seed, so a scenario
// can be reproduced exactly with `dojo start <lab> --seed N`.
func (l *Lab) PickVariant(seed int64) *Variant {
	if l.Variants == nil || len(l.Variants.Options) == 0 {
		return nil
	}
	h := fnv.New64a()
	fmt.Fprintf(h, "%s/%d", l.ID, seed)
	idx := int(h.Sum64() % uint64(len(l.Variants.Options)))
	return &l.Variants.Options[idx]
}

// VariantByID finds a named variant.
func (l *Lab) VariantByID(id string) *Variant {
	if l.Variants == nil {
		return nil
	}
	for i := range l.Variants.Options {
		if l.Variants.Options[i].ID == id {
			return &l.Variants.Options[i]
		}
	}
	return nil
}

// Plan is the concrete shape of one attempt: base setup merged with the
// chosen variant.
type Plan struct {
	Lab         *Lab
	Variant     *Variant
	Manifests   []string
	Faults      []fault.Fault
	Checks      []grader.Checker
	AnyChecks   []grader.Checker
	Hints       []string
	Checkpoints []CheckpointPlan
}

// CheckpointPlan is a decoded, runnable checkpoint.
type CheckpointPlan struct {
	Definition Checkpoint
	Checks     []grader.Checker
	AnyChecks  []grader.Checker
}

// Build resolves a lab plus a variant into runnable faults and checks.
func (l *Lab) Build(v *Variant) (*Plan, error) {
	p := &Plan{Lab: l, Variant: v, Hints: l.Hints}
	p.Manifests = append(p.Manifests, l.Setup.Apply...)
	specs := append([]spec.Spec{}, l.Setup.Faults...)
	grading := l.Grading

	if v != nil {
		p.Manifests = append(p.Manifests, v.Apply...)
		specs = append(specs, v.Faults...)
		if v.Grading != nil {
			grading = *v.Grading
		}
		if len(v.Hints) > 0 {
			p.Hints = v.Hints
		}
	}

	var err error
	if p.Faults, err = fault.BuildAll(specs); err != nil {
		return nil, fmt.Errorf("lab %s: %w", l.ID, err)
	}
	if p.Checks, err = grader.BuildAll(grading.All); err != nil {
		return nil, fmt.Errorf("lab %s: %w", l.ID, err)
	}
	if p.AnyChecks, err = grader.BuildAll(grading.Any); err != nil {
		return nil, fmt.Errorf("lab %s: %w", l.ID, err)
	}
	for _, checkpoint := range l.Checkpoints {
		cp := CheckpointPlan{Definition: checkpoint}
		if cp.Checks, err = grader.BuildAll(checkpoint.Grading.All); err != nil {
			return nil, fmt.Errorf("lab %s checkpoint %s: %w", l.ID, checkpoint.ID, err)
		}
		if cp.AnyChecks, err = grader.BuildAll(checkpoint.Grading.Any); err != nil {
			return nil, fmt.Errorf("lab %s checkpoint %s: %w", l.ID, checkpoint.ID, err)
		}
		p.Checkpoints = append(p.Checkpoints, cp)
	}
	return p, nil
}

// Validate checks everything that can be checked without a cluster. This is
// what makes adding content safe.
func (l *Lab) Validate(knownDomains, knownSkills map[string]bool, knownProfiles map[string]bool) []error {
	var errs []error
	add := func(format string, a ...any) { errs = append(errs, fmt.Errorf(format, a...)) }

	if l.SchemaVersion != 1 {
		add("unsupported schemaVersion %d (want 1)", l.SchemaVersion)
	}
	if l.ID == "" {
		add("id is required")
	}
	if l.Title == "" {
		add("title is required")
	}
	if err := l.LearningStage.Validate(); err != nil {
		add("%v", err)
	}
	if l.Difficulty < 1 || l.Difficulty > 5 {
		add("difficulty must be 1-5, got %d", l.Difficulty)
	}
	if l.TargetMinutes <= 0 {
		add("targetMinutes must be positive")
	}
	if len(l.Domains) == 0 {
		add("at least one domain is required")
	}
	for _, d := range l.Domains {
		if knownDomains != nil && !knownDomains[d] {
			add("unknown domain %q", d)
		}
	}
	if len(l.Skills) == 0 {
		add("at least one skill is required; progress is tracked by skill, not by lab")
	}
	for _, s := range l.Skills {
		if knownSkills != nil && !knownSkills[s] {
			add("unknown skill %q; add it to curriculum.yaml first", s)
		}
	}
	if p := l.Environment.Profile; p != "" && knownProfiles != nil && !knownProfiles[p] {
		add("unknown environment profile %q", p)
	}
	if _, err := l.Task(); err != nil {
		add("%v", err)
	}
	if _, err := l.Solution(); err != nil {
		add("%v", err)
	}
	if l.LearningStage.NeedsHints() && len(l.Hints) == 0 && (l.Variants == nil || len(l.Variants.Options) == 0) {
		add("no hints; a lab without progressive hints cannot be run in guided mode")
	}
	checkpointIDs := map[string]bool{}
	for i, checkpoint := range l.Checkpoints {
		if checkpoint.ID == "" {
			add("checkpoint %d has no id", i+1)
		}
		if checkpointIDs[checkpoint.ID] {
			add("duplicate checkpoint id %q", checkpoint.ID)
		}
		checkpointIDs[checkpoint.ID] = true
		if strings.TrimSpace(checkpoint.Task) == "" {
			add("checkpoint %q has no task", checkpoint.ID)
		}
		if checkpoint.Answer == nil && checkpoint.Grading.empty() {
			add("checkpoint %q has neither an answer nor grading", checkpoint.ID)
		}
		if checkpoint.Answer != nil && len(checkpoint.Answer.Accepted) == 0 {
			add("checkpoint %q answer has no accepted values", checkpoint.ID)
		}
	}

	// Every variant, and the base lab, must build and must grade something.
	type candidate struct {
		name string
		v    *Variant
	}
	cands := []candidate{{name: "base", v: nil}}
	if l.Variants != nil {
		seen := map[string]bool{}
		for i := range l.Variants.Options {
			v := &l.Variants.Options[i]
			if v.ID == "" {
				add("variant %d has no id", i)
				continue
			}
			if seen[v.ID] {
				add("duplicate variant id %q", v.ID)
			}
			seen[v.ID] = true
			cands = append(cands, candidate{name: "variant " + v.ID, v: v})
		}
	}
	for _, c := range cands {
		if c.v == nil && l.Variants != nil && len(l.Variants.Options) > 0 {
			// A lab with variants need not grade anything at the base level.
			if l.Grading.empty() {
				continue
			}
		}
		grading := l.Grading
		if c.v != nil && c.v.Grading != nil {
			grading = *c.v.Grading
		}
		if grading.empty() {
			add("%s has no grading", c.name)
		}
		if _, err := l.Build(c.v); err != nil {
			add("%s: %v", c.name, err)
		}
	}

	// Manifests must exist.
	manifests := append([]string{}, l.Setup.Apply...)
	if l.Variants != nil {
		for _, v := range l.Variants.Options {
			manifests = append(manifests, v.Apply...)
		}
	}
	for _, m := range manifests {
		if _, err := fs.Stat(l.Files, m); err != nil {
			add("manifest %s not found in the lab directory", m)
		}
	}
	return errs
}

// Load reads one lab from the content root.
func Load(src *content.Source, dir string) (*Lab, error) {
	data, err := src.Read(path.Join(dir, "lab.yaml"))
	if err != nil {
		return nil, fmt.Errorf("%s: no lab.yaml", dir)
	}
	l := &Lab{}
	dec := yaml.NewDecoder(strings.NewReader(string(data)))
	dec.KnownFields(true)
	if err := dec.Decode(l); err != nil {
		return nil, fmt.Errorf("%s/lab.yaml: %w", dir, err)
	}
	sub, err := fs.Sub(src.FS, dir)
	if err != nil {
		return nil, err
	}
	l.Dir = dir
	l.Files = sub
	if l.ID == "" {
		l.ID = path.Base(dir)
	}
	parts := strings.Split(dir, "/")
	for i, p := range parts {
		if p == "modules" && i+1 < len(parts) {
			l.Module = parts[i+1]
		}
	}
	return l, nil
}

// SortByID orders labs deterministically.
func SortByID(labs []*Lab) {
	sort.Slice(labs, func(i, j int) bool { return labs[i].ID < labs[j].ID })
}
