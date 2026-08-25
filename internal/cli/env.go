package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/gustavfredrikson/cka-dojo/internal/config"
	"github.com/gustavfredrikson/cka-dojo/internal/environment"
	"github.com/gustavfredrikson/cka-dojo/internal/ui"
	"github.com/spf13/cobra"
)

func newEnvCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "env",
		Short: "Manage training environments",
		Long: `Environments are kept on disk per profile and only one runs at a time.

Switching profiles stops the running one rather than deleting it, so going
back to your everyday cluster does not mean rebuilding it.`,
	}
	cmd.AddCommand(
		newEnvStatusCmd(app),
		newEnvListCmd(app),
		newEnvStopCmd(app),
		newEnvResetCmd(app),
		newEnvDestroyCmd(app),
	)
	return cmd
}

func newEnvStatusCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show the current environment's machines",
		RunE: func(cmd *cobra.Command, args []string) error {
			m, err := app.Manager("")
			if err != nil {
				return err
			}
			sts, err := m.Status(cmd.Context())
			if err != nil {
				return err
			}
			ui.Heading("Environment %s", m.Profile.ID)
			rows := make([][]string, 0, len(sts))
			for _, s := range sts {
				ip := s.IP
				if ip == "" {
					ip = "-"
				}
				rows = append(rows, []string{s.Node.Name, s.Node.Role, string(s.Status), ip, s.Node.Memory})
			}
			ui.Table([]string{"NODE", "ROLE", "STATE", "ADDRESS", "MEMORY"}, rows)

			if running, _ := m.Running(cmd.Context()); running && m.Profile.ControlPlane() != nil {
				ui.Blank()
				out, err := m.Kubectl(cmd.Context(), "get", "nodes", "-o", "wide", "--no-headers")
				if err != nil {
					ui.Warn("the API server is not answering: %v", err)
					return nil
				}
				ui.Heading("Kubernetes nodes")
				for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
					f := strings.Fields(line)
					if len(f) >= 6 {
						ui.Info("%-10s %-10s %-10s %s", f[0], f[1], f[4], f[5])
					}
				}
			}
			return nil
		},
	}
}

func newEnvListCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List every dojo environment on this machine",
		RunE: func(cmd *cobra.Command, args []string) error {
			profiles, err := environment.ListProfiles(app.Src)
			if err != nil {
				return err
			}
			nodes, err := app.Provider.List(cmd.Context(), environment.NamePrefix)
			if err != nil {
				return err
			}
			counts := map[string]int{}
			running := map[string]int{}
			for _, n := range nodes {
				name := strings.TrimPrefix(n.Name, environment.NamePrefix)
				profile := name
				if i := strings.LastIndex(name, "-"); i > 0 {
					profile = name[:i]
				}
				counts[profile]++
				if n.Status == "running" {
					running[profile]++
				}
			}
			rows := make([][]string, 0, len(profiles))
			for _, p := range profiles {
				state := "not created"
				switch {
				case counts[p] == 0:
				case running[p] == counts[p]:
					state = "running"
				case running[p] == 0:
					state = "stopped"
				default:
					state = "partly running"
				}
				marker := ""
				if p == app.Profile() {
					marker = "*"
				}
				rows = append(rows, []string{marker + p, state, fmt.Sprintf("%d/%d", running[p], counts[p])})
			}
			ui.Table([]string{"PROFILE", "STATE", "UP"}, rows)
			ui.Blank()
			ui.Info("Stopped machines keep their disks, so switching profiles is cheap.")
			return nil
		},
	}
}

func newEnvStopCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Shut the environment down, keeping its disks",
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
			if err := m.Stop(cmd.Context()); err != nil {
				return err
			}
			ui.OK("environment %q stopped", m.Profile.ID)
			return nil
		},
	}
}

func newEnvResetCmd(app *App) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Destroy and rebuild the environment",
		Long: `reset is the guaranteed escape hatch.

A lab's own reset repairs what the lab did. It cannot undo everything a
learner might do to a cluster, and trying to make it do so would be a
bottomless engineering problem. When the cluster is beyond saving, rebuild it.`,
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
			if !yes && !confirm(fmt.Sprintf("Destroy and rebuild environment %q? This takes about 15-25 minutes.", m.Profile.ID)) {
				ui.Info("nothing changed")
				return nil
			}
			if err := m.Destroy(cmd.Context()); err != nil {
				return err
			}
			if err := config.ClearState(); err != nil {
				return err
			}
			return setupEnvironment(cmd.Context(), m, false)
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "do not ask for confirmation")
	return cmd
}

func newEnvDestroyCmd(app *App) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "destroy",
		Short: "Delete the environment's machines and disks",
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
			if !yes && !confirm(fmt.Sprintf("Delete every machine in environment %q?", m.Profile.ID)) {
				ui.Info("nothing changed")
				return nil
			}
			if err := m.Destroy(cmd.Context()); err != nil {
				return err
			}
			if err := config.ClearState(); err != nil {
				return err
			}
			ui.OK("environment %q destroyed", m.Profile.ID)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "do not ask for confirmation")
	return cmd
}

// confirm asks a yes/no question. A non-interactive stdin answers no, so an
// unattended run never destroys anything by accident.
func confirm(question string) bool {
	fi, err := os.Stdin.Stat()
	if err != nil || fi.Mode()&os.ModeCharDevice == 0 {
		ui.Warn("%s (no terminal to ask on; assuming no -- pass --yes to proceed)", question)
		return false
	}
	fmt.Printf("%s [y/N] ", question)
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return false
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes"
}
