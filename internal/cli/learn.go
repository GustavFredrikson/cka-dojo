package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/gustavfredrikson/cka-dojo/internal/curriculum"
	"github.com/gustavfredrikson/cka-dojo/internal/lab"
	"github.com/gustavfredrikson/cka-dojo/internal/learning"
	"github.com/gustavfredrikson/cka-dojo/internal/progress"
	"github.com/gustavfredrikson/cka-dojo/internal/ui"
	"github.com/spf13/cobra"
)

func newLearnCmd(app *App) *cobra.Command {
	var noPager bool
	cmd := &cobra.Command{
		Use:   "learn [module]",
		Short: "Read a concise lesson before practising",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cur, err := loadCurriculum(app)
			if err != nil {
				return err
			}
			if len(args) == 0 {
				history, err := progress.Load()
				if err != nil {
					return err
				}
				rows := make([][]string, 0, len(cur.Modules))
				var total moduleProgress
				lessonsOpened := 0
				for _, module := range cur.Modules {
					summary := summarizeModule(module, history)
					total.add(summary)
					if history.LessonViewed(module.ID) {
						lessonsOpened++
					}
					rows = append(rows, []string{
						module.ID,
						module.Name,
						moduleLevel(module, history),
						progressFraction(summary.Attempted, summary.Total, ""),
						progressFraction(summary.Passed, summary.Total, "✓"),
						progressFraction(summary.Mastered, summary.Total, "★"),
						strings.Join(module.Domains, ", "),
					})
				}
				ui.Heading("Curriculum %s — Kubernetes %s", cur.ID, cur.Kubernetes.Minor)
				ui.Table([]string{"MODULE", "TOPIC", "LEVEL", "ATTEMPTED", "PASSED", "MASTERED", "DOMAINS"}, rows)
				ui.Blank()
				ui.Info("Overall: %d/%d attempted · %d/%d passed · %d/%d mastered · %d/%d lessons opened",
					total.Attempted, total.Total, total.Passed, total.Total, total.Mastered, total.Total,
					lessonsOpened, len(cur.Modules))
				ui.Blank()
				ui.Info("Read one with `dojo learn <module>`, for example `dojo learn services`.")
				return nil
			}

			module := cur.ModuleByID(args[0])
			if module == nil {
				return fmt.Errorf("no module matches %q; run `dojo learn` to list them", args[0])
			}
			lesson, err := module.Lesson(app.Src)
			if err != nil {
				return err
			}
			history, err := progress.Load()
			if err != nil {
				return err
			}
			history.MarkLesson(module.ID, time.Now())
			if err := history.Save(); err != nil {
				return err
			}
			printLearningPath(module, history)
			ui.Blank()
			if err := ui.PageMarkdown(lesson, noPager); err != nil {
				return fmt.Errorf("show lesson: %w", err)
			}
			summary := summarizeModule(module, history)
			if summary.Total > 0 && summary.Passed == summary.Total {
				ui.Blank()
				ui.OK("All %d %s exercises passed.", summary.Total, module.Name)
				if summary.Mastered < summary.Total {
					ui.Info("%d/%d mastered. `dojo recommend` will choose useful repeat practice.",
						summary.Mastered, summary.Total)
				}
			} else if next := nextExercise(module, history); next != nil {
				ui.Blank()
				ui.Info("Practice next: `dojo start %s`", next.ID)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&noPager, "no-pager", false, "print directly instead of opening less")
	return cmd
}

func printLearningPath(module *curriculum.Module, history *progress.File) {
	ui.Heading("%s learning path", module.Name)
	rows := [][]string{{"✓", "0", "Learn", "mental model", module.Name}}
	for _, exercise := range module.Labs {
		info, _ := exercise.LearningStage.Info()
		rows = append(rows, []string{
			learning.Status(exercise, history),
			fmt.Sprintf("%d", info.Level),
			info.Name,
			exercise.ID,
			exercise.Title,
		})
	}
	ui.Table([]string{"", "LEVEL", "STAGE", "EXERCISE", "TITLE"}, rows)
	summary := summarizeModule(module, history)
	ui.Info("Progress: %d/%d attempted · %d/%d passed · %d/%d mastered",
		summary.Attempted, summary.Total, summary.Passed, summary.Total, summary.Mastered, summary.Total)
	ui.Info("Legend: · available  ↻ attempted  ✓ passed  ★ mastered  🔒 prerequisites incomplete")
}

type moduleProgress struct {
	Attempted int
	Passed    int
	Mastered  int
	Total     int
}

func (p *moduleProgress) add(other moduleProgress) {
	p.Attempted += other.Attempted
	p.Passed += other.Passed
	p.Mastered += other.Mastered
	p.Total += other.Total
}

func summarizeModule(module *curriculum.Module, history *progress.File) moduleProgress {
	summary := moduleProgress{Total: len(module.Labs)}
	for _, exercise := range module.Labs {
		attempt := history.Labs[exercise.ID]
		if attempt == nil {
			continue
		}
		if attempt.Attempts > 0 {
			summary.Attempted++
		}
		if attempt.Passes > 0 {
			summary.Passed++
		}
		if attempt.Mastered() {
			summary.Mastered++
		}
	}
	return summary
}

func progressFraction(done, total int, completeMark string) string {
	if total == 0 {
		return "—"
	}
	value := fmt.Sprintf("%d/%d", done, total)
	if completeMark != "" && done == total {
		return completeMark + " " + value
	}
	return value
}

func nextExercise(module *curriculum.Module, history *progress.File) *lab.Lab {
	for _, exercise := range module.Labs {
		attempt := history.Labs[exercise.ID]
		if (attempt == nil || attempt.Passes == 0) && learning.Unlocked(exercise, history) {
			return exercise
		}
	}
	return nil
}

func moduleLevel(module *curriculum.Module, history *progress.File) string {
	level := -1
	if history.LessonViewed(module.ID) {
		level = 0
	}
	for _, exercise := range module.Labs {
		a := history.Labs[exercise.ID]
		if a == nil || a.Passes == 0 {
			continue
		}
		if info, ok := exercise.LearningStage.Info(); ok && info.Level > level {
			level = info.Level
		}
	}
	if level < 0 {
		return "not started"
	}
	return fmt.Sprintf("%d/7", level)
}
