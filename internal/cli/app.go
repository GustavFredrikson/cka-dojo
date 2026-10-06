// Package cli wires the engine to cobra commands.
package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gustavfredrikson/cka-dojo/internal/config"
	"github.com/gustavfredrikson/cka-dojo/internal/content"
	"github.com/gustavfredrikson/cka-dojo/internal/environment"
	"github.com/gustavfredrikson/cka-dojo/internal/provider"
	"github.com/gustavfredrikson/cka-dojo/internal/provider/lima"
	"github.com/gustavfredrikson/cka-dojo/internal/ui"
)

// App is the shared context every command builds on.
type App struct {
	Cfg      *config.Config
	Src      *content.Source
	Provider provider.Provider

	contentFlag string
	profileFlag string
}

// Profile is the environment profile the command should act on.
func (a *App) Profile() string {
	if a.profileFlag != "" {
		return a.profileFlag
	}
	if a.Cfg.Profile != "" {
		return a.Cfg.Profile
	}
	return config.DefaultProfile
}

// CurrentManager follows an explicit profile first, then the running lab's
// profile. A lab can select a special environment without changing config.
func (a *App) CurrentManager() (*environment.Manager, error) {
	if a.profileFlag != "" {
		return a.Manager(a.profileFlag)
	}
	st, err := config.LoadState()
	if err != nil {
		return nil, err
	}
	if st.Active() && st.Profile != "" {
		return a.Manager(st.Profile)
	}
	return a.Manager("")
}

// Manager loads the profile and returns an environment manager for it.
func (a *App) Manager(id string) (*environment.Manager, error) {
	if id == "" {
		id = a.Profile()
	}
	prof, err := environment.LoadProfile(a.Src, id)
	if err != nil {
		return nil, err
	}
	return environment.New(prof, a.Provider, a.Src), nil
}

// init prepares config, content and logging. It runs before every command;
// command is the name of the one about to run.
func (a *App) init(command string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	a.Cfg = cfg
	if cfg.Verbose {
		ui.SetVerbose(true)
	}
	src, err := content.Resolve(a.contentFlag, cfg.ContentDir)
	if err != nil {
		return err
	}
	a.Src = src
	a.Provider = lima.New()

	if p, err := config.LogPath(); err == nil {
		if f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err == nil {
			ui.SetLogFile(f)
		}
	}
	ui.Detail("content source: %s", src.Origin)
	announceHistory(command)
	return nil
}

// writesHistory lists the commands that record an attempt. `start` counts the
// attempt and `grade` counts the pass; everything else only reads.
var writesHistory = map[string]bool{"start": true, "grade": true}

// announceHistory keeps the two study histories from being confused for each
// other. Development mode is always announced, because every reading command
// — progress, learn, recommend, readiness — then answers from the development
// record. The reverse nudge fires only when a repo build is about to write a
// real attempt, which is exactly how dogfooding lands in a learner's history.
func announceHistory(command string) {
	if config.DevMode() {
		ui.Warn("dev mode: attempts record to progress-dev.json, not your study history")
		return
	}
	if writesHistory[command] && repoBuild() {
		ui.Warn("recording to your real study history from a repo build; run `make dogfood` first to dogfood instead")
	}
}

// repoBuild reports whether the running binary is a `make build` artefact
// sitting in a source checkout, rather than an installed dojo.
func repoBuild() bool {
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	root := filepath.Dir(filepath.Dir(exe)) // bin/dojo -> the checkout
	for _, marker := range []string{"go.mod", "curriculum"} {
		if _, err := os.Stat(filepath.Join(root, marker)); err != nil {
			return false
		}
	}
	return true
}

// requireRunning fails with a useful message when the environment is not up.
func requireRunning(ctx context.Context, m *environment.Manager) error {
	running, err := m.Running(ctx)
	if err != nil {
		return err
	}
	if !running {
		return fmt.Errorf("environment %q is not running; start it with `dojo setup`", m.Profile.ID)
	}
	return nil
}
