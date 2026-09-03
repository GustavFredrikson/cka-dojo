package cli

import (
	"fmt"
	"time"

	"github.com/gustavfredrikson/cka-dojo/internal/progress"
	"github.com/gustavfredrikson/cka-dojo/internal/readiness"
	"github.com/gustavfredrikson/cka-dojo/internal/ui"
	"github.com/spf13/cobra"
)

func newReadinessCmd(app *App) *cobra.Command {
	var detail bool
	cmd := &cobra.Command{
		Use:   "readiness",
		Short: "Assess whether you are ready to sit the exam",
		Long: `Readiness is stricter than progress.

Progress reports what you have done. Readiness reports what you have proven,
weighted by what each domain is worth on the exam. Only mastered labs count: a
lab passed once, or passed with a hint, is evidence that you have seen the
topic, not that you can do it under pressure.

A domain you have never touched is a blocker, not an average. No amount of
strength elsewhere buys past it, because the exam will ask anyway.

The report also names what it cannot see: competencies this content set does
not cover, and the fact that nothing here is timed.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cur, err := loadCurriculum(app)
			if err != nil {
				return err
			}
			prog, err := progress.Load()
			if err != nil {
				return err
			}
			r := readiness.Assess(cur, prog, time.Now())
			printReadiness(r, detail)
			return nil
		},
	}
	cmd.Flags().BoolVar(&detail, "detail", false, "list the specific labs behind each finding")
	return cmd
}

func printReadiness(r *readiness.Report, detail bool) {
	ui.Heading("Readiness")

	rows := make([][]string, 0, len(r.Domains))
	for _, d := range r.Domains {
		state := fmt.Sprintf("%.0f%%", d.Coverage*100)
		if d.Labs == 0 {
			state = "no labs"
		} else if d.Untouched() {
			state = "untouched"
		}
		rows = append(rows, []string{
			d.Name,
			fmt.Sprintf("%d%%", d.Weight),
			fmt.Sprintf("%d", d.Labs),
			fmt.Sprintf("%d", d.Attempted),
			fmt.Sprintf("%d", d.Mastered),
			state,
		})
	}
	ui.Table([]string{"DOMAIN", "EXAM", "LABS", "ATTEMPTED", "MASTERED", "COVERAGE"}, rows)

	ui.Blank()
	line := fmt.Sprintf("%s — %.0f%% of the existing curriculum mastered, exam-weighted.",
		r.Verdict, r.Score*100)
	switch r.Verdict {
	case readiness.VerdictLikelyReady:
		ui.OK("%s", line)
	case readiness.VerdictApproaching:
		ui.Warn("%s", line)
	default:
		ui.Fail("%s", line)
	}

	if steps := r.NextSteps(); len(steps) > 0 {
		ui.Blank()
		ui.Heading("Do this next")
		for i, s := range steps {
			ui.Info("%d. %s", i+1, s)
		}
	}

	if detail {
		printReadinessDetail(r)
	}

	ui.Blank()
	ui.Heading("What this score cannot see")
	ui.Info("Nothing here is timed. A mastered lab says you can do the task, not")
	ui.Info("that you can do seventeen of them in two hours.")
	if len(r.Uncovered) > 0 {
		ui.Blank()
		ui.Info("%d published competencies have no lab behind them:", len(r.Uncovered))
		for _, g := range r.Uncovered {
			ui.Info("  %s — %s", g.Name, g.Note)
		}
		ui.Blank()
		ui.Info("Study those elsewhere. Reaching 100%% here would not cover them.")
	}
	if !detail {
		ui.Blank()
		ui.Info("`dojo readiness --detail` lists the labs behind each finding.")
	}
}

func printReadinessDetail(r *readiness.Report) {
	section := func(title string, labs []string) {
		if len(labs) == 0 {
			return
		}
		ui.Blank()
		ui.Heading("%s (%d)", title, len(labs))
		for _, id := range labs {
			ui.Info("  %s", id)
		}
	}
	section("Passed once, needs a second clean pass", r.Unproven)
	section("Attempted but never passed", r.Failing)
	section("Never passed without help", r.HintDependent)

	if len(r.Slow) > 0 {
		ui.Blank()
		ui.Heading("Ran long (%d)", len(r.Slow))
		for _, p := range r.Slow {
			ui.Info("  %s — %s against a %s target", p.Lab,
				formatSeconds(p.Seconds), formatSeconds(p.Target))
		}
	}
}
