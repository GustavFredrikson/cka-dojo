// Package curriculum is the training content's table of contents: exam
// domains and their weights, the skills progress is tracked against, and the
// modules that hold lessons and labs.
package curriculum

import (
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/gustavfredrikson/cka-dojo/internal/content"
	"github.com/gustavfredrikson/cka-dojo/internal/lab"
	"gopkg.in/yaml.v3"
)

// Domain is a scored exam area.
type Domain struct {
	ID   string `yaml:"id"`
	Name string `yaml:"name"`
	// Weight is the published percentage of the exam.
	Weight int `yaml:"weight"`
}

// Skill is a trackable competency. Progress is reported per skill rather than
// per lab, because "5 of 12 labs done" says nothing about readiness.
type Skill struct {
	ID   string `yaml:"id"`
	Name string `yaml:"name"`
}

// Module groups a lesson with the labs that exercise it.
type Module struct {
	ID      string   `yaml:"id"`
	Name    string   `yaml:"name"`
	Summary string   `yaml:"summary"`
	Domains []string `yaml:"domains"`
	// LabOrder optionally fixes the teaching order; unlisted labs follow.
	LabOrder []string `yaml:"labs"`

	Dir  string     `yaml:"-"`
	Labs []*lab.Lab `yaml:"-"`
}

// HasLesson reports whether the module ships a lesson.
func (m *Module) HasLesson(src *content.Source) bool {
	return src.Exists(path.Join(m.Dir, "lesson.md"))
}

// Lesson returns the module's teaching text.
func (m *Module) Lesson(src *content.Source) (string, error) {
	b, err := src.Read(path.Join(m.Dir, "lesson.md"))
	if err != nil {
		return "", fmt.Errorf("module %s has no lesson yet", m.ID)
	}
	return string(b), nil
}

// Curriculum is one exam profile's content.
type Curriculum struct {
	SchemaVersion int    `yaml:"schemaVersion"`
	ID            string `yaml:"id"`
	Certification struct {
		Name string `yaml:"name"`
		// CurriculumRevision is the published competency revision this
		// content targets.
		CurriculumRevision string `yaml:"curriculumRevision"`
	} `yaml:"certification"`
	Kubernetes struct {
		// Minor is the exam's Kubernetes minor, which lags upstream.
		Minor string `yaml:"minor"`
	} `yaml:"kubernetes"`
	Domains   []Domain `yaml:"domains"`
	Skills    []Skill  `yaml:"skills"`
	ModuleIDs []string `yaml:"modules"`

	Dir     string    `yaml:"-"`
	Modules []*Module `yaml:"-"`
}

// Load reads curriculum/<id> with its modules and labs.
func Load(src *content.Source, id string) (*Curriculum, error) {
	dir := path.Join("curriculum", id)
	data, err := src.Read(path.Join(dir, "curriculum.yaml"))
	if err != nil {
		return nil, fmt.Errorf("curriculum %q not found", id)
	}
	c := &Curriculum{}
	dec := yaml.NewDecoder(strings.NewReader(string(data)))
	dec.KnownFields(true)
	if err := dec.Decode(c); err != nil {
		return nil, fmt.Errorf("%s/curriculum.yaml: %w", dir, err)
	}
	c.Dir = dir
	if c.ID == "" {
		c.ID = id
	}

	ids := c.ModuleIDs
	if len(ids) == 0 {
		// Fall back to discovery so a new module works before it is listed.
		entries, err := fs.ReadDir(src.FS, path.Join(dir, "modules"))
		if err != nil {
			return nil, fmt.Errorf("%s has no modules directory", dir)
		}
		for _, e := range entries {
			if e.IsDir() {
				ids = append(ids, e.Name())
			}
		}
		sort.Strings(ids)
	}

	for _, mid := range ids {
		m, err := loadModule(src, dir, mid)
		if err != nil {
			return nil, err
		}
		c.Modules = append(c.Modules, m)
	}
	return c, nil
}

