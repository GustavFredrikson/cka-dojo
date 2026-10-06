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
	// SetupPending records an incomplete scenario setup or reset. Grading is
	// unavailable until setup completes successfully.
	SetupPending bool `json:"setupPending,omitempty"`
	// EnvironmentPending means machine provisioning has not finished. Recovery
	// resumes provisioning before attempting to repair or rebuild the scenario.
	EnvironmentPending bool `json:"environmentPending,omitempty"`
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

// ValidateID checks names used as filesystem components and VM identifiers.
// Dots are allowed for profiles such as upgrade-1.34, but path separators,
// whitespace, shell syntax and leading punctuation are never valid names.
func ValidateID(id string) error {
	if id == "" {
		return fmt.Errorf("id is required")
	}
	for i, c := range id {
		alnum := c >= 'a' && c <= 'z' || c >= '0' && c <= '9'
		if !alnum && (i == 0 || c != '-' && c != '_' && c != '.') {
			return fmt.Errorf("invalid id %q: use lowercase letters, digits, dots, hyphens or underscores, starting with a letter or digit", id)
		}
	}
	return nil
}

// openEnvRoot opens the owned environment tree without following an env/
// symlink. os.Root also prevents symlink races from escaping the state home.
// A nil root means the directory is absent and create is false.
func openEnvRoot(create bool) (*os.Root, string, error) {
	home, err := Home()
	if err != nil {
		return nil, "", err
	}
	home, err = filepath.Abs(home)
	if err != nil {
		return nil, "", err
	}
	if create {
		if err := os.MkdirAll(home, 0o755); err != nil {
			return nil, "", err
		}
	}
	root, err := os.OpenRoot(home)
	if !create && errors.Is(err, os.ErrNotExist) {
		return nil, filepath.Join(home, "env"), nil
	}
	if err != nil {
		return nil, "", err
	}
	defer root.Close()
	if create {
		if err := root.Mkdir("env", 0o755); err != nil && !errors.Is(err, os.ErrExist) {
			return nil, "", err
		}
	}
	info, err := root.Lstat("env")
	if !create && errors.Is(err, os.ErrNotExist) {
		return nil, filepath.Join(home, "env"), nil
	}
	if err != nil {
		return nil, "", err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, "", fmt.Errorf("environment directory %s must be a directory, not a symlink", filepath.Join(home, "env"))
	}
	envRoot, err := root.OpenRoot("env")
	if err != nil {
		return nil, "", err
	}
	opened, err := envRoot.Stat(".")
	if err != nil || !os.SameFile(info, opened) {
		envRoot.Close()
		return nil, "", fmt.Errorf("environment directory changed while opening %s", filepath.Join(home, "env"))
	}
	return envRoot, filepath.Join(home, "env"), nil
}

func checkEnvDir(root *os.Root, profile string) error {
	info, err := root.Lstat(profile)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("environment directory for %q must be a directory, not a symlink", profile)
	}
	return nil
}

// EnvDirPath validates the per-profile host path without creating it. This
// lets destruction reject unsafe paths before changing any VM or host files.
func EnvDirPath(profile string) (string, error) {
	if err := ValidateID(profile); err != nil {
		return "", err
	}
	root, base, err := openEnvRoot(false)
	if err != nil {
		return "", err
	}
	if root != nil {
		defer root.Close()
		if err := checkEnvDir(root, profile); err != nil {
			return "", err
		}
	}
	return filepath.Join(base, profile), nil
}

// EnvDir is the per-profile directory holding generated SSH keys and any other
// host-side artefacts for an environment.
func EnvDir(profile string) (string, error) {
	if err := ValidateID(profile); err != nil {
		return "", err
	}
	root, base, err := openEnvRoot(true)
	if err != nil {
		return "", err
	}
	defer root.Close()
	if err := root.Mkdir(profile, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
		return "", err
	}
	if err := checkEnvDir(root, profile); err != nil {
		return "", err
	}
	return filepath.Join(base, profile), nil
}

// RemoveEnvDir deletes only this profile's owned host artifacts. It never
// creates directories and Root.RemoveAll cannot follow a link outside env/.
func RemoveEnvDir(profile string) error {
	if err := ValidateID(profile); err != nil {
		return err
	}
	root, _, err := openEnvRoot(false)
	if err != nil || root == nil {
		return err
	}
	defer root.Close()
	if err := checkEnvDir(root, profile); err != nil {
		return err
	}
	return root.RemoveAll(profile)
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
	if err := ValidateID(cfg.Profile); err != nil {
		return nil, fmt.Errorf("config profile: %w", err)
	}
	if err := ValidateID(cfg.Curriculum); err != nil {
		return nil, fmt.Errorf("config curriculum: %w", err)
	}
	return cfg, nil
}

// Save writes config.yaml.
func (c *Config) Save() error {
	if c.Profile != "" {
		if err := ValidateID(c.Profile); err != nil {
			return fmt.Errorf("config profile: %w", err)
		}
	}
	if c.Curriculum != "" {
		if err := ValidateID(c.Curriculum); err != nil {
			return fmt.Errorf("config curriculum: %w", err)
		}
	}
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
