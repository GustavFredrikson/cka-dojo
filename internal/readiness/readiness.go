// Package readiness answers one question: is the learner ready to sit the
// exam?
//
// It is deliberately harder to satisfy than `dojo progress`. Progress reports
// what has been done; readiness reports what has been *proven*, weighted by
// how much of the exam each domain is worth, and it refuses to convert a pile
// of single passes into confidence.
//
// Three rules shape the whole package:
//
//  1. Only mastery counts. A lab passed once, or passed with a hint, is
//     evidence of exposure, not of competence.
//  2. A domain with no history at all is a blocker, never an average. Scoring
//     an untouched domain as zero and letting a strong domain outweigh it
//     would report a number that reads as "nearly ready" while a tenth of the
//     exam has never been seen.
//  3. The report states what it cannot see. Uncovered curriculum and the
//     absence of timed conditions are printed with the verdict, not hidden
//     behind it.
package readiness

import (
	"fmt"
	"sort"
	"time"

	"github.com/gustavfredrikson/cka-dojo/internal/curriculum"
	"github.com/gustavfredrikson/cka-dojo/internal/progress"
)

// staleAfter is when a mastered lab stops counting as current. Six weeks is
// roughly the point at which an untouched kubectl workflow starts needing the
// documentation again.
const staleAfter = 42 * 24 * time.Hour

// slowFactor is how far past a lab's target time a run has to be before it
// counts against exam pace. Targets are tight by design, so a small overrun
// is normal; 1.5x is not.
const slowFactor = 1.5

// Verdict is the headline answer.
type Verdict string

const (
	VerdictNotReady    Verdict = "not ready"
	VerdictApproaching Verdict = "approaching ready"
	VerdictLikelyReady Verdict = "likely ready"
)

// Domain is one published exam domain, scored.
type Domain struct {
	ID        string
	Name      string
	Weight    int
	Labs      int
	Attempted int
	Passed    int
	Mastered  int
	// Stale counts mastered labs last practised beyond staleAfter.
	Stale int
	// Coverage is mastered/labs in 0..1.
	Coverage float64
}

// Untouched reports a domain the learner has never attempted at all.
func (d *Domain) Untouched() bool { return d.Attempted == 0 }

// Report is the whole assessment.
type Report struct {
	Verdict Verdict
	// Score is the exam-weighted share of mastered content, 0..1. It is a
	// measure of the curriculum that exists, not of the exam: read it
	// alongside Uncovered.
	Score   float64
	Domains []Domain

	// HintDependent are labs passed at least once but never cleanly.
	HintDependent []string
	// Slow are labs whose most recent run took more than slowFactor times
	// their target, with the seconds taken and the target seconds.
	Slow []Pace
	// Unproven are labs passed exactly once. They are the cheapest available
	// route to a higher score, so they are called out separately.
	Unproven []string
	// Failing are labs attempted but never passed.
	Failing []string
	// Uncovered are published competencies with no lab behind them, taken
	// from the curriculum's knownGaps list.
	Uncovered []curriculum.Gap
}

// Pace is one lab that ran long.
type Pace struct {
	Lab     string
	Seconds int
	Target  int
}

