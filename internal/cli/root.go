package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/gustavfredrikson/cka-dojo/internal/ui"
	"github.com/spf13/cobra"
)

// Version is stamped at build time.
var Version = "0.1.0-dev"

// Execute runs the CLI.
func Execute() int {
	app := &App{}

	root := &cobra.Command{
		Use:   "dojo",
		Short: "CKA training on disposable, real Kubernetes clusters",
		Long: `dojo builds real kubeadm clusters in local VMs, breaks them on purpose,
and grades the state you leave behind rather than the commands you type.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return app.init()
		},
	}
	root.PersistentFlags().BoolVarP(new(bool), "verbose", "v", false, "show every step and command")
	root.PersistentFlags().StringVar(&app.contentFlag, "content", "", "read curriculum and environments from this directory")
	root.PersistentFlags().StringVar(&app.profileFlag, "profile", "", "environment profile to act on")
	// The flag has to take effect before PersistentPreRunE reads config.
	cobra.OnInitialize(func() {
		if v, _ := root.PersistentFlags().GetBool("verbose"); v {
			ui.SetVerbose(true)
		}
	})

	root.AddCommand(
		newDoctorCmd(app),
		newSetupCmd(app),
		newShellCmd(app),
		newEnvCmd(app),
		newStatusCmd(app),
		newProgressCmd(app),
		newLearnCmd(app),
		newRecommendCmd(app),
		newTutorContextCmd(app),
		newLabsCmd(app),
		newStartCmd(app),
		newTaskCmd(app),
		newCheckCmd(app),
		newGradeCmd(app),
		newHintCmd(app),
		newSolutionCmd(app),
		newResetCmd(app),
		newStopCmd(app),
		newContentCmd(app),
		newVersionCmd(),
	)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	err := root.ExecuteContext(ctx)
	ui.CloseLog()
	if err != nil {
		ui.Fail("%v", err)
		return 1
	}
	return 0
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the dojo version",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println(Version)
			return nil
		},
	}
}
