package cli

import (
	"fmt"
	"strings"

	"github.com/gustavfredrikson/cka-dojo/internal/environment"
	"github.com/gustavfredrikson/cka-dojo/internal/fault"
	"github.com/gustavfredrikson/cka-dojo/internal/grader"
	"github.com/gustavfredrikson/cka-dojo/internal/ui"
	"github.com/spf13/cobra"
)

func newContentCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "content",
		Short: "Inspect and validate the training content",
		Long: `Content commands make adding curriculum safe.

validate catches the mistakes that would otherwise surface as a confusing
failure halfway through a lab: unknown fault or grader types, missing
manifests, labs with no hints, skills that are not declared.`,
	}
	cmd.AddCommand(newContentValidateCmd(app), newContentListCmd(app), newContentTypesCmd())
	return cmd
}

func newContentValidateCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Check every profile, module and lab",
		RunE: func(cmd *cobra.Command, args []string) error {
			problems := 0
			report := func(where string, errs []error) {
				for _, err := range errs {
					problems++
					ui.Fail("%s: %v", where, err)
				}
			}

			profileIDs, err := environment.ListProfiles(app.Src)
			if err != nil {
				return err
			}
			knownProfiles := map[string]bool{}
			for _, id := range profileIDs {
				if _, err := environment.LoadProfile(app.Src, id); err != nil {
					report("environments/"+id, []error{err})
					continue
				}
				knownProfiles[id] = true
			}
			ui.OK("%d environment profile(s): %s", len(knownProfiles), strings.Join(profileIDs, ", "))

			cur, err := loadCurriculum(app)
			if err != nil {
				return err
			}
			report("curriculum "+cur.ID, cur.Validate())

			domains := cur.DomainSet()
			skills := cur.SkillSet()
			labCount := 0
			for _, m := range cur.Modules {
				report("module "+m.ID, m.ValidateLesson(app.Src))
				for _, l := range m.Labs {
					labCount++
					report("lab "+l.ID, l.Validate(domains, skills, knownProfiles))
				}
			}

			ui.OK("%d module(s), %d lab(s) checked", len(cur.Modules), labCount)
			if problems > 0 {
				return fmt.Errorf("%d content problem(s)", problems)
			}
			ui.OK("content is valid")
			return nil
		},
	}
}

func newContentListCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List modules and their labs",
		RunE: func(cmd *cobra.Command, args []string) error {
			cur, err := loadCurriculum(app)
			if err != nil {
				return err
			}
			ui.Heading("%s (%s, Kubernetes %s)", cur.ID, cur.Certification.Name, cur.Kubernetes.Minor)
			ui.Blank()
			for _, m := range cur.Modules {
				lesson := ""
				if !m.HasLesson(app.Src) {
					lesson = "  (no lesson yet)"
				}
				ui.Heading("%s -- %s%s", m.ID, m.Name, lesson)
				for _, l := range m.Labs {
					variants := ""
					if l.Variants != nil && len(l.Variants.Options) > 0 {
						variants = fmt.Sprintf("  [%d variants]", len(l.Variants.Options))
					}
					ui.Info("  %-28s %s%s", l.ID, l.Title, variants)
				}
				if len(m.Labs) == 0 {
					ui.Info("  (no labs yet)")
				}
				ui.Blank()
			}
			return nil
		},
	}
}

func newContentTypesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "types",
		Short: "List the fault and grader types a lab may use",
		RunE: func(cmd *cobra.Command, args []string) error {
			ui.Heading("Fault types")
			for _, t := range fault.Types() {
				ui.Info("  %s", t)
			}
			ui.Blank()
			ui.Heading("Grader types")
			for _, t := range grader.Types() {
				ui.Info("  %s", t)
			}
			return nil
		},
	}
}
