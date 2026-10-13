package pdftext

import (
	"math"

	"github.com/danielededo/cedolo/internal/layout"
)

// Agreement summarizes how closely two readers' outputs match on the same
// file. It carries counts and distances only, never text, so it can be shared
// without exposing the document.
type Agreement struct {
	Matched   int     // tokens with identical text and boxes within tolerance
	OnlyA     int     // tokens found only by the first reader
	OnlyB     int     // tokens found only by the second reader
	MaxOffset float64 // largest box corner distance among matched tokens
}

// Ratio returns matched tokens over the union of tokens, between 0 and 1.
func (a Agreement) Ratio() float64 {
	total := a.Matched + a.OnlyA + a.OnlyB
	if total == 0 {
		return 1
	}
	return float64(a.Matched) / float64(total)
}

// Compare pairs tokens of a and b page by page: a pair needs identical text
// and bottom-left corners at most tolerance points apart. The bottom edge is
// used because readers disagree on the ascent of fonts that are not embedded
// in the file, while baselines and descents agree closely.
func Compare(a, b layout.Document, tolerance float64) Agreement {
	var ag Agreement
	pages := max(len(a.Pages), len(b.Pages))
	for i := 0; i < pages; i++ {
		var ta, tb []layout.Token
		if i < len(a.Pages) {
			ta = a.Pages[i].Tokens
		}
		if i < len(b.Pages) {
			tb = b.Pages[i].Tokens
		}
		used := make([]bool, len(tb))
		for _, x := range ta {
			best, bestDist := -1, math.Inf(1)
			for j, y := range tb {
				if used[j] || x.Text != y.Text {
					continue
				}
				if d := math.Hypot(x.X-y.X, (x.Y+x.H)-(y.Y+y.H)); d <= tolerance && d < bestDist {
					best, bestDist = j, d
				}
			}
			if best < 0 {
				ag.OnlyA++
				continue
			}
			used[best] = true
			ag.Matched++
			ag.MaxOffset = math.Max(ag.MaxOffset, bestDist)
		}
		for _, u := range used {
			if !u {
				ag.OnlyB++
			}
		}
	}
	return ag
}
