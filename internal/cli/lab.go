package cli

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/gustavfredrikson/cka-dojo/internal/config"
	"github.com/gustavfredrikson/cka-dojo/internal/curriculum"
	"github.com/gustavfredrikson/cka-dojo/internal/environment"
	"github.com/gustavfredrikson/cka-dojo/internal/grader"
	"github.com/gustavfredrikson/cka-dojo/internal/lab"
	"github.com/gustavfredrikson/cka-dojo/internal/learning"
	"github.com/gustavfredrikson/cka-dojo/internal/progress"
	"github.com/gustavfredrikson/cka-dojo/internal/provider"
	"github.com/gustavfredrikson/cka-dojo/internal/ui"
	"github.com/spf13/cobra"
)

// labTimeout bounds a single engine operation on a lab, not the learner.
const labTimeout = 20 * time.Minute

// active is the currently running lab, resolved from state on disk.
type active struct {
	State  *config.State
	Cur    *curriculum.Curriculum
	Lab    *lab.Lab
	Module *curriculum.Module
	Env    *environment.Manager
	Runner *lab.Runner
	Plan   *lab.Plan
}

// loadActive rebuilds everything needed to act on the running lab. State on
// disk holds only identifiers; the plan is rebuilt from content so that
// editing a lab and re-grading works while authoring.
func (app *App) loadActive(ctx context.Context) (*active, error) {
	st, err := config.LoadState()
	if err != nil {
		return nil, err
	}
	if !st.Active() {
		return nil, fmt.Errorf("no lab is running; start one with `dojo start <lab>` or list them with `dojo labs`")
	}
	cur, err := loadCurriculum(app)
	if err != nil {
		return nil, err
	}
	l, mod := cur.LabByID(st.ActiveLab)
	if l == nil {
		return nil, fmt.Errorf("the active lab %q is no longer in the curriculum; run `dojo stop`", st.ActiveLab)
	}
	m, err := app.Manager(st.Profile)
	if err != nil {
		return nil, err
	}
	var variant *lab.Variant
	if st.Variant != "" {
		if variant = l.VariantByID(st.Variant); variant == nil {
			return nil, fmt.Errorf("lab %s no longer has variant %q; run `dojo stop`", l.ID, st.Variant)
		}
	}
	plan, err := l.Build(variant)
	if err != nil {
		return nil, err
	}
	return &active{
		State: st, Cur: cur, Lab: l, Module: mod, Env: m,
		Plan: plan, Runner: lab.NewRunner(m, plan),
	}, nil
}

func newLabsCmd(app *App) *cobra.Command {
	var domain, skill string
	cmd := &cobra.Command{
		Use:   "labs",
		Short: "List the available labs",
		RunE: func(cmd *cobra.Command, args []string) error {
			cur, err := loadCurriculum(app)
			if err != nil {
				return err
			}
			prog, err := progress.Load()
			if err != nil {
				return err
			}
			rows := [][]string{}
			for _, m := range cur.Modules {
				for _, l := range m.Labs {
					if domain != "" && !contains(l.Domains, domain) {
						continue
					}
					if skill != "" && !contains(l.Skills, skill) {
						continue
					}
					rows = append(rows, []string{
						l.ID,
						m.ID,
						stageLabel(l),
						strings.Repeat("*", l.Difficulty),
						fmt.Sprintf("%dm", l.TargetMinutes),
						statusMark(prog.Labs[l.ID]),
						l.Title,
					})
				}
			}
			if len(rows) == 0 {
				ui.Info("no labs match that filter")
				return nil
			}
			ui.Table([]string{"LAB", "MODULE", "STAGE", "DIFF", "TARGET", "STATE", "TITLE"}, rows)
			ui.Blank()
			ui.Info("Start one with `dojo start <lab>`.")
			return nil
		},
	}
	cmd.Flags().StringVar(&domain, "domain", "", "only labs in this exam domain")
	cmd.Flags().StringVar(&skill, "skill", "", "only labs exercising this skill")
	return cmd
}

func stageLabel(l *lab.Lab) string {
	if info, ok := l.LearningStage.Info(); ok {
		return fmt.Sprintf("%d %s", info.Level, info.Name)
	}
	return "?"
}

