// Package insights compares payslips over time and reports the changes worth
// a look, each with the most likely benign explanation. It never says a
// payslip is wrong: a change is a reason to read the payslip, not an error.
package insights

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/danielededo/cedolo/internal/i18n"
	"github.com/danielededo/cedolo/internal/money"
	"github.com/danielededo/cedolo/internal/payslip"
)

// Change is one difference between a payslip and the one before it.
type Change struct {
	// Code identifies the kind of change, such as "net" or "item_new".
	Code string `json:"code"`
	// Text describes the change and Why gives likely explanations, both in
	// the requested language.
	Text string   `json:"text"`
	Why  []string `json:"why,omitempty"`
	// Key is the taxonomy key of the item a change is about, if any.
	Key string `json:"key,omitempty"`
}

// Period is the payslip a list of changes refers to.
type Period struct {
	Year    int          `json:"year"`
	Month   int          `json:"month"`
	Kind    payslip.Kind `json:"kind"`
	Label   string       `json:"label"`
	Changes []Change     `json:"changes"`
}

// Thresholds above which a variation is reported.
const (
	relativeThreshold = 0.10 // net or gross, as a fraction of the previous value
	taxRateThreshold  = 3.0  // percentage points of gross
	inpsRateThreshold = 0.05 // percentage points of the INPS base
)

// Sort orders payslips by period; additional months follow the monthly
// payslip of the same month, and payslips without a period go last.
func Sort(slips []payslip.Payslip) {
	order := func(p payslip.Payslip) (int, int, int) {
		if p.Month == 0 && p.Year != 0 {
			// An additional month known only by its year: the thirteenth
			// is paid in December, the fourteenth usually in July.
			switch p.Kind {
			case payslip.Thirteenth:
				return p.Year, 12, 1
			case payslip.Fourteenth:
				return p.Year, 7, 1
			}
		}
		if p.Month == 0 {
			return math.MaxInt32, 0, 0
		}
		k := 0
		if p.Kind != payslip.Monthly {
			k = 1
		}
		return p.Year, p.Month, k
	}
	sort.SliceStable(slips, func(i, j int) bool {
		yi, mi, ki := order(slips[i])
		yj, mj, kj := order(slips[j])
		if yi != yj {
			return yi < yj
		}
		if mi != mj {
			return mi < mj
		}
		return ki < kj
	})
}

// Dedupe drops payslips that repeat an earlier one: same period and kind,
// and the same gross, deductions and net. The same file given twice, or a
// payslip printed again, would otherwise count twice in comparisons and
// charts. It keeps the order of the rest and returns how many were dropped.
func Dedupe(slips []payslip.Payslip) ([]payslip.Payslip, int) {
	type key struct {
		year, month     int
		kind            payslip.Kind
		gross, ded, net int64
	}
	seen := map[key]bool{}
	out := slips[:0:0]
	for _, p := range slips {
		if (p.Month == 0 && (p.Kind == payslip.Monthly || p.Year == 0)) || !p.Net.Found() {
			out = append(out, p)
			continue
		}
		k := key{p.Year, p.Month, p.Kind, p.Gross.Cents, p.Deductions.Cents, p.Net.Cents}
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, p)
	}
	return out, len(slips) - len(out)
}

// Label names a payslip's period.
func Label(l i18n.Lang, p payslip.Payslip) string {
	if p.Month == 0 && p.Year != 0 && p.Kind != payslip.Monthly {
		return i18n.T(l, "period."+string(p.Kind)+".year", p.Year)
	}
	if p.Month == 0 {
		return i18n.T(l, "period.unknown")
	}
	switch p.Kind {
	case payslip.Thirteenth:
		return i18n.T(l, "period.13th", i18n.Month(l, p.Month), p.Year)
	case payslip.Fourteenth:
		return i18n.T(l, "period.14th", i18n.Month(l, p.Month), p.Year)
	}
	return i18n.T(l, "period", i18n.Month(l, p.Month), p.Year)
}

