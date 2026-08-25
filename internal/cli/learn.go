package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/gustavfredrikson/cka-dojo/internal/curriculum"
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
				for _, module := range cur.Modules {
					rows = append(rows, []string{
						module.ID,
						module.Name,
						moduleLevel(module, history),
						fmt.Sprintf("%d", len(module.Labs)),
						strings.Join(module.Domains, ", "),
					})
				}
				ui.Heading("Curriculum %s — Kubernetes %s", cur.ID, cur.Kubernetes.Minor)
				ui.Table([]string{"MODULE", "TOPIC", "LEVEL", "LABS", "DOMAINS"}, rows)
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
			if len(module.Labs) > 0 {
				ui.Blank()
				ui.Info("Practice next: `dojo start %s`", module.Labs[0].ID)
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
	ui.Info("Legend: · available  ↻ attempted  ✓ passed  ★ mastered  🔒 prerequisites incomplete")
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