// Assess scores history against the curriculum at now.
func Assess(cur *curriculum.Curriculum, history *progress.File, now time.Time) *Report {
	byDomain := map[string]*Domain{}
	order := make([]string, 0, len(cur.Domains))
	for _, d := range cur.Domains {
		byDomain[d.ID] = &Domain{ID: d.ID, Name: d.Name, Weight: d.Weight}
		order = append(order, d.ID)
	}

	r := &Report{Uncovered: cur.KnownGaps}

	for _, l := range cur.Labs() {
		attempt := history.Labs[l.ID]
		for _, id := range l.Domains {
			d := byDomain[id]
			if d == nil {
				// A lab tagged with a domain the curriculum does not declare
				// is a content bug, not a readiness signal. content validate
				// catches it; here it is simply not counted.
				continue
			}
			d.Labs++
			if attempt == nil || attempt.Attempts == 0 {
				continue
			}
			d.Attempted++
			if attempt.Passes > 0 {
				d.Passed++
			}
			if attempt.Mastered() {
				d.Mastered++
				if now.Sub(attempt.LastAt) > staleAfter {
					d.Stale++
				}
			}
		}

		if attempt == nil || attempt.Attempts == 0 {
			continue
		}
		switch {
		case attempt.Passes == 0:
			r.Failing = append(r.Failing, l.ID)
		case attempt.CleanPasses == 0:
			r.HintDependent = append(r.HintDependent, l.ID)
		case attempt.Passes == 1:
			r.Unproven = append(r.Unproven, l.ID)
		}
		if target := l.TargetMinutes * 60; target > 0 {
			if spent := attempt.LastSeconds; spent > int(float64(target)*slowFactor) {
				r.Slow = append(r.Slow, Pace{Lab: l.ID, Seconds: spent, Target: target})
			}
		}
	}

	for _, id := range order {
		d := byDomain[id]
		if d.Labs > 0 {
			// Stale mastery counts for half. It is real evidence that has
			// aged, which is different from never having been shown.
			effective := float64(d.Mastered) - 0.5*float64(d.Stale)
			d.Coverage = effective / float64(d.Labs)
		}
		r.Domains = append(r.Domains, *d)
	}

	sort.Strings(r.Failing)
	sort.Strings(r.HintDependent)
	sort.Strings(r.Unproven)
	sort.Slice(r.Slow, func(i, j int) bool { return r.Slow[i].Lab < r.Slow[j].Lab })

	r.Score = score(r.Domains)
	r.Verdict = verdict(r)
	return r
}

// score is the exam-weighted mean of per-domain coverage. Domains with no
// labs are excluded rather than counted as complete, so an unwritten module
// cannot inflate the result.
func score(domains []Domain) float64 {
	var num, den float64
	for _, d := range domains {
		if d.Labs == 0 {
			continue
		}
		num += d.Coverage * float64(d.Weight)
		den += float64(d.Weight)
	}
	if den == 0 {
		return 0
	}
	return num / den
}

// verdict converts the score into words, with two hard gates that no amount
// of strength elsewhere can buy past: an untouched domain, and a domain whose
// coverage is far below the rest.
func verdict(r *Report) Verdict {
	for _, d := range r.Domains {
		if d.Labs > 0 && d.Untouched() {
			return VerdictNotReady
		}
	}
	switch {
	case r.Score >= 0.75:
		for _, d := range r.Domains {
			if d.Labs > 0 && d.Coverage < 0.5 {
				return VerdictApproaching
			}
		}
		return VerdictLikelyReady
	case r.Score >= 0.4:
		return VerdictApproaching
	default:
		return VerdictNotReady
	}
}

// NextSteps ranks the cheapest routes to a higher score, most valuable first.
// It exists so the report ends with an instruction rather than a number.
func (r *Report) NextSteps() []string {
	var out []string

	var untouched []Domain
	for _, d := range r.Domains {
		if d.Labs > 0 && d.Untouched() {
			untouched = append(untouched, d)
		}
	}
	sort.Slice(untouched, func(i, j int) bool { return untouched[i].Weight > untouched[j].Weight })
	for _, d := range untouched {
		out = append(out, fmt.Sprintf(
			"Start %s: %d labs, %d%% of the exam, no history at all.", d.Name, d.Labs, d.Weight))
	}

	if n := len(r.Unproven); n > 0 {
		out = append(out, fmt.Sprintf(
			"Re-run the %d labs passed exactly once. Mastery needs a second clean pass, "+
				"and this is the cheapest score you can buy.", n))
	}
	if n := len(r.Failing); n > 0 {
		out = append(out, fmt.Sprintf("Finish the %d labs attempted but never passed.", n))
	}
	if n := len(r.HintDependent); n > 0 {
		out = append(out, fmt.Sprintf(
			"Redo the %d labs never passed without help. A hinted pass is exposure, not competence.", n))
	}
	if n := len(r.Slow); n > 0 {
		out = append(out, fmt.Sprintf(
			"Work on pace: %d labs last ran over %.1fx their target.", n, slowFactor))
	}

	var stale int
	for _, d := range r.Domains {
		stale += d.Stale
	}
	if stale > 0 {
		out = append(out, fmt.Sprintf(
			"Refresh %d mastered labs untouched for more than six weeks.", stale))
	}
	return out
}