// Compare sorts the payslips and returns, for each one, its changes from the
// previous ordinary monthly payslip. Additional months (13th, 14th) are
// compared with nothing: their amounts are not comparable with a month.
func Compare(l i18n.Lang, slips []payslip.Payslip) []Period {
	Sort(slips)
	var out []Period
	var prev *payslip.Payslip
	for i := range slips {
		cur := &slips[i]
		per := Period{Year: cur.Year, Month: cur.Month, Kind: cur.Kind, Label: Label(l, *cur), Changes: []Change{}}
		if cur.Kind == payslip.Monthly && cur.Month != 0 {
			if prev != nil {
				per.Changes = append(per.Changes, changes(l, *prev, *cur)...)
			}
			prev = cur
		}
		if inconsistent(*cur) {
			per.Changes = append(per.Changes, Change{Code: "unchecked", Text: i18n.T(l, "change.unchecked")})
		}
		out = append(out, per)
	}
	return out
}

func inconsistent(p payslip.Payslip) bool {
	for _, a := range []payslip.Amount{p.Gross, p.Deductions, p.Net} {
		if a.Status == payslip.Inconsistent {
			return true
		}
	}
	return false
}

func changes(l i18n.Lang, prev, cur payslip.Payslip) []Change {
	var out []Change
	add := func(code, text string, why ...string) {
		out = append(out, Change{Code: code, Text: text, Why: why})
	}

	if gap := (cur.Year*12 + cur.Month) - (prev.Year*12 + prev.Month); gap > 1 {
		add("gap", i18n.T(l, "change.gap", Label(l, prev), Label(l, cur)))
	}
	if prev.CCNL != "" && cur.CCNL != "" && prev.CCNL != cur.CCNL {
		add("ccnl", i18n.T(l, "change.ccnl", prev.CCNL, cur.CCNL), i18n.T(l, "why.different_employ"))
	}
	if prev.Level != "" && cur.Level != "" && prev.Level != cur.Level {
		add("level", i18n.T(l, "change.level", prev.Level, cur.Level), i18n.T(l, "why.new_contract"))
	}

	// Explanations shared by pay variations, from what changed in the items.
	var reasons []string
	newKeys, goneKeys := itemKeys(prev, cur), itemKeys(cur, prev)
	if newKeys["earnings.thirteenth"] || cur.Month == 12 && has(cur, "earnings.thirteenth") {
		reasons = append(reasons, i18n.T(l, "why.thirteenth"))
	}
	if newKeys["earnings.fourteenth"] {
		reasons = append(reasons, i18n.T(l, "why.fourteenth"))
	}
	if sumKey(prev, "earnings.overtime") != sumKey(cur, "earnings.overtime") {
		reasons = append(reasons, i18n.T(l, "why.overtime"))
	}
	if cur.Month == 12 || cur.Month == 2 || cur.Month == 3 {
		reasons = append(reasons, i18n.T(l, "why.year_end"))
	}

	for _, v := range []struct {
		code      string
		prev, cur payslip.Amount
		textKey   string
	}{
		{"net", prev.Net, cur.Net, "change.net"},
		{"gross", prev.Gross, cur.Gross, "change.gross"},
	} {
		if !v.prev.Found() || !v.cur.Found() || v.prev.Cents == 0 {
			continue
		}
		rel := float64(v.cur.Cents-v.prev.Cents) / float64(v.prev.Cents)
		if math.Abs(rel) > relativeThreshold {
			add(v.code, i18n.T(l, v.textKey, money.Format(v.prev.Cents), money.Format(v.cur.Cents), percent(l, rel*100, 1, true)), reasons...)
		}
	}

	for _, key := range sortedKeys(newKeys) {
		var why []string
		switch key {
		case "tax.surcharge.regional", "tax.surcharge.municipal":
			why = append(why, i18n.T(l, "why.surcharge_start"))
		case "earnings.thirteenth":
			why = append(why, i18n.T(l, "why.thirteenth"))
		case "earnings.fourteenth":
			why = append(why, i18n.T(l, "why.fourteenth"))
		}
		add("item_new", i18n.T(l, "change.item_new", i18n.ItemLabel(l, key), money.Format(sumKey(cur, key))), why...)
		out[len(out)-1].Key = key
	}
	for _, key := range sortedKeys(goneKeys) {
		var why []string
		if key == "tax.surcharge.regional" || key == "tax.surcharge.municipal" {
			why = append(why, i18n.T(l, "why.surcharge_end"))
		}
		add("item_gone", i18n.T(l, "change.item_gone", i18n.ItemLabel(l, key)), why...)
		out[len(out)-1].Key = key
	}

	if pr, cr, ok := rates(taxes(prev), prev.Gross, taxes(cur), cur.Gross); ok && math.Abs(cr-pr) > taxRateThreshold {
		var why []string
		if sumKey(prev, "pension_fund.employee") != sumKey(cur, "pension_fund.employee") {
			why = append(why, i18n.T(l, "why.pension_fund"))
		}
		if cur.Month == 12 || cur.Month == 2 || cur.Month == 3 || prev.Month == 12 || cur.Adjustment || prev.Adjustment {
			why = append(why, i18n.T(l, "why.year_end"))
		}
		add("tax_rate", i18n.T(l, "change.tax_rate", percent(l, pr, 1, false), percent(l, cr, 1, false)), why...)
	}
	if pr, cr, ok := rates(inps(prev), prev.INPSBase, inps(cur), cur.INPSBase); ok && math.Abs(cr-pr) > inpsRateThreshold {
		add("inps_rate", i18n.T(l, "change.inps_rate", percent(l, pr, 2, false), percent(l, cr, 2, false)), i18n.T(l, "why.inps_threshold"))
	}

	for _, lc := range cur.Leave {
		for _, lp := range prev.Leave {
			if lp.Kind != lc.Kind || lp.Remaining == nil || lc.Remaining == nil || *lp.Remaining == *lc.Remaining {
				continue
			}
			delta := *lc.Remaining - *lp.Remaining
			var why []string
			if delta < 0 {
				why = append(why, i18n.T(l, "why.leave_used"))
			}
			sign := "+"
			if delta < 0 {
				sign = "-"
				delta = -delta
			}
			add("leave", i18n.T(l, "change.leave", lc.Label, money.Format(*lp.Remaining), money.Format(*lc.Remaining),
				sign+money.Format(delta)), why...)
		}
	}
	return out
}

