// Package learning evaluates ordered curriculum paths without coupling the
// content model to CLI presentation.
package learning

import (
	"fmt"

	"github.com/gustavfredrikson/cka-dojo/internal/lab"
	"github.com/gustavfredrikson/cka-dojo/internal/progress"
)

// MissingRequirement explains one prerequisite that has not been met.
type MissingRequirement struct {
	LabID   string
	Mastery bool
}

func (r MissingRequirement) String() string {
	if r.Mastery {
		return fmt.Sprintf("%s (mastery)", r.LabID)
	}
	return fmt.Sprintf("%s (one pass)", r.LabID)
}

// Missing returns unmet pass and mastery gates for an exercise.
func Missing(exercise *lab.Lab, history *progress.File) []MissingRequirement {
	var missing []MissingRequirement
	for _, id := range exercise.Prerequisites {
		a := history.Labs[id]
		if a == nil || a.Passes == 0 {
			missing = append(missing, MissingRequirement{LabID: id})
		}
	}
	for _, id := range exercise.MasteryPrerequisites {
		a := history.Labs[id]
		if a == nil || !a.Mastered() {
			missing = append(missing, MissingRequirement{LabID: id, Mastery: true})
		}
	}
	return missing
}

// Unlocked reports whether the default path would offer this exercise now.
func Unlocked(exercise *lab.Lab, history *progress.File) bool {
	return len(Missing(exercise, history)) == 0
}

// Status is the compact marker used in a learning-path display.
func Status(exercise *lab.Lab, history *progress.File) string {
	a := history.Labs[exercise.ID]
	switch {
	case a != nil && a.Mastered():
		return "★"
	case a != nil && a.Passes > 0:
		return "✓"
	case !Unlocked(exercise, history):
		return "🔒"
	case a != nil && a.Attempts > 0:
		return "↻"
	default:
		return "·"
	}
}
