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
	padLeft     = 64.0
	padRight    = 88.0 // room for end labels; 16 without them
	padTop      = 16.0
	padBottom   = 32.0
	barMaxWidth = 24.0
	surfaceGap  = 2.0
	barRadius   = 4.0
)

// size is the drawing size of a chart. Wide charts scale down on a phone
// until their text is too small to read, so the columns charts are drawn
// twice, wide and narrow, and the page shows the one that fits.
type size struct {
	w, h float64
	name string // "wide" or "narrow", a class of the chart
}

var (
	wide   = size{chartW, chartH, "wide"}
	narrow = size{360, 240, "narrow"}
)

// series is one line or one stacked category; its color is a CSS custom
// property so that light and dark themes swap in one place.
type series struct {
	Name   string
	Color  string // e.g. "var(--series-1)"
	Values []int64
}

// niceMax returns a round upper bound and step for an axis in hundredths
// (cents, or hundredths of an hour or day).
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

// whole renders an axis tick in whole units with Italian thousands
// separators: "3.000". Ticks are round, so nothing is lost.
func whole(hundredths int64) string {
	return strings.TrimSuffix(money.Format(hundredths/100*100), ",00")
}

// euros renders an axis tick as whole euros: "3.000 €".
func euros(cents int64) string { return whole(cents) + " €" }

type plot struct {
	labels   []string
	top      int64
	step     int64
	w, h     float64
	padRight float64
	tick     func(int64) string
}

func (p plot) x(i int) float64 {
	n := len(p.labels)
	inner := p.w - padLeft - p.padRight
	if n == 1 {
		return padLeft + inner/2
	}
	return padLeft + inner*float64(i)/float64(n-1)
}

// band returns the center of column i when each month gets an equal band.
func (p plot) band(i int) float64 {
	inner := p.w - padLeft - p.padRight
	return padLeft + inner*(float64(i)+0.5)/float64(len(p.labels))
}

func (p plot) y(v int64) float64 {
	return padTop + (p.h-padTop-padBottom)*(1-float64(v)/float64(p.top))
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
		fmt.Fprintf(b, `<line class="%s" x1="%.1f" x2="%.1f" y1="%.1f" y2="%.1f"/>`, cls, padLeft, p.w-p.padRight, y, y)
		fmt.Fprintf(b, `<text class="tick" x="%.1f" y="%.1f" text-anchor="end">%s</text>`, padLeft-8, y+4, p.tick(v))
	}
	// Labels need about 60 units each: with many months only every k-th is
	// printed, counting back from the latest.
	n := len(p.labels)
	k := max(1, int(math.Ceil(float64(n)*60/(p.w-padLeft-p.padRight))))
	for i, l := range p.labels {
		if (n-1-i)%k != 0 {
			continue
		}
		cls := "tick month"
		// The latest label ends at its point when there is no room after it.
		anchor := "middle"
		if i == n-1 && xs(i)+30 > p.w {
			anchor = "end"
		}
		fmt.Fprintf(b, `<text class="%s" data-i="%d" x="%.1f" y="%.1f" text-anchor="%s">%s</text>`, cls, i, xs(i), p.h-10, anchor, html.EscapeString(l))
	}
}

// lineOptions vary a line chart: its height, how ticks and values read, and
// whether the last value is labeled at the end of each line.
type lineOptions struct {
	width     float64
	height    float64
	tick      func(int64) string
	value     func(int64) string
	endLabels bool
}

// lineChart draws one line per series over the months, with an end dot and,
// if asked, a direct label at the last value of each series. Every value is
// also in the data attributes read by the hover layer.
func lineChart(id string, labels []string, ss []series, o lineOptions) string {
	var maxV int64
	for _, s := range ss {
		for _, v := range s.Values {
			maxV = max(maxV, v)
		}
	}
	top, step := niceMax(maxV)
	p := plot{labels: labels, top: top, step: step, w: o.width, h: o.height, padRight: 16, tick: o.tick}
	if o.endLabels {
		p.padRight = padRight
	}
	var b strings.Builder
	fmt.Fprintf(&b, `<svg id="%s" class="chart line" viewBox="0 0 %.0f %.0f" role="img" aria-labelledby="%s-title" data-x="%s" data-points="%s">`,
		id, o.width, o.height, id, xsAttr(len(labels), p.x), html.EscapeString(pointsJSON(labels, ss, o.value, nil)))
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
	if o.endLabels {
		for si, s := range ss {
			if last := len(s.Values) - 1; last >= 0 {
				fmt.Fprintf(&b, `<text class="end-label" x="%.1f" y="%.1f">%s</text>`, p.x(last)+10, lastYs[si]+4, o.value(s.Values[last]))
			}
		}
	}
	fmt.Fprintf(&b, `<line class="crosshair" x1="0" x2="0" y1="%.1f" y2="%.1f" visibility="hidden"/>`, padTop, o.height-padBottom)
	fmt.Fprintf(&b, `<rect class="hit" x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="transparent"/>`, padLeft-20, padTop, o.width-padLeft-p.padRight+40, o.height-padTop-padBottom)
	b.WriteString(`</svg>`)
	return b.String()
}

