package lab

import "fmt"

// LearningStage is the kind of support an exercise gives the learner. It is
// independent of technical difficulty: following an advanced kubelet
// walkthrough and blindly diagnosing an easy selector typo are different axes.
type LearningStage string

const (
	StageFollow        LearningStage = "follow"
	StageBuild         LearningStage = "build"
	StageInspect       LearningStage = "inspect"
	StageGuidedFix     LearningStage = "guided-fix"
	StageContextualFix LearningStage = "contextual-fix"
	StageDiagnose      LearningStage = "diagnose"
	StageExam          LearningStage = "exam"
)

// StageInfo is the learner-facing progression metadata.
type StageInfo struct {
	Level int
	Name  string
}

var stages = map[LearningStage]StageInfo{
	StageFollow:        {Level: 1, Name: "Follow"},
	StageBuild:         {Level: 2, Name: "Build"},
	StageInspect:       {Level: 3, Name: "Inspect"},
	StageGuidedFix:     {Level: 4, Name: "Fix, guided"},
	StageContextualFix: {Level: 5, Name: "Fix, contextual"},
	StageDiagnose:      {Level: 6, Name: "Diagnose"},
	StageExam:          {Level: 7, Name: "Exam"},
}

// Info returns the stage's level and display name.
func (s LearningStage) Info() (StageInfo, bool) {
	info, ok := stages[s]
	return info, ok
}

// Validate rejects missing or unknown stage names.
func (s LearningStage) Validate() error {
	if _, ok := s.Info(); !ok {
		return fmt.Errorf("unknown learningStage %q; want follow, build, inspect, guided-fix, contextual-fix, diagnose or exam", s)
	}
	return nil
}

// NeedsHints reports whether progressive hints are part of this stage. Follow,
// build and inspect tasks provide scaffolding in the task itself; exam tasks
// deliberately provide none.
func (s LearningStage) NeedsHints() bool {
	return s == StageGuidedFix || s == StageContextualFix || s == StageDiagnose
}
