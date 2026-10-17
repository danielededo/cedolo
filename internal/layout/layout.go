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
	"unicode/utf8"
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

// JoinSpaced rejoins words that a PDF prints one or two characters at a
// time, each placed on its own with a little extra spacing: a text layer
// then reports "1", ".", "2", "3", "4" instead of "1.234". Two tokens on the
// same line are joined when the gap between them is well below a space,
// under a fifth of the text height, and their heights match. Only pages
// where most tokens are single characters are touched: elsewhere tokens
// that touch are separate cells, and the page is returned as it is.
func JoinSpaced(doc Document) Document {
	out := Document{Pages: make([]Page, len(doc.Pages))}
	for i, p := range doc.Pages {
		if !spaced(p.Tokens) {
			out.Pages[i] = p
			continue
		}
		np := p
		np.Tokens = nil
		for _, l := range p.Lines(0.25 * medianHeight(p.Tokens)) {
			var cur Token
			for k, t := range l.Tokens {
				if k > 0 && joinable(cur, t) {
					right := math.Max(cur.Right(), t.Right())
					bottom := math.Max(cur.Y+cur.H, t.Y+t.H)
					cur.Text += t.Text
					cur.Y = math.Min(cur.Y, t.Y)
					cur.W = right - cur.X
					cur.H = bottom - cur.Y
					continue
				}
				if k > 0 {
					np.Tokens = append(np.Tokens, cur)
				}
				cur = t
			}
			if len(l.Tokens) > 0 {
				np.Tokens = append(np.Tokens, cur)
			}
		}
		out.Pages[i] = np
	}
	return out
}

// joinable tells whether b continues the word a ends: on the same baseline,
// as tall, and closer than a space would be. Some text layers leave the
// extra spacing out of each character's box, so the gaps inside an amount
// look wider than the spaces between words; a decimal or thousands
// separator next to a digit is joined across a wider gap.
func joinable(a, b Token) bool {
	h := math.Max(a.H, b.H)
	if h <= 0 || a.H < 0.8*b.H || b.H < 0.8*a.H {
		return false
	}
	limit := 0.2 * h
	if separates(a.Text, b.Text) {
		limit = 0.5 * h
	}
	gap := b.X - a.Right()
	return gap > -0.1*h && gap < limit && math.Abs((a.Y+a.H)-(b.Y+b.H)) < 0.2*h
}

// separates tells whether a and b meet inside a number: a digit then "," or
// ".", or a separator or a minus sign then a digit.
func separates(a, b string) bool {
	last, _ := utf8.DecodeLastRuneInString(a)
	first, _ := utf8.DecodeRuneInString(b)
	sep := func(r rune) bool { return r == ',' || r == '.' }
	return unicode.IsDigit(last) && (b == "," || b == ".") ||
		(sep(last) || a == "-") && unicode.IsDigit(first)
}

// spaced tells whether a page prints its words a character at a time: at
// least two tokens in five hold a single character.
func spaced(toks []Token) bool {
	single := 0
	for _, t := range toks {
		if utf8.RuneCountInString(t.Text) == 1 {
			single++
		}
	}
	return len(toks) >= 20 && single*5 >= len(toks)*2
}

func medianHeight(toks []Token) float64 {
	if len(toks) == 0 {
		return 0
	}
	hs := make([]float64, len(toks))
	for i, t := range toks {
		hs[i] = t.H
	}
	sort.Float64s(hs)
	return hs[len(hs)/2]
}
