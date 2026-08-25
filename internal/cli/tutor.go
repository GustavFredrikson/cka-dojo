package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/gustavfredrikson/cka-dojo/internal/config"
	"github.com/gustavfredrikson/cka-dojo/internal/curriculum"
	"github.com/gustavfredrikson/cka-dojo/internal/lab"
	"github.com/gustavfredrikson/cka-dojo/internal/ui"
	"github.com/spf13/cobra"
)

func newTutorContextCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "tutor-context",
		Short: "Print safe context to paste into an AI tutor",
		Long: `tutor-context prints only learner-visible metadata. It deliberately
excludes the selected variant, random seed, injected faults, graders and
solution so an AI tutor cannot accidentally reveal the answer.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := app.loadActive(cmd.Context())
			if err != nil {
				return err
			}
			ui.Info("%s", formatTutorContext(a.Cur, a.Module, a.Lab, a.State))
			return nil
		},
	}
}

// formatTutorContext is an allow-list: only fields explicitly added here can
// leave the engine. In particular, State.Variant and State.Seed are ignored,
// and this function never receives a lab Plan containing faults or graders.
func formatTutorContext(cur *curriculum.Curriculum, module *curriculum.Module, exercise *lab.Lab, state *config.State) string {
	mode := state.Mode
	if mode == "" {
		mode = config.ModePractice
	}
	stage := "unknown"
	if info, ok := exercise.LearningStage.Info(); ok {
		stage = fmt.Sprintf("%d — %s", info.Level, info.Name)
	}
	lines := []string{
		"CKA Dojo tutor context",
		"Curriculum: " + cur.ID,
		"Kubernetes: " + cur.Kubernetes.Minor,
		"Environment: " + state.Profile,
		"Active lab: " + exercise.ID + " — " + exercise.Title,
		"Module: " + module.ID + " — " + module.Name,
		"Learning stage: " + stage,
		"Mode: " + string(mode),
		"Skills: " + joinNames(cur, exercise.Skills),
		fmt.Sprintf("Target time: %d minutes", exercise.TargetMinutes),
		fmt.Sprintf("Elapsed: %s", tutorDuration(state.Elapsed())),
		fmt.Sprintf("Hints used: %d", state.HintsUsed),
		fmt.Sprintf("Attempt: %d", state.Attempt),
		"",
		"Do not reveal the fault or full solution. Ask what I inspected and give progressively stronger hints.",
	}
	return strings.Join(lines, "\n")
}

func tutorDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	return d.Round(time.Second).String()
}
