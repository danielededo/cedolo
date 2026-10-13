// Package layout defines the text layout: the positioned text of a document
// that every text layer produces (a Go PDF reader, poppler, pdf.js in the
// browser) and that the extraction core consumes. It is the boundary
// described in ADR 0004: nothing past this package needs to know about PDF.
package layout

import (
	"math"
	"sort"
	"strings"
	"unicode"
)

// Document is the positioned text of one file.
type Document struct {
	Pages []Page `json:"pages"`
}

// Page holds the tokens of one page. Coordinates are in PDF points
// (1/72 inch) with the origin at the top-left corner and Y growing downward,
// which matches reading order and the coordinate system of most UI toolkits.
type Page struct {
	Number int     `json:"number"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
	Tokens []Token `json:"tokens"`
}

// Token is a word: a run of non-space characters printed close together on
// the same baseline. X and Y locate its top-left corner.
type Token struct {
	Text string  `json:"text"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	W    float64 `json:"w"`
	H    float64 `json:"h"`
}

// Right returns the X coordinate of the token's right edge.
func (t Token) Right() float64 { return t.X + t.W }

// Line is a group of tokens sharing approximately the same vertical position,
// sorted left to right.
type Line struct {
	Y      float64
	Tokens []Token
}

// Text joins the tokens of the line with single spaces.
func (l Line) Text() string {
	parts := make([]string, len(l.Tokens))
	for i, t := range l.Tokens {
		parts[i] = t.Text
	}
	return strings.Join(parts, " ")
}

// Lines groups the page tokens into lines. Two tokens belong to the same line
// when their top edges differ by at most tolerance points.
func (p Page) Lines(tolerance float64) []Line {
	tokens := append([]Token(nil), p.Tokens...)
	sort.SliceStable(tokens, func(i, j int) bool {
		if tokens[i].Y != tokens[j].Y {
			return tokens[i].Y < tokens[j].Y
		}
		return tokens[i].X < tokens[j].X
	})
	var lines []Line
	for _, t := range tokens {
		n := len(lines)
		if n > 0 && math.Abs(lines[n-1].Y-t.Y) <= tolerance {
			lines[n-1].Tokens = append(lines[n-1].Tokens, t)
			continue
		}
		lines = append(lines, Line{Y: t.Y, Tokens: []Token{t}})
	}
	for i := range lines {
		sort.SliceStable(lines[i].Tokens, func(a, b int) bool {
			return lines[i].Tokens[a].X < lines[i].Tokens[b].X
		})
	}
	return lines
}

// TokenCount returns the number of tokens across all pages.
func (d Document) TokenCount() int {
	n := 0
	for _, p := range d.Pages {
		n += len(p.Tokens)
	}
	return n
}

// Redact returns a copy of the document in which every letter is replaced by
// x or X and every digit by 0, keeping punctuation, token lengths and
// positions. The result shows the shape of a layout without its content. It
// is a blunt first step: the anonymizer planned in the design document keeps
// known payroll labels through an allowlist, which this function does not.
func (d Document) Redact() Document {
	out := Document{Pages: make([]Page, len(d.Pages))}
	for i, p := range d.Pages {
		rp := p
		rp.Tokens = make([]Token, len(p.Tokens))
		for j, t := range p.Tokens {
			t.Text = redactText(t.Text)
			rp.Tokens[j] = t
		}
		out.Pages[i] = rp
	}
	return out
}

func redactText(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case unicode.IsDigit(r):
			return '0'
		case unicode.IsUpper(r):
			return 'X'
		case unicode.IsLetter(r):
			return 'x'
		default:
			return r
		}
	}, s)
}