func statusMark(a *progress.Attempt) string {
	switch {
	case a == nil || a.Attempts == 0:
		return "new"
	case a.Mastered():
		return "mastered"
	case a.Passes > 0:
		return "passed"
	default:
		return "attempted"
	}
}

func contains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

func newStartCmd(app *App) *cobra.Command {
	var (
		seed              int64
		variant           string
		mode              string
		force             bool
		skipPrerequisites bool
	)
	cmd := &cobra.Command{
		Use:   "start <lab>",
		Short: "Build a lab scenario and show its task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			lock, err := config.Acquire()
			if err != nil {
				return err
			}
			defer lock.Release()

			cur, err := loadCurriculum(app)
			if err != nil {
				return err
			}
			l, _ := cur.LabByID(args[0])
			if l == nil {
				return fmt.Errorf("no lab called %q; see `dojo labs`", args[0])
			}
			history, err := progress.Load()
			if err != nil {
				return err
			}
			if missing := learning.Missing(l, history); len(missing) > 0 && !skipPrerequisites {
				parts := make([]string, len(missing))
				for i, requirement := range missing {
					parts[i] = requirement.String()
				}
				return fmt.Errorf("%s is locked; complete %s first, or pass --skip-prerequisites to jump ahead",
					l.ID, strings.Join(parts, ", "))
			}

			st, err := config.LoadState()
			if err != nil {
				return err
			}
			if st.Active() && st.ActiveLab != l.ID && !force {
				return fmt.Errorf("lab %q is still running; finish it, or run `dojo stop` first", st.ActiveLab)
			}

			m, err := app.Manager(l.Profile(app.Profile()))
			if err != nil {
				return err
			}
			if seed == 0 {
				seed = rand.Int63n(9_000_000) + 1_000_000
			}
			v := l.PickVariant(seed)
			if variant != "" {
				if v = l.VariantByID(variant); v == nil {
					return fmt.Errorf("lab %s has no variant %q", l.ID, variant)
				}
			}
			plan, err := l.Build(v)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), setupTimeout)
			defer cancel()

			// Validate the new plan before touching the old lab. Clean it up
			// before switching profiles, while its machines are still running.
			if st.Active() {
				prev, err := app.loadActive(ctx)
				if err != nil {
					return err
				}
				prev.State.SetupPending = true
				if err := config.SaveState(prev.State); err != nil {
					return err
				}
				if err := prepareScenarioEnvironment(ctx, prev); err != nil {
					return incompleteSetupError(err)
				}
				ui.Step("clearing the previous lab")
				if err := prev.Runner.Teardown(ctx); err != nil {
					return incompleteSetupError(fmt.Errorf("could not clean up the previous lab: %w", err))
				}
			}

			nextAttempt := 1
			if att := history.Labs[l.ID]; att != nil {
				nextAttempt = att.Attempts + 1
			}
			newState := &config.State{
				ActiveLab:          l.ID,
				Profile:            m.Profile.ID,
				Mode:               config.Mode(mode),
				Variant:            variantID(v),
				Seed:               seed,
				Attempt:            nextAttempt,
				SetupPending:       true,
				EnvironmentPending: true,
			}
			if err := config.SaveState(newState); err != nil {
				return err
			}
			runner := lab.NewRunner(m, plan)
			if err := ensureUp(ctx, m); err != nil {
				return incompleteSetupError(err)
			}
			newState.EnvironmentPending = false
			if err := config.SaveState(newState); err != nil {
				return incompleteSetupError(err)
			}
			ui.Step("building scenario for %s", l.ID)
			if err := runner.Setup(ctx); err != nil {
				return incompleteSetupError(err)
			}
			a := &active{State: newState, Lab: l}
			if err := finishSetup(a, history); err != nil {
				return incompleteSetupError(err)
			}

			ui.Blank()
			printTask(l, newState)
			return nil
		},
	}
	cmd.Flags().Int64Var(&seed, "seed", 0, "reproduce an exact scenario")
	cmd.Flags().StringVar(&variant, "variant", "", "force a specific variant (spoils the surprise)")
	cmd.Flags().StringVar(&mode, "mode", string(config.ModePractice), "guided, practice or exam")
	cmd.Flags().BoolVar(&force, "force", false, "abandon a lab that is still running")
	cmd.Flags().BoolVar(&skipPrerequisites, "skip-prerequisites", false, "jump ahead in the learning path")
	return cmd
}

