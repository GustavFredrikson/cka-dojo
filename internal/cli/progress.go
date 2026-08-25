package cli

import (
	"fmt"
	"sort"
	"time"

	"github.com/gustavfredrikson/cka-dojo/internal/curriculum"
	"github.com/gustavfredrikson/cka-dojo/internal/progress"
	"github.com/gustavfredrikson/cka-dojo/internal/ui"
	"github.com/spf13/cobra"
)

func newProgressCmd(app *App) *cobra.Command {
	var byLab bool
	cmd := &cobra.Command{
		Use:   "progress",
		Short: "Show what you have mastered and what you have not",
		Long: `Progress is reported by skill rather than by lab.

"26 of 50 labs complete" says nothing about whether you can fix a Service
under time pressure. A skill counts as mastered only when every lab touching
it has been passed twice, the most recent time without hints.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cur, err := loadCurriculum(app)
			if err != nil {
				return err
			}
			prog, err := progress.Load()
			if err != nil {
				return err
			}

			if byLab {
				return printLabProgress(cur, prog)
			}

			labSkills := map[string][]string{}
			for _, l := range cur.Labs() {
				labSkills[l.ID] = l.Skills
			}
			stats := prog.BySkill(labSkills)

			// BySkill sorts by id; sort by the name actually shown instead.
			sort.Slice(stats, func(i, j int) bool {
				return cur.SkillName(stats[i].Skill) < cur.SkillName(stats[j].Skill)
			})
			rows := make([][]string, 0, len(stats))
			for _, s := range stats {
				rows = append(rows, []string{
					cur.SkillName(s.Skill),
					fmt.Sprintf("%d", s.Attempts),
					fmt.Sprintf("%d", s.CleanPasses),
					fmt.Sprintf("%d/%d", s.MasteredLab, s.Labs),
					formatSeconds(s.BestSeconds),
				})
			}
			ui.Heading("Skills")
			ui.Table([]string{"SKILL", "ATTEMPTS", "CLEAN PASSES", "MASTERED", "BEST"}, rows)

			ui.Blank()
			total, mastered := 0, 0
			for _, l := range cur.Labs() {
				total++
				if a := prog.Labs[l.ID]; a != nil && a.Mastered() {
					mastered++
				}
			}
			ui.Info("%d of %d labs mastered. `dojo progress --by-lab` for the detail.", mastered, total)
			return nil
		},
	}
	cmd.Flags().BoolVar(&byLab, "by-lab", false, "show per-lab history instead of skills")
	return cmd
}

// printLabProgress shows the raw history, including the seed of the last
// attempt so a scenario that beat you can be replayed exactly.
func printLabProgress(cur *curriculum.Curriculum, prog *progress.File) error {
	rows := [][]string{}
	for _, l := range cur.Labs() {
		a := prog.Labs[l.ID]
		if a == nil {
			rows = append(rows, []string{l.ID, "new", "-", "-", "-", "-"})
			continue
		}
		rows = append(rows, []string{
			l.ID,
			statusMark(a),
			fmt.Sprintf("%d", a.Attempts),
			fmt.Sprintf("%d", a.Passes),
			formatSeconds(a.BestSeconds),
			seedOf(a),
		})
	}
	ui.Heading("Labs")
	ui.Table([]string{"LAB", "STATE", "ATTEMPTS", "PASSES", "BEST", "LAST SEED"}, rows)
	ui.Blank()
	ui.Info("Replay a scenario with `dojo start <lab> --seed <seed>`.")
	return nil
}

func seedOf(a *progress.Attempt) string {
	if a.LastSeed == 0 {
		return "-"
	}
	return fmt.Sprintf("%d", a.LastSeed)
}

func formatSeconds(s int) string {
	if s <= 0 {
		return "-"
	}
	d := time.Duration(s) * time.Second
	return fmt.Sprintf("%d:%02d", int(d.Minutes()), s%60)
}
