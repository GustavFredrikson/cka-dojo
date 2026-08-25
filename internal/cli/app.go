// Package cli wires the engine to cobra commands.
package cli

import (
	"context"
	"fmt"
	"os"

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

// init prepares config, content and logging. It runs before every command.
func (a *App) init() error {
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
	return nil
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
