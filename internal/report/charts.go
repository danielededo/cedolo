package report

import (
	"encoding/json"
	"fmt"
	"html"
	"math"
	"strings"

	"github.com/danielededo/cedolo/internal/money"
)

// Chart geometry, in SVG user units. Charts scale to their container.
const (
	chartW      = 680.0
	chartH      = 260.0
	padLeft     = 56.0
	padRight    = 88.0 // room for end labels
	padTop      = 16.0
	padBottom   = 32.0
	barMaxWidth = 24.0
	surfaceGap  = 2.0
)

// series is one line or one stacked category; its color is a CSS custom
// property so that light and dark themes swap in one place.
type series struct {
	Name   string
	Color  string // e.g. "var(--series-1)"
	Values []int64
}

// niceMax returns a round upper bound and step for a money axis in cents.
func niceMax(v int64) (top, step int64) {
	if v <= 0 {
		return 10000, 2500
	}
	raw := float64(v) / 4
	mag := math.Pow(10, math.Floor(math.Log10(raw)))
	for _, m := range []float64{1, 2, 2.5, 5, 10} {
		if raw <= m*mag {
			step = int64(m * mag)
			break
		}
	}
	top = step * int64(math.Ceil(float64(v)/float64(step)))
	return top, step
}

// compact renders an axis tick in euros, with thousands as "1,5K".
func compact(cents int64) string {
	euros := cents / 100
	if euros < 1000 {
		return fmt.Sprintf("%d", euros)
	}
	whole, tenth := euros/1000, euros%1000/100
	if tenth == 0 {
		return fmt.Sprintf("%dK", whole)
	}
	return fmt.Sprintf("%d,%dK", whole, tenth)
}

type plot struct {
	labels []string
	top    int64
	step   int64
}

func (p plot) x(i int) float64 {
	n := len(p.labels)
	inner := chartW - padLeft - padRight
	if n == 1 {
		return padLeft + inner/2
	}
	return padLeft + inner*float64(i)/float64(n-1)
}

func (p plot) band() float64 {
	return (chartW - padLeft - padRight) / float64(max(len(p.labels), 1))
}

func (p plot) xBand(i int) float64 {
	return padLeft + p.band()*(float64(i)+0.5)
}

func (p plot) y(v int64) float64 {
	return padTop + (chartH-padTop-padBottom)*(1-float64(v)/float64(p.top))
}

// axes draws the recessive horizontal gridlines with their ticks and the
// month labels under the given x positions.
func (p plot) axes(b *strings.Builder, xs func(int) float64) {
	for v := int64(0); v <= p.top; v += p.step {
		y := p.y(v)
		cls := "grid"
		if v == 0 {
			cls = "baseline"
		}
		fmt.Fprintf(b, `<line class="%s" x1="%.1f" x2="%.1f" y1="%.1f" y2="%.1f"/>`, cls, padLeft, chartW-padRight, y, y)
		fmt.Fprintf(b, `<text class="tick" x="%.1f" y="%.1f" text-anchor="end">%s</text>`, padLeft-8, y+4, compact(v))
	}
	n := len(p.labels)
	for i, l := range p.labels {
		// On narrow screens every other month label is hidden, keeping
		// the latest one.
		cls := "tick"
		if (n-1-i)%2 == 1 {
			cls += " thin"
		}
		fmt.Fprintf(b, `<text class="%s" x="%.1f" y="%.1f" text-anchor="middle">%s</text>`, cls, xs(i), chartH-10, html.EscapeString(l))
	}
}