// itemKeys returns the pay item keys of b that a does not have, ignoring
// information items and roundings.
func itemKeys(a, b payslip.Payslip) map[string]bool {
	inA := map[string]bool{}
	for _, it := range a.Items {
		inA[it.Key] = true
	}
	out := map[string]bool{}
	for _, it := range b.Items {
		if it.Side == payslip.Info || it.Key == "rounding" || inA[it.Key] {
			continue
		}
		out[it.Key] = true
	}
	return out
}

func has(p payslip.Payslip, key string) bool {
	_, n := p.SumKey(key)
	return n > 0
}

func sumKey(p payslip.Payslip, key string) int64 {
	s, _ := p.SumKey(key)
	return s
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// taxes returns income taxes withheld: the summary box when printed,
// otherwise the IRPEF and surcharge items.
func taxes(p payslip.Payslip) int64 {
	if p.Taxes.Found() {
		return p.Taxes.Cents
	}
	return sumKey(p, "tax.irpef") + sumKey(p, "tax.surcharge.regional") + sumKey(p, "tax.surcharge.municipal")
}

// inps returns the INPS contribution: items, summary box, or social total.
func inps(p payslip.Payslip) int64 {
	if s, n := p.SumKey("deductions.inps"); n > 0 {
		return s
	}
	if p.INPS.Found() {
		return p.INPS.Cents
	}
	return p.SocialContributions.Cents
}

// rates returns two amounts as percentages of their bases.
func rates(prevValue int64, prevBase payslip.Amount, curValue int64, curBase payslip.Amount) (float64, float64, bool) {
	if !prevBase.Found() || !curBase.Found() || prevBase.Cents == 0 || curBase.Cents == 0 || prevValue == 0 || curValue == 0 {
		return 0, 0, false
	}
	return 100 * float64(prevValue) / float64(prevBase.Cents), 100 * float64(curValue) / float64(curBase.Cents), true
}

// percent formats a percentage with the language's decimal mark: "-30,3"
// in Italian, "-30.3" in English; signed adds a plus to a rise.
func percent(l i18n.Lang, v float64, decimals int, signed bool) string {
	s := strconv.FormatFloat(v, 'f', decimals, 64)
	if signed && v >= 0 {
		s = "+" + s
	}
	if l == i18n.IT {
		s = strings.Replace(s, ".", ",", 1)
	}
	return s
}

// String renders a change for terminals.
func (c Change) String() string {
	s := c.Text
	for _, w := range c.Why {
		s += fmt.Sprintf("\n    %s", w)
	}
	return s
}