// stackCharts draws a columns chart in both sizes.
func stackCharts(id string, labels []string, ss []series, total series) string {
	return stackChart(id, labels, ss, total, wide) + stackChart(id+"-narrow", labels, ss, total, narrow)
}

// stackChart draws one column per month, its segments stacked from the
// first series up with a surface gap between them and a rounded top. The
// column's height is the sum of its segments, given as totals for the
// hover layer. A transparent band over each month is its hit target.
func stackChart(id string, labels []string, ss []series, total series, sz size) string {
	var maxV int64
	for _, v := range total.Values {
		maxV = max(maxV, v)
	}
	top, step := niceMax(maxV)
	p := plot{labels: labels, top: top, step: step, w: sz.w, h: sz.h, padRight: 16, tick: euros}
	n := len(labels)
	slot := (sz.w - padLeft - p.padRight) / float64(n)
	w := math.Min(barMaxWidth, slot*0.6)
	var b strings.Builder
	fmt.Fprintf(&b, `<svg id="%s" class="chart stack %s" viewBox="0 0 %.0f %.0f" role="img" aria-labelledby="%s-title" data-points="%s">`,
		id, sz.name, sz.w, sz.h, id, html.EscapeString(pointsJSON(labels, ss, moneyValue, &total)))
	p.axes(&b, p.band)
	for i := range labels {
		x := p.band(i) - w/2
		fmt.Fprintf(&b, `<g class="col" data-i="%d">`, i)
		var cum int64
		var drawn []int // indexes of the segments that show
		for si, s := range ss {
			if s.Values[i] > 0 {
				drawn = append(drawn, si)
			}
		}
		for k, si := range drawn {
			v := ss[si].Values[i]
			y0, y1 := p.y(cum), p.y(cum+v)
			cum += v
			if k > 0 {
				y0 -= surfaceGap // the gap sits on the segment below
			}
			if y0-y1 < 0.5 {
				continue
			}
			if k == len(drawn)-1 {
				fmt.Fprintf(&b, `<path style="fill:%s" d="%s"/>`, ss[si].Color, roundedTop(x, y1, w, y0-y1, barRadius))
			} else {
				fmt.Fprintf(&b, `<rect style="fill:%s" x="%.1f" y="%.1f" width="%.1f" height="%.1f"/>`, ss[si].Color, x, y1, w, y0-y1)
			}
		}
		b.WriteString(`</g>`)
	}
	for i := range labels {
		fmt.Fprintf(&b, `<rect class="col-hit" data-i="%d" x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="transparent"/>`,
			i, p.band(i)-slot/2, padTop, slot, sz.h-padTop)
	}
	b.WriteString(`</svg>`)
	return b.String()
}

// roundedTop returns the path of a rectangle whose two top corners are
// rounded and whose bottom stays square on the baseline.
func roundedTop(x, y, w, h, r float64) string {
	r = math.Min(r, math.Min(w/2, h))
	return fmt.Sprintf("M%.1f %.1fV%.1fQ%.1f %.1f %.1f %.1fH%.1fQ%.1f %.1f %.1f %.1fV%.1fZ",
		x, y+h, y+r, x, y, x+r, y, x+w-r, x+w, y, x+w, y+r, y+h)
}

func moneyValue(c int64) string { return money.Format(c) + " €" }

func xsAttr(n int, x func(int) float64) string {
	parts := make([]string, n)
	for i := range parts {
		parts[i] = fmt.Sprintf("%.1f", x(i))
	}
	return strings.Join(parts, ",")
}

// pointsJSON lists, per month, the label, each series' formatted value and,
// when given, the total, for the hover layer. A single series with a total
// gives the total alone.
func pointsJSON(labels []string, ss []series, format func(int64) string, total *series) string {
	type value struct {
		Name  string `json:"name"`
		Color string `json:"color,omitempty"`
		Value string `json:"value"`
	}
	type point struct {
		Label  string  `json:"label"`
		Total  *value  `json:"total,omitempty"`
		Values []value `json:"values"`
	}
	pts := make([]point, len(labels))
	for i, l := range labels {
		pts[i].Label = l
		if total != nil {
			pts[i].Total = &value{Name: total.Name, Value: format(total.Values[i])}
			if len(ss) == 1 {
				continue // the total says it all
			}
		}
		for _, s := range ss {
			pts[i].Values = append(pts[i].Values, value{s.Name, s.Color, format(s.Values[i])})
		}
	}
	out, _ := json.Marshal(pts)
	return string(out)
}
