package cli

import (
	"fmt"
	"strings"

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
				rows := make([][]string, 0, len(cur.Modules))
				for _, module := range cur.Modules {
					rows = append(rows, []string{
						module.ID,
						module.Name,
						fmt.Sprintf("%d", len(module.Labs)),
						strings.Join(module.Domains, ", "),
					})
				}
				ui.Heading("Curriculum %s — Kubernetes %s", cur.ID, cur.Kubernetes.Minor)
				ui.Table([]string{"MODULE", "TOPIC", "LABS", "DOMAINS"}, rows)
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
