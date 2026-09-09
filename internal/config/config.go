// Package config owns everything the engine persists on the host: the user
// config file, the single-active-lab state, and the filesystem lock that keeps
// two dojo processes from fighting over one cluster.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// DefaultProfile is the environment profile used when nothing else is set.
const DefaultProfile = "standard"

// Config is ~/.cka-dojo/config.yaml. Every field is optional.
type Config struct {
	// ContentDir points the engine at a curriculum/environments checkout
	// instead of the content embedded in the binary.
	ContentDir string `yaml:"contentDir,omitempty"`
	// Curriculum selects a directory under curriculum/.
	Curriculum string `yaml:"curriculum,omitempty"`
	// Profile is the default environment profile.
	Profile string `yaml:"profile,omitempty"`
	// Verbose mirrors --verbose for people who always want it.
	Verbose bool `yaml:"verbose,omitempty"`
}

// Mode is how a lab attempt is being run.
type Mode string

const (
	ModeGuided   Mode = "guided"
	ModePractice Mode = "practice"
	ModeExam     Mode = "exam"
)

// State is ~/.cka-dojo/state.json: the single active lab or exam.
type State struct {
	ActiveLab    string    `json:"activeLab,omitempty"`
	Profile      string    `json:"profile,omitempty"`
	Mode         Mode      `json:"mode,omitempty"`
	Variant      string    `json:"variant,omitempty"`
	Seed         int64     `json:"seed,omitempty"`
	StartedAt    time.Time `json:"startedAt,omitempty"`
	HintsUsed    int       `json:"hintsUsed,omitempty"`
	Attempt      int       `json:"attempt,omitempty"`
	Checkpoint   int       `json:"checkpoint,omitempty"`
	SolutionRead bool      `json:"solutionRead,omitempty"`
	// Passed records that this attempt has already been graded correct, so
	// re-grading does not inflate the pass count.
	Passed bool `json:"passed,omitempty"`
}

// Active reports whether a lab is currently running.
func (s *State) Active() bool { return s != nil && s.ActiveLab != "" }

// Elapsed is wall-clock time since the attempt started. It survives the CLI
// exiting: the clock lives in the file, not in a process.
func (s *State) Elapsed() time.Duration {
	if s == nil || s.StartedAt.IsZero() {
		return 0
	}
	return time.Since(s.StartedAt)
}

// Home is ~/.cka-dojo, overridable with DOJO_HOME for tests.
func Home() (string, error) {
	if v := os.Getenv("DOJO_HOME"); v != "" {
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cka-dojo"), nil
}

// DevMode reports whether this process is developing the dojo rather than
// studying with it, which DOJO_DEV=1 declares.
//
// It exists because the engine cannot tell a dogfooded pass from a real one.
// Proving that a new exercise starts, breaks and grades correctly writes the
// same attempt record a learner's own work writes, and `recommend`, `learn`
// and `readiness` then rank a curriculum the learner has never touched as
// already passed. Development runs keep their own history instead; every
// other piece of state — the cluster, its SSH keys, the active lab and the
// lock that protects them — stays shared, because there is only one cluster.
func DevMode() bool {
	switch v := strings.ToLower(strings.TrimSpace(os.Getenv("DOJO_DEV"))); v {
	case "":
		// Unset: fall through to the marker file.
	case "0", "false", "no", "off":
		return false
	default:
		return true
	}
	// A marker in the working directory turns a whole checkout into a
	// development one. The environment variable does not survive a new shell,
	// and a dogfooding session is hundreds of commands across many of them;
	// forgetting the export once is all it takes to write a synthetic pass
	// into a real history.
	if _, err := os.Stat(DevMarker); err == nil {
		return true
	}
	return false
}

// DevMarker is the file whose presence in the working directory declares a
// development checkout. `make dogfood` creates it; `make study` removes it.
const DevMarker = ".dojo-dev"

func path(name string) (string, error) {
	home, err := Home()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, name), nil
}

// EnsureHome creates the state directory tree.
func EnsureHome() (string, error) {
	home, err := Home()
	if err != nil {
		return "", err
	}
	for _, d := range []string{"", "logs", "env"} {
		if err := os.MkdirAll(filepath.Join(home, d), 0o755); err != nil {
			return "", err
		}
	}
	return home, nil
}

// EnvDir is the per-profile directory holding generated SSH keys and any other
// host-side artefacts for an environment.
func EnvDir(profile string) (string, error) {
	p, err := path(filepath.Join("env", profile))
	if err != nil {
		return "", err
	}
	return p, os.MkdirAll(p, 0o700)
}

// Load reads config.yaml, returning defaults when it does not exist.
func Load() (*Config, error) {
	p, err := path("config.yaml")
	if err != nil {
		return nil, err
	}
	cfg := &Config{Profile: DefaultProfile, Curriculum: "cka-2026"}
	data, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", p, err)
	}
	if cfg.Profile == "" {
		cfg.Profile = DefaultProfile
	}
	if cfg.Curriculum == "" {
		cfg.Curriculum = "cka-2026"
	}
	return cfg, nil
}

// Save writes config.yaml.
func (c *Config) Save() error {
	if _, err := EnsureHome(); err != nil {
		return err
	}
	p, err := path("config.yaml")
	if err != nil {
		return err
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}

// LoadState reads state.json, returning an empty state when absent.
func LoadState() (*State, error) {
	p, err := path("state.json")
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return &State{}, nil
	}
	if err != nil {
		return nil, err
	}
	st := &State{}
	if err := json.Unmarshal(data, st); err != nil {
		// A corrupt state file must not wedge the CLI; dojo status should
		// still be able to tell the user to reset.
		return &State{}, nil
	}
	return st, nil
}

// SaveState atomically writes state.json.
func SaveState(st *State) error {
	if _, err := EnsureHome(); err != nil {
		return err
	}
	p, err := path("state.json")
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

// ClearState removes the active lab.
func ClearState() error { return SaveState(&State{}) }

// LogPath is where verbose output is always mirrored.
func LogPath() (string, error) {
	if _, err := EnsureHome(); err != nil {
		return "", err
	}
	return path(filepath.Join("logs", "dojo.log"))
}
