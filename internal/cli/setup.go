package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/gustavfredrikson/cka-dojo/internal/config"
	"github.com/gustavfredrikson/cka-dojo/internal/curriculum"
	"github.com/gustavfredrikson/cka-dojo/internal/environment"
	"github.com/gustavfredrikson/cka-dojo/internal/ui"
	"github.com/spf13/cobra"
)

// setupTimeout covers a cold build: image download, apt, kubeadm init, CNI.
const setupTimeout = 60 * time.Minute

func loadCurriculum(app *App) (*curriculum.Curriculum, error) {
	return curriculum.Load(app.Src, app.Cfg.Curriculum)
}

func newSetupCmd(app *App) *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Build the training environment",
		Long: `setup creates the machines and installs Kubernetes on them.

It is idempotent and resumable: interrupt it, run it again, and it picks up
where it stopped rather than starting over.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			lock, err := config.Acquire()
			if err != nil {
				return err
			}
			defer lock.Release()

			m, err := app.Manager("")
			if err != nil {
				return err
			}
			return setupEnvironment(cmd.Context(), m, force)
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "re-run provisioning steps that already completed")
	return cmd
}

func setupEnvironment(ctx context.Context, m *environment.Manager, force bool) error {
	ctx, cancel := context.WithTimeout(ctx, setupTimeout)
	defer cancel()

	ui.Heading("Building environment %q", m.Profile.ID)
	ui.Info("%s", m.Profile.Description)
	ui.Info("Kubernetes %s, %s %s, %d machines",
		m.Profile.Kubernetes.Version, m.Profile.CNI.Provider, m.Profile.CNI.Version, len(m.Profile.Nodes))
	ui.Info("The first run downloads an OS image and installs packages; expect 15-25 minutes.")
	ui.Blank()

	// Only one environment runs at a time. Stop the other profile's machines
	// before claiming this one's memory, rather than letting both fight for it.
	if err := m.StopOthers(ctx); err != nil {
		return err
	}

	start := time.Now()
	if err := m.Up(ctx, force); err != nil {
		return err
	}
	ui.Blank()
	ui.OK("environment %q ready in %s", m.Profile.ID, time.Since(start).Round(time.Second))
	if ws := m.Profile.Workstation(); ws != nil {
		ui.Info("Open a shell with `dojo shell`, then try `kubectl get nodes`.")
	}
	return nil
}

// ensureUp brings the environment up if it is not already running, so that
// `dojo start <lab>` works from a cold machine.
func ensureUp(ctx context.Context, m *environment.Manager) error {
	running, err := m.Running(ctx)
	if err != nil {
		return err
	}
	if running {
		// Already up, but another profile may be up alongside it -- a lab that
		// names a second profile starts this one without ever stopping that.
		return m.StopOthers(ctx)
	}
	ui.Warn("environment %q is not running; building it first", m.Profile.ID)
	return setupEnvironment(ctx, m, false)
}

func newShellCmd(app *App) *cobra.Command {
	var node string
	cmd := &cobra.Command{
		Use:   "shell",
		Short: "Open a shell on the workstation",
		Long: `shell drops you onto the terminal machine as the student user.

The workstation is not part of the cluster, which is what lets a lab break a
control-plane node without breaking the shell you are working from. From
there, ssh cp1, ssh worker1 and ssh worker2 all work.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			m, err := app.Manager("")
			if err != nil {
				return err
			}
			target := node
			if target == "" {
				ws := m.Profile.Workstation()
				if ws == nil {
					return fmt.Errorf("profile %q has no workstation; use --node", m.Profile.ID)
				}
				target = ws.Name
			}
			if m.Profile.NodeByName(target) == nil {
				return fmt.Errorf("profile %q has no node %q", m.Profile.ID, target)
			}
			if err := requireRunning(cmd.Context(), m); err != nil {
				return err
			}
			return m.Prov.Shell(cmd.Context(), m.VMName(target), environment.StudentUser, args)
		},
	}
	cmd.Flags().StringVar(&node, "node", "", "open a shell on this node instead of the workstation")
	return cmd
}
