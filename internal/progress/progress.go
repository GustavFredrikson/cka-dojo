// Package progress records attempts on disk, outside the repository, so that
// reinstalling or re-cloning never loses a study history.
package progress

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/gustavfredrikson/cka-dojo/internal/config"
)

// Attempt is one graded run of a lab.
type Attempt struct {
	Attempts int `json:"attempts"`
	Passes   int `json:"passes"`
	// CleanPasses are passes that used no hints and read no solution.
	CleanPasses int `json:"cleanPasses"`
	BestSeconds int `json:"bestSeconds,omitempty"`
	LastSeconds int `json:"lastSeconds,omitempty"`
	LastHints   int `json:"lastHints"`
	// LastPassHints belongs to the most recent successful attempt. It is
	// deliberately separate from LastHints: beginning or failing a later
	// attempt must not rewrite the evidence that established mastery.
	LastPassHints int       `json:"lastPassHints,omitempty"`
	LastSeed      int64     `json:"lastSeed,omitempty"`
	LastVariant   string    `json:"lastVariant,omitempty"`
	LastPassed    bool      `json:"lastPassed"`
	LastAt        time.Time `json:"lastAt"`
	Skills        []string  `json:"skills,omitempty"`
	Domains       []string  `json:"domains,omitempty"`
}

// Mastered is the readiness rule: passed at least twice, and the most recent
// pass needed no help.
//
// It deliberately ignores speed. Time pressure matters on the exam, but a
// learner who is fast and hint-dependent is not ready.
func (a *Attempt) Mastered() bool {
	return a.Passes >= 2 && a.LastPassHints == 0
}

// File is the whole progress record.
type File struct {
	Version int                 `json:"version"`
	Labs    map[string]*Attempt `json:"labs"`
}

func path() (string, error) {
	home, err := config.EnsureHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "progress.json"), nil
}

// Load reads progress.json, returning an empty record when absent.
func Load() (*File, error) {
	p, err := path()
	if err != nil {
		return nil, err
	}
	f := &File{Version: 1, Labs: map[string]*Attempt{}}
	data, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return f, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, f); err != nil {
		return nil, err
	}
	if f.Labs == nil {
		f.Labs = map[string]*Attempt{}
	}
	return f, nil
}

// Save writes progress.json atomically.
func (f *File) Save() error {
	p, err := path()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

// Archive moves the current progress file to a timestamped backup and writes
// a fresh empty record. It is the recoverable implementation behind
// `dojo progress reset`; study history is never deleted outright.
func Archive(now time.Time) (string, error) {
	p, err := path()
	if err != nil {
		return "", err
	}
	backup := ""
	if _, err := os.Stat(p); err == nil {
		base := filepath.Join(filepath.Dir(p), "progress-"+now.UTC().Format("20060102-150405")+".json")
		backup, err = availableBackupPath(base)
		if err != nil {
			return "", err
		}
		if err := os.Rename(p, backup); err != nil {
			return "", err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	fresh := &File{Version: 1, Labs: map[string]*Attempt{}}
	if err := fresh.Save(); err != nil {
		// Best effort: put the original back if creating the new record failed.
		if backup != "" {
			_ = os.Rename(backup, p)
		}
		return "", err
	}
	return backup, nil
}

func availableBackupPath(base string) (string, error) {
	if _, err := os.Stat(base); errors.Is(err, os.ErrNotExist) {
		return base, nil
	} else if err != nil {
		return "", err
	}
	for i := 1; ; i++ {
		candidate := fmt.Sprintf("%s.%d", base, i)
		if _, err := os.Stat(candidate); errors.Is(err, os.ErrNotExist) {
			return candidate, nil
		} else if err != nil {
			return "", err
		}
	}
}

// Get returns a lab's record, creating it if needed.
func (f *File) Get(labID string) *Attempt {
	if a, ok := f.Labs[labID]; ok {
		return a
	}
	a := &Attempt{}
	f.Labs[labID] = a
	return a
}

// StartAttempt records that a learner began a lab. An attempt is one
// `dojo start`, not one `dojo grade`: grading repeatedly while working on the
// same scenario is normal and should not inflate the history.
func (f *File) StartAttempt(labID string, seed int64, variant string, skills, domains []string) *Attempt {
	a := f.Get(labID)
	a.Attempts++
	a.LastSeed = seed
	a.LastVariant = variant
	a.LastPassed = false
	a.LastHints = 0
	a.LastAt = time.Now()
	a.Skills = skills
	a.Domains = domains
	return a
}

// RecordGrade folds a grading result into the current attempt. firstPass says
// whether this is the first successful grade of this attempt, which is what
// makes the pass counters meaningful.
func (f *File) RecordGrade(labID string, passed, firstPass bool, elapsed time.Duration, hints int, solutionRead bool) *Attempt {
	a := f.Get(labID)
	a.LastPassed = passed
	a.LastHints = hints
	a.LastAt = time.Now()
	secs := int(elapsed.Seconds())
	if secs > 0 {
		a.LastSeconds = secs
	}
	if passed && firstPass {
		a.Passes++
		a.LastPassHints = hints
		if !solutionRead && hints == 0 {
			a.CleanPasses++
		}
		if secs > 0 && (a.BestSeconds == 0 || secs < a.BestSeconds) {
			a.BestSeconds = secs
		}
	}
	return a
}

// SkillStat aggregates one skill across labs.
type SkillStat struct {
	Skill       string
	Attempts    int
	Passes      int
	CleanPasses int
	Labs        int
	MasteredLab int
	BestSeconds int
}

// Mastered reports whether every lab touching this skill is mastered.
func (s *SkillStat) Mastered() bool { return s.Labs > 0 && s.MasteredLab == s.Labs }

// BySkill aggregates the record over skills. labSkills maps every known lab
// to its skills, so skills with no attempts still show up as gaps.
func (f *File) BySkill(labSkills map[string][]string) []SkillStat {
	stats := map[string]*SkillStat{}
	get := func(skill string) *SkillStat {
		if s, ok := stats[skill]; ok {
			return s
		}
		s := &SkillStat{Skill: skill}
		stats[skill] = s
		return s
	}
	for labID, skills := range labSkills {
		att := f.Labs[labID]
		for _, skill := range skills {
			s := get(skill)
			s.Labs++
			if att == nil {
				continue
			}
			s.Attempts += att.Attempts
			s.Passes += att.Passes
			s.CleanPasses += att.CleanPasses
			if att.Mastered() {
				s.MasteredLab++
			}
			if att.BestSeconds > 0 && (s.BestSeconds == 0 || att.BestSeconds < s.BestSeconds) {
				s.BestSeconds = att.BestSeconds
			}
		}
	}
	out := make([]SkillStat, 0, len(stats))
	for _, s := range stats {
		out = append(out, *s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Skill < out[j].Skill })
	return out
}