// lineChart draws one line per series over the months, with an end dot and
// a direct label at the last value of each series. Every value is also in
// the data attributes read by the hover layer and in the report's table.
func lineChart(id string, labels []string, ss []series) string {
	var maxV int64
	for _, s := range ss {
		for _, v := range s.Values {
			maxV = max(maxV, v)
		}
	}
	top, step := niceMax(maxV)
	p := plot{labels: labels, top: top, step: step}
	var b strings.Builder
	fmt.Fprintf(&b, `<svg id="%s" class="chart line" viewBox="0 0 %.0f %.0f" role="img" aria-labelledby="%s-title" data-x="%s" data-points="%s">`,
		id, chartW, chartH, id, xsAttr(p, len(labels), p.x), html.EscapeString(pointsJSON(labels, ss)))
	p.axes(&b, p.x)
	lastYs := make([]float64, len(ss))
	for si, s := range ss {
		var pts []string
		for i, v := range s.Values {
			pts = append(pts, fmt.Sprintf("%.1f,%.1f", p.x(i), p.y(v)))
		}
		if len(pts) > 1 {
			fmt.Fprintf(&b, `<polyline class="series-line" style="stroke:%s" points="%s"/>`, s.Color, strings.Join(pts, " "))
		}
		last := len(s.Values) - 1
		if last >= 0 {
			lastYs[si] = p.y(s.Values[last])
			fmt.Fprintf(&b, `<circle class="dot" style="fill:%s" cx="%.1f" cy="%.1f" r="4"/>`, s.Color, p.x(last), lastYs[si])
		}
	}
	// End labels, nudged apart when they would collide.
	if len(ss) == 2 && math.Abs(lastYs[0]-lastYs[1]) < 14 {
		if lastYs[0] < lastYs[1] {
			lastYs[0], lastYs[1] = lastYs[0]-7, lastYs[1]+7
		} else {
			lastYs[0], lastYs[1] = lastYs[0]+7, lastYs[1]-7
		}
	}
	for si, s := range ss {
		last := len(s.Values) - 1
		if last < 0 {
			continue
		}
		fmt.Fprintf(&b, `<text class="end-label" x="%.1f" y="%.1f">%s</text>`, p.x(last)+10, lastYs[si]+4, money.Format(s.Values[last]))
	}
	fmt.Fprintf(&b, `<line class="crosshair" x1="0" x2="0" y1="%.1f" y2="%.1f" visibility="hidden"/>`, padTop, chartH-padBottom)
	fmt.Fprintf(&b, `<rect class="hit" x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="transparent"/>`, padLeft-20, padTop, chartW-padLeft-padRight+40, chartH-padTop-padBottom)
	b.WriteString(`</svg>`)
	return b.String()
}

// stackedBars draws one column per month, its segments stacked in series
// order and separated by a surface gap, with a rounded top.
func stackedBars(id string, labels []string, ss []series) string {
	var maxV int64
	for i := range labels {
		var total int64
		for _, s := range ss {
			total += s.Values[i]
		}
		maxV = max(maxV, total)
	}
	top, step := niceMax(maxV)
	p := plot{labels: labels, top: top, step: step}
	w := math.Min(barMaxWidth, p.band()*0.6)
	var b strings.Builder
	fmt.Fprintf(&b, `<svg id="%s" class="chart bars" viewBox="0 0 %.0f %.0f" role="img" aria-labelledby="%s-title">`, id, chartW, chartH, id)
	p.axes(&b, p.xBand)
	for i := range labels {
		x := p.xBand(i) - w/2
		var acc int64
		// The topmost non-empty segment gets the rounded end.
		topSeg := -1
		for si, s := range ss {
			if s.Values[i] > 0 {
				topSeg = si
			}
		}
		for si, s := range ss {
			v := s.Values[i]
			if v <= 0 {
				continue
			}
			y0, y1 := p.y(acc), p.y(acc+v)
			acc += v
			h := y0 - y1
			if si != topSeg {
				h -= surfaceGap // the gap above each segment but the last
			}
			if h <= 0 {
				continue
			}
			data := fmt.Sprintf(`data-month="%s" data-series="%s" data-value="%s"`,
				html.EscapeString(labels[i]), html.EscapeString(s.Name), money.Format(v)+" €")
			if si == topSeg {
				r := math.Min(4, h)
				fmt.Fprintf(&b, `<path class="seg" tabindex="0" style="fill:%s" %s d="M%.1f %.1f V%.1f Q%.1f %.1f %.1f %.1f H%.1f Q%.1f %.1f %.1f %.1f V%.1f Z"/>`,
					s.Color, data, x, y0, y1+r, x, y1, x+r, y1, x+w-r, x+w, y1, x+w, y1+r, y0)
			} else {
				fmt.Fprintf(&b, `<rect class="seg" tabindex="0" style="fill:%s" %s x="%.1f" y="%.1f" width="%.1f" height="%.1f"/>`,
					s.Color, data, x, y0-h, w, h)
			}
		}
	}
	b.WriteString(`</svg>`)
	return b.String()
}

func xsAttr(p plot, n int, x func(int) float64) string {
	parts := make([]string, n)
	for i := range parts {
		parts[i] = fmt.Sprintf("%.1f", x(i))
	}
	return strings.Join(parts, ",")
}

// pointsJSON lists, per month, the label and each series' formatted value,
// for the hover layer.
func pointsJSON(labels []string, ss []series) string {
	type value struct {
		Name  string `json:"name"`
		Color string `json:"color"`
		Value string `json:"value"`
	}
	type point struct {
		Label  string  `json:"label"`
		Values []value `json:"values"`
	}
	pts := make([]point, len(labels))
	for i, l := range labels {
		pts[i].Label = l
		for _, s := range ss {
			pts[i].Values = append(pts[i].Values, value{s.Name, s.Color, money.Format(s.Values[i]) + " €"})
		}
	}
	out, _ := json.Marshal(pts)
	return string(out)
}