func variantID(v *lab.Variant) string {
	if v == nil {
		return ""
	}
	return v.ID
}

func incompleteSetupError(err error) error {
	return fmt.Errorf("scenario setup is incomplete: %w; retry with `dojo reset` or clean up with `dojo stop` (`dojo env reset` rebuilds the environment)", err)
}

func requireCompleteSetup(st *config.State) error {
	if st.SetupPending || st.EnvironmentPending {
		return incompleteSetupError(fmt.Errorf("lab %q is not ready for grading", st.ActiveLab))
	}
	return nil
}

// Scenario faults may deliberately make the API server or kubelet unhealthy.
// Start stopped guests without the provisioning health checks, then let
// teardown repair those faults. An unfinished or missing environment must
// finish provisioning first, and keeps that phase persisted on failure.
func prepareScenarioEnvironment(ctx context.Context, a *active) error {
	if !a.State.EnvironmentPending {
		statuses, err := a.Env.Status(ctx)
		if err != nil {
			return err
		}
		for _, status := range statuses {
			if status.Status == provider.StatusMissing {
				a.State.EnvironmentPending = true
				if err := config.SaveState(a.State); err != nil {
					return err
				}
				break
			}
		}
	}
	if a.State.EnvironmentPending {
		if err := setupEnvironment(ctx, a.Env, false); err != nil {
			return err
		}
		a.State.EnvironmentPending = false
		return config.SaveState(a.State)
	}
	if err := a.Env.StopOthers(ctx); err != nil {
		return err
	}
	return a.Env.EnsureNodes(ctx)
}

// finishSetup starts history only after the complete scenario is built. The
// intended attempt number is persisted first, so a failed history/state write
// can be retried without counting the same attempt twice.
func finishSetup(a *active, history *progress.File) error {
	att := history.Labs[a.Lab.ID]
	if att == nil || att.Attempts < a.State.Attempt {
		att = history.StartAttempt(a.Lab.ID, a.State.Seed, a.State.Variant, a.Lab.Skills, a.Lab.Domains)
		if err := history.Save(); err != nil {
			return err
		}
		a.State.Attempt = att.Attempts
	}
	a.State.SetupPending = false
	a.State.StartedAt = time.Now()
	a.State.Passed = false
	a.State.Checkpoint = 0
	return config.SaveState(a.State)
}

func printTask(l *lab.Lab, st *config.State) {
	task, err := l.Task()
	if err != nil {
		ui.Fail("%v", err)
		return
	}
	ui.Heading("%s", l.Title)
	ui.Info("lab %s  |  stage %s  |  target %d minutes  |  difficulty %s  |  seed %d",
		l.ID, stageLabel(l), l.TargetMinutes, strings.Repeat("*", l.Difficulty), st.Seed)
	ui.Blank()
	// task.md repeats the title as an H1 so it reads well on its own; drop it
	// here rather than printing the same line twice.
	ui.Markdown(stripLeadingHeading(task))
	ui.Blank()
	if len(l.Checkpoints) > 0 {
		printCheckpoint(l, st.Checkpoint)
		ui.Info("Work in `dojo shell`. Validate this step with `dojo check`.")
	} else {
		ui.Info("Work in `dojo shell`. Check your work with `dojo grade`.")
	}
}

func printCheckpoint(l *lab.Lab, index int) {
	if index < 0 || index >= len(l.Checkpoints) {
		return
	}
	cp := l.Checkpoints[index]
	ui.Blank()
	ui.Heading("Checkpoint %d of %d — %s", index+1, len(l.Checkpoints), cp.ID)
	ui.Markdown(cp.Task)
	if cp.Answer != nil {
		ui.Info("Answer with `dojo check <answer>`.")
	}
}

// stripLeadingHeading removes a leading level-one Markdown heading.
func stripLeadingHeading(text string) string {
	trimmed := strings.TrimLeft(text, "\n")
	if !strings.HasPrefix(trimmed, "# ") {
		return text
	}
	if i := strings.Index(trimmed, "\n"); i >= 0 {
		return strings.TrimLeft(trimmed[i+1:], "\n")
	}
	return ""
}

func newTaskCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "task",
		Short: "Show the running lab's task again",
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := app.loadActive(cmd.Context())
			if err != nil {
				return err
			}
			printTask(a.Lab, a.State)
			return nil
		},
	}
}

func newCheckCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "check [answer]",
		Short: "Validate and advance an interactive checkpoint",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			lock, err := config.Acquire()
			if err != nil {
				return err
			}
			defer lock.Release()
			a, err := app.loadActive(cmd.Context())
			if err != nil {
				return err
			}
			if err := requireCompleteSetup(a.State); err != nil {
				return err
			}
			if len(a.Plan.Checkpoints) == 0 {
				return fmt.Errorf("lab %s has no interactive checkpoints; use `dojo grade`", a.Lab.ID)
			}
			if a.State.Checkpoint >= len(a.Plan.Checkpoints) {
				return fmt.Errorf("all checkpoints are complete; use `dojo grade`")
			}
			if a.State.Mode == config.ModeExam {
				return fmt.Errorf("checkpoints are unavailable during an exam")
			}
			if err := requireRunning(cmd.Context(), a.Env); err != nil {
				return err
			}

			cp := a.Plan.Checkpoints[a.State.Checkpoint]
			if cp.Definition.Answer != nil {
				answer := strings.Join(args, " ")
				if answer == "" {
					return fmt.Errorf("this checkpoint expects an answer: `dojo check <answer>`")
				}
				if !cp.Definition.Answer.Matches(answer) {
					return fmt.Errorf("not quite; inspect the resources again and retry")
				}
			} else if len(args) > 0 {
				return fmt.Errorf("this checkpoint validates cluster state and takes no answer")
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), labTimeout)
			defer cancel()
			if len(cp.Checks) > 0 || len(cp.AnyChecks) > 0 {
				rep, err := a.Runner.GradeCheckpoint(ctx, a.State.Checkpoint)
				if err != nil {
					return err
				}
				printReport(rep)
				if !rep.Passed {
					return fmt.Errorf("checkpoint is not complete yet")
				}
			}

			a.State.Checkpoint++
			if err := config.SaveState(a.State); err != nil {
				return err
			}
			ui.OK("checkpoint complete")
			if a.State.Checkpoint < len(a.Plan.Checkpoints) {
				printCheckpoint(a.Lab, a.State.Checkpoint)
				return nil
			}

			ui.Step("checking the completed exercise")
			rep, err := a.Runner.Grade(ctx)
			if err != nil {
				return err
			}
			printReport(rep)
			if !rep.Passed {
				return fmt.Errorf("checkpoints are complete, but the final required state is not")
			}
			att, err := recordGrade(a, true)
			if err != nil {
				return err
			}
			ui.OK("exercise passed in %s", a.State.Elapsed().Round(time.Second))
			if att.Mastered() {
				ui.OK("this exercise now counts as mastered")
			}
			ui.Info("Finish with `dojo stop`, then return to `dojo learn %s`.", a.Module.Name)
			return nil
		},
	}
}

func newGradeCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "grade",
		Short: "Check the cluster against the lab's requirements",
		Long: `grade looks at the state of the cluster, never at what you typed.

Imperative kubectl, an edited manifest, kubectl patch and kubectl edit are all
equally correct if they leave the cluster in the required state.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			lock, err := config.Acquire()
			if err != nil {
				return err
			}
			defer lock.Release()

			a, err := app.loadActive(cmd.Context())
			if err != nil {
				return err
			}
			if err := requireCompleteSetup(a.State); err != nil {
				return err
			}
			if a.State.Mode == config.ModeExam {
				return fmt.Errorf("grading is hidden during an exam; finish it with `dojo exam finish`")
			}
			if len(a.Plan.Checkpoints) > 0 && a.State.Checkpoint < len(a.Plan.Checkpoints) {
				return fmt.Errorf("complete checkpoint %d of %d with `dojo check` before final grading",
					a.State.Checkpoint+1, len(a.Plan.Checkpoints))
			}
			if err := requireRunning(cmd.Context(), a.Env); err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), labTimeout)
			defer cancel()

			ui.Step("grading %s", a.Lab.ID)
			rep, err := a.Runner.Grade(ctx)
			if err != nil {
				return err
			}
			ui.Blank()
			printReport(rep)

			att, err := recordGrade(a, rep.Passed)
			if err != nil {
				return err
			}

			ui.Blank()
			elapsed := a.State.Elapsed().Round(time.Second)
			if rep.Passed {
				ui.OK("passed in %s with %d hint(s)", elapsed, a.State.HintsUsed)
				target := time.Duration(a.Lab.TargetMinutes) * time.Minute
				if elapsed > target {
					ui.Info("Target time is %s. Speed comes later; correctness first.", target)
				}
				if att.Mastered() {
					ui.OK("this lab now counts as mastered")
				} else {
					ui.Info("Mastery needs two passes, the most recent without hints.")
				}
				ui.Info("Finish up with `dojo stop`, or start the next lab.")
			} else {
				ui.Fail("not there yet after %s", elapsed)
				ui.Info("`dojo hint` for a nudge, `dojo reset` to start the scenario over.")
			}
			return nil
		},
	}
}

func recordGrade(a *active, passed bool) (*progress.Attempt, error) {
	if err := requireCompleteSetup(a.State); err != nil {
		return nil, err
	}
	firstPass := passed && !a.State.Passed
	history, err := progress.Load()
	if err != nil {
		return nil, err
	}
	att := history.RecordGrade(a.Lab.ID, passed, firstPass,
		a.State.Elapsed(), a.State.HintsUsed, a.State.SolutionRead)
	if err := history.Save(); err != nil {
		return nil, err
	}
	if firstPass {
		a.State.Passed = true
		if err := config.SaveState(a.State); err != nil {
			return nil, err
		}
	}
	return att, nil
}

func printReport(rep *lab.Report) {
	ui.Heading("Requirements")
	for _, r := range rep.Required {
		printResult(r)
	}
	if len(rep.Any) > 0 {
		ui.Blank()
		ui.Heading("At least one of")
		for _, r := range rep.Any {
			printResult(r)
		}
	}
}

func printResult(r grader.Result) {
	switch {
	case r.Err != nil:
		ui.Warn("%s (could not check: %v)", r.Description, r.Err)
	case r.Passed:
		ui.OK("%s", r.Description)
	case r.Detail != "":
		ui.Fail("%s -- %s", r.Description, r.Detail)
	default:
		ui.Fail("%s", r.Description)
	}
}

func newHintCmd(app *App) *cobra.Command {
	var all bool
	cmd := &cobra.Command{
		Use:   "hint",
		Short: "Reveal the next hint",
		Long: `Hints escalate: the area, then the object, then the diagnostic approach,
then the commands. Each one is recorded, because a pass that needed help is
not the same as a pass that did not.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			lock, err := config.Acquire()
			if err != nil {
				return err
			}
			defer lock.Release()
			a, err := app.loadActive(cmd.Context())
			if err != nil {
				return err
			}
			if a.State.Mode == config.ModeExam {
				return fmt.Errorf("hints are unavailable during an exam")
			}
			hints := a.Plan.Hints
			if len(hints) == 0 {
				return fmt.Errorf("lab %s has no hints", a.Lab.ID)
			}
			if all {
				for i, h := range hints {
					ui.Heading("Hint %d of %d", i+1, len(hints))
					ui.Markdown(h)
				}
				a.State.HintsUsed = len(hints)
				return config.SaveState(a.State)
			}
			if a.State.HintsUsed >= len(hints) {
				ui.Info("That was the last hint. `dojo solution` shows a worked answer.")
				return nil
			}
			idx := a.State.HintsUsed
			ui.Heading("Hint %d of %d", idx+1, len(hints))
			ui.Markdown(hints[idx])
			a.State.HintsUsed++
			return config.SaveState(a.State)
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "show every hint at once")
	return cmd
}

func newSolutionCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "solution",
		Short: "Show a worked solution",
		RunE: func(cmd *cobra.Command, args []string) error {
			lock, err := config.Acquire()
			if err != nil {
				return err
			}
			defer lock.Release()
			a, err := app.loadActive(cmd.Context())
			if err != nil {
				return err
			}
			if a.State.Mode == config.ModeExam {
				return fmt.Errorf("solutions are unavailable during an exam")
			}
			text, err := a.Lab.Solution()
			if err != nil {
				return err
			}
			ui.Markdown(text)
			a.State.SolutionRead = true
			return config.SaveState(a.State)
		},
	}
}

func newResetCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "reset",
		Short: "Rebuild the running lab's scenario",
		Long: `reset repairs what the lab did and builds the scenario again.

It cannot undo everything you might have done to the cluster. If something
still looks wrong afterwards, ` + "`dojo env reset`" + ` rebuilds the whole thing.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			lock, err := config.Acquire()
			if err != nil {
				return err
			}
			defer lock.Release()
			a, err := app.loadActive(cmd.Context())
			if err != nil {
				return err
			}
			history, err := progress.Load()
			if err != nil {
				return err
			}
			a.State.SetupPending = true
			if err := config.SaveState(a.State); err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), setupTimeout)
			defer cancel()
			if err := prepareScenarioEnvironment(ctx, a); err != nil {
				return incompleteSetupError(err)
			}
			scenarioCtx, cancelScenario := context.WithTimeout(ctx, labTimeout)
			defer cancelScenario()
			ui.Step("resetting %s", a.Lab.ID)
			if err := a.Runner.Teardown(scenarioCtx); err != nil {
				return incompleteSetupError(err)
			}
			if err := a.Runner.Setup(scenarioCtx); err != nil {
				return incompleteSetupError(err)
			}
			if err := finishSetup(a, history); err != nil {
				return incompleteSetupError(err)
			}
			ui.OK("scenario rebuilt; the clock restarted")
			return nil
		},
	}
}

func newStopCmd(app *App) *cobra.Command {
	var keep bool
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "End the running lab and clean up its objects",
		RunE: func(cmd *cobra.Command, args []string) error {
			lock, err := config.Acquire()
			if err != nil {
				return err
			}
			defer lock.Release()
			a, err := app.loadActive(cmd.Context())
			if err != nil {
				return err
			}
			if !keep {
				a.State.SetupPending = true
				if err := config.SaveState(a.State); err != nil {
					return err
				}
				ctx, cancel := context.WithTimeout(cmd.Context(), labTimeout)
				defer cancel()
				ui.Step("cleaning up %s", a.Lab.ID)
				if err := a.Runner.Teardown(ctx); err != nil {
					return incompleteSetupError(fmt.Errorf("cleanup was incomplete: %w", err))
				}
			}
			if err := config.ClearState(); err != nil {
				return err
			}
			ui.OK("lab %s ended", a.Lab.ID)
			return nil
		},
	}
	cmd.Flags().BoolVar(&keep, "keep", false, "leave the lab's objects in the cluster")
	return cmd
}

func newStatusCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show what is currently running",
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := config.LoadState()
			if err != nil {
				return err
			}
			if !st.Active() {
				ui.Info("no lab is running")
				ui.Info("`dojo labs` lists what is available.")
				return nil
			}
			cur, err := loadCurriculum(app)
			if err != nil {
				return err
			}
			l, mod := cur.LabByID(st.ActiveLab)
			title, module := st.ActiveLab, "?"
			if l != nil {
				title = l.Title
				module = mod.ID
			}
			ui.Heading("%s", title)
			rows := [][]string{
				{"lab", st.ActiveLab},
				{"module", module},
				{"profile", st.Profile},
				{"mode", string(st.Mode)},
				{"elapsed", st.Elapsed().Round(time.Second).String()},
				{"hints used", strconv.Itoa(st.HintsUsed)},
				{"attempt", strconv.Itoa(st.Attempt)},
				{"seed", strconv.FormatInt(st.Seed, 10)},
			}
			if st.Variant != "" {
				rows = append(rows, []string{"variant", st.Variant})
			}
			if st.Passed {
				rows = append(rows, []string{"graded", "passed"})
			}
			if l != nil && len(l.Checkpoints) > 0 {
				rows = append(rows, []string{"checkpoint", fmt.Sprintf("%d/%d", st.Checkpoint, len(l.Checkpoints))})
			}
			ui.Table(nil, rows)
			if st.SetupPending {
				ui.Warn("scenario setup is incomplete; grading and checkpoints are unavailable")
				ui.Info("Retry with `dojo reset`, clean up with `dojo stop`, or rebuild with `dojo env reset`.")
			}
			return nil
		},
	}
}