func loadModule(src *content.Source, curDir, id string) (*Module, error) {
	dir := path.Join(curDir, "modules", id)
	m := &Module{ID: id, Dir: dir}
	if data, err := src.Read(path.Join(dir, "module.yaml")); err == nil {
		dec := yaml.NewDecoder(strings.NewReader(string(data)))
		dec.KnownFields(true)
		if err := dec.Decode(m); err != nil {
			return nil, fmt.Errorf("%s/module.yaml: %w", dir, err)
		}
		m.Dir = dir
		if m.ID == "" {
			m.ID = id
		}
	} else {
		return nil, fmt.Errorf("module %s has no module.yaml", id)
	}

	labsDir := path.Join(dir, "labs")
	entries, err := fs.ReadDir(src.FS, labsDir)
	if err != nil {
		// A module may exist as a lesson before it has labs.
		return m, nil
	}
	byID := map[string]*lab.Lab{}
	var found []*lab.Lab
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if !src.Exists(path.Join(labsDir, e.Name(), "lab.yaml")) {
			continue
		}
		l, err := lab.Load(src, path.Join(labsDir, e.Name()))
		if err != nil {
			return nil, err
		}
		l.Module = m.ID
		found = append(found, l)
		byID[l.ID] = l
	}
	lab.SortByID(found)

	// Explicit order first, then anything not listed.
	seen := map[string]bool{}
	for _, want := range m.LabOrder {
		if l, ok := byID[want]; ok {
			m.Labs = append(m.Labs, l)
			seen[l.ID] = true
		} else {
			return nil, fmt.Errorf("module %s lists lab %q, which does not exist", m.ID, want)
		}
	}
	for _, l := range found {
		if !seen[l.ID] {
			m.Labs = append(m.Labs, l)
		}
	}
	return m, nil
}

// Labs returns every lab in teaching order.
func (c *Curriculum) Labs() []*lab.Lab {
	var out []*lab.Lab
	for _, m := range c.Modules {
		out = append(out, m.Labs...)
	}
	return out
}

// LabCount is the total number of labs.
func (c *Curriculum) LabCount() int { return len(c.Labs()) }

// LabByID finds a lab by its id.
func (c *Curriculum) LabByID(id string) (*lab.Lab, *Module) {
	for _, m := range c.Modules {
		for _, l := range m.Labs {
			if l.ID == id {
				return l, m
			}
		}
	}
	return nil, nil
}

// ModuleByID finds a module by id or by name prefix, so `dojo learn services`
// finds 05-services.
func (c *Curriculum) ModuleByID(q string) *Module {
	for _, m := range c.Modules {
		if m.ID == q {
			return m
		}
	}
	q = strings.ToLower(q)
	for _, m := range c.Modules {
		if strings.Contains(strings.ToLower(m.ID), q) || strings.EqualFold(m.Name, q) {
			return m
		}
	}
	return nil
}

// DomainSet is the set of valid domain ids.
func (c *Curriculum) DomainSet() map[string]bool {
	out := map[string]bool{}
	for _, d := range c.Domains {
		out[d.ID] = true
	}
	return out
}

// SkillSet is the set of valid skill ids.
func (c *Curriculum) SkillSet() map[string]bool {
	out := map[string]bool{}
	for _, s := range c.Skills {
		out[s.ID] = true
	}
	return out
}

// SkillName resolves a skill id to its display name.
func (c *Curriculum) SkillName(id string) string {
	for _, s := range c.Skills {
		if s.ID == id {
			return s.Name
		}
	}
	return id
}

// Validate checks the curriculum's own consistency.
func (c *Curriculum) Validate() []error {
	var errs []error
	add := func(format string, a ...any) { errs = append(errs, fmt.Errorf(format, a...)) }

	if c.SchemaVersion != 1 {
		add("unsupported schemaVersion %d (want 1)", c.SchemaVersion)
	}
	if c.Kubernetes.Minor == "" {
		add("kubernetes.minor is required")
	}
	total := 0
	seenDomain := map[string]bool{}
	for _, d := range c.Domains {
		if seenDomain[d.ID] {
			add("duplicate domain %q", d.ID)
		}
		seenDomain[d.ID] = true
		total += d.Weight
	}
	if len(c.Domains) > 0 && total != 100 {
		add("domain weights sum to %d, not 100", total)
	}
	seenSkill := map[string]bool{}
	for _, s := range c.Skills {
		if seenSkill[s.ID] {
			add("duplicate skill %q", s.ID)
		}
		seenSkill[s.ID] = true
	}
	seenLab := map[string]string{}
	for _, m := range c.Modules {
		for _, d := range m.Domains {
			if !seenDomain[d] {
				add("module %s: unknown domain %q", m.ID, d)
			}
		}
		for _, l := range m.Labs {
			if prev, ok := seenLab[l.ID]; ok {
				add("duplicate lab id %q (in %s and %s)", l.ID, prev, m.ID)
			}
			seenLab[l.ID] = m.ID
		}
	}
	return errs
}
