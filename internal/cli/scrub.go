package cli

import (
	"github.com/gustavfredrikson/cka-dojo/internal/config"
	"github.com/gustavfredrikson/cka-dojo/internal/environment"
	"github.com/gustavfredrikson/cka-dojo/internal/ui"
	"github.com/spf13/cobra"
)

func newScrubCmd(app *App) *cobra.Command {
	var opts environment.ScrubOptions
	var node string
	cmd := &cobra.Command{
		Use:   "scrub",
		Short: "Return the student account to a bare, exam-morning state",
		Long: `scrub removes what you left behind in the student home directory.

Aliases, $do and $now, vim settings, shell history and scratch YAML all go,
and the distro skeleton comes back, on every node. The exam does not hand you
three weeks of your own shortcuts, so practising on top of them teaches a
setup you will not have. Run it after ` + "`dojo stop`" + `.

It keeps .ssh and .kube/config, because without them the environment stops
working. It is student-scope: files a lab had you create as root are left
alone, as is dojo's own state, which ` + "`dojo reset`" + ` depends on.`,
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
			if err := requireRunning(cmd.Context(), m); err != nil {
				return err
			}
			nodes := []string{}
			if node != "" {
				nodes = append(nodes, node)
			} else {
				for _, n := range m.Profile.Nodes {
					nodes = append(nodes, n.Name)
				}
			}
			if err := m.Scrub(cmd.Context(), nodes, opts); err != nil {
				return err
			}
			ui.Blank()
			if opts.DryRun {
				ui.OK("dry run: nothing was removed")
				return nil
			}
			ui.OK("student account scrubbed on %d node(s)", len(nodes))
			if !opts.Defaults {
				ui.Info("`dojo shell` now starts bare. Set up your own aliases as you would on exam day.")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "report what would go without removing anything")
	cmd.Flags().BoolVar(&opts.Defaults, "defaults", false, "re-seed the dojo alias block after the wipe")
	cmd.Flags().StringVar(&node, "node", "", "scrub this node only")
	return cmd
}
