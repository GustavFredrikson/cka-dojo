// Package recommend ranks labs using the published exam weights and the
// learner's own history. The formula is intentionally small enough to explain
// at the terminal; recommendations should never feel arbitrary.
package recommend

import (
	"sort"
	"time"

	"github.com/gustavfredrikson/cka-dojo/internal/curriculum"
	"github.com/gustavfredrikson/cka-dojo/internal/lab"
	"github.com/gustavfredrikson/cka-dojo/internal/learning"
	"github.com/gustavfredrikson/cka-dojo/internal/progress"
)

// Recommendation is one ranked lab with the factors that produced its score.
type Recommendation struct {
	Lab           *lab.Lab
	Module        *curriculum.Module
	Score         float64
	ExamWeight    float64
	MasteryGap    float64
	Recency       float64
	StageFactor   float64
	Unlocked      bool
	Reason        string
	LastPracticed time.Time
}

// Rank returns every lab from most to least useful at now.
//
// Score = average published domain weight x mastery gap x recency x stage.
//
// Average domain weight avoids giving a lab a free multiplier just because it
// is tagged with two domains. Mastery gap prioritises a failed attempt above a
// brand-new lab, then an incomplete pass, then a mastered lab. Recency starts
// low immediately after practice and reaches its full value after two weeks.
func Rank(cur *curriculum.Curriculum, history *progress.File, now time.Time) []Recommendation {
	weights := make(map[string]int, len(cur.Domains))
	for _, d := range cur.Domains {
		weights[d.ID] = d.Weight
	}

	var out []Recommendation
	for _, module := range cur.Modules {
		for _, exercise := range module.Labs {
			attempt := history.Labs[exercise.ID]
			examWeight := averageWeight(exercise.Domains, weights)
			gap, reason := gapFor(attempt)
			recency := recencyFor(attempt, now)
			r := Recommendation{
				Lab: exercise, Module: module,
				ExamWeight: examWeight, MasteryGap: gap, Recency: recency,
				StageFactor: stageFactor(exercise.LearningStage),
				Unlocked:    learning.Unlocked(exercise, history),
				Reason:      reason,
			}
			if attempt != nil {
				r.LastPracticed = attempt.LastAt
			}
			if r.Unlocked {
				r.Score = r.ExamWeight * r.MasteryGap * r.Recency * r.StageFactor
			} else {
				r.Reason = "prerequisites incomplete"
			}
			out = append(out, r)
		}
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Score == out[j].Score {
			return out[i].Lab.ID < out[j].Lab.ID
		}
		return out[i].Score > out[j].Score
	})
	return out
}

// stageFactor makes the default recommendation climb the learning ladder
// before it offers blind diagnosis. It is a gentle preference, not a lock;
// exam weight and demonstrated gaps still matter.
func stageFactor(stage lab.LearningStage) float64 {
	switch stage {
	case lab.StageFollow:
		return 1.40
	case lab.StageBuild:
		return 1.20
	case lab.StageInspect:
		return 1.05
	case lab.StageGuidedFix:
		return 1.00
	case lab.StageContextualFix:
		return 0.90
	case lab.StageDiagnose:
		return 0.75
	case lab.StageExam:
		return 0.60
	default:
		return 1.00
	}
}

func averageWeight(domains []string, weights map[string]int) float64 {
	if len(domains) == 0 {
		return 1
	}
	total := 0
	for _, domain := range domains {
		total += weights[domain]
	}
	return float64(total) / float64(len(domains))
}

func gapFor(a *progress.Attempt) (float64, string) {
	switch {
	case a == nil || a.Attempts == 0:
		return 1.0, "not attempted"
	case a.Passes == 0:
		return 1.25, "attempted but not passed"
	case !a.Mastered():
		return 0.75, "passed, not yet mastered"
	default:
		return 0.20, "mastered; spaced review"
	}
}

func recencyFor(a *progress.Attempt, now time.Time) float64 {
	if a == nil || a.LastAt.IsZero() {
		return 1.25
	}
	days := now.Sub(a.LastAt).Hours() / 24
	if days < 0 {
		days = 0
	}
	if days > 14 {
		days = 14
	}
	// 0.75 now, rising linearly to 1.25 after fourteen days.
	return 0.75 + days*(0.50/14.0)
}
