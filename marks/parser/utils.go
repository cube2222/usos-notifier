package parser

import (
	"sort"

	"github.com/cube2222/usos-notifier/marks"
)

// The scores are sorted by name alphabetically
func MakeClassWithScores(classID string, className string, scores map[string]*Score) marks.Class {
	out := marks.Class{
		ClassHeader: marks.ClassHeader{
			ID:   classID,
			Name: className,
		},
		Scores: make([]marks.Score, 0, len(scores)),
	}

	for name, score := range scores {
		out.Scores = append(out.Scores, marks.Score{
			Name:    name,
			Unknown: score.Unknown,
			Hidden:  score.Hidden,
			Actual:  score.Actual,
			Max:     score.Max,
		})
	}
	sort.Slice(out.Scores, func(i, j int) bool {
		return out.Scores[i].Name < out.Scores[j].Name
	})

	return out
}
