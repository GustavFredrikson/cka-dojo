package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/gustavfredrikson/cka-dojo/internal/curriculum"
	"github.com/gustavfredrikson/cka-dojo/internal/progress"
	"github.com/gustavfredrikson/cka-dojo/internal/recommend"
	"github.com/gustavfredrikson/cka-dojo/internal/ui"
	"github.com/spf13/cobra"
)

func newRecommendCmd(app *App) *cobra.Command {
	var all bool
	cmd := &cobra.Command{
		Use:   "recommend",
		Short: "Choose the most useful lab to practise next",
		Long: `Recommendations use four visible factors: the published exam weight,
how far the exercise is from mastery, how long it has been since practice, and
its place in the learning ladder. Locked exercises are never recommended.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cur, err := loadCurriculum(app)
			if err != nil {
				return err
			}
			history, err := progress.Load()
			if err != nil {
				return err
			}
			ranked := recommend.Rank(cur, history, time.Now())
			if len(ranked) == 0 {
				return fmt.Errorf("curriculum %s has no labs to recommend", cur.ID)
			}

			if all {
				rows := make([][]string, 0, len(ranked))
				for _, r := range ranked {
					path := "ready"
					if !r.Unlocked {
						path = "locked"
					}
					rows = append(rows, []string{
						r.Lab.ID,
						fmt.Sprintf("%.1f", r.Score),
						fmt.Sprintf("%.1f", r.ExamWeight),
						fmt.Sprintf("%.2f", r.MasteryGap),
						fmt.Sprintf("%.2f", r.Recency),
						fmt.Sprintf("%.2f", r.StageFactor),
						path,
						r.Reason,
					})
				}
				ui.Table([]string{"LAB", "SCORE", "WEIGHT", "GAP", "RECENCY", "STAGE", "PATH", "WHY"}, rows)
				return nil
			}

			r := ranked[0]
			ui.Heading("Recommended next: %s", r.Lab.Title)
			ui.Info("Lab:    %s", r.Lab.ID)
			ui.Info("Module: %s", r.Module.Name)
			ui.Info("Why:    %s; exam weight %.1f", r.Reason, r.ExamWeight)
			ui.Info("Skills: %s", joinNames(cur, r.Lab.Skills))
			ui.Blank()
			ui.Info("Start it with `dojo start %s`.", r.Lab.ID)
			ui.Info("See the full ranking with `dojo recommend --all`.")
			return nil
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "show the complete ranking and its factors")
	return cmd
}

func joinNames(cur *curriculum.Curriculum, ids []string) string {
	names := make([]string, len(ids))
	for i, id := range ids {
		names[i] = cur.SkillName(id)
	}
	return strings.Join(names, ", ")
}
