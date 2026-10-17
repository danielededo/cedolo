package extract

import (
	"regexp"
	"strconv"

	"github.com/danielededo/cedolo/internal/payslip"
)

var numericPeriod = regexp.MustCompile(`^(0?[1-9]|1[0-2]|13|14)[/.-]((?:19|20)\d\d)$`)

// findPeriod returns the first "month year" in reading order, or a numeric
// "mm/yyyy" on a line that is, or is just below, a period label. Some
// payroll software numbers the additional months 13 and 14: "13/2025" is
// the thirteenth month, paid in December, and "14/2025" the fourteenth,
// whose month of payment the period does not tell.
func findPeriod(lines []line) (year, month int, kind payslip.Kind) {
	for i, l := range lines {
		for k, w := range l.words {
			m, ok := monthNames[w.text]
			if !ok || k+1 >= len(l.words) {
				continue
			}
			if y, err := strconv.Atoi(l.words[k+1].text); err == nil && y >= 1990 && y <= 2099 {
				return y, m, ""
			}
		}
		labelled := l.has(periodLabels...) || (i > 0 && lines[i-1].has(periodLabels...))
		if !labelled {
			continue
		}
		for _, t := range l.toks {
			if m := numericPeriod.FindStringSubmatch(t.Text); m != nil {
				mo, _ := strconv.Atoi(m[1])
				y, _ := strconv.Atoi(m[2])
				switch mo {
				case 13:
					return y, 12, payslip.Thirteenth
				case 14:
					return y, 0, payslip.Fourteenth
				}
				return y, mo, ""
			}
		}
	}
	// "13MA MENS. 2025": an additional month named with its year only.
	for _, l := range lines {
		for _, c := range []struct {
			words []string
			kind  payslip.Kind
			month int
		}{{thirteenthWords, payslip.Thirteenth, 12}, {fourteenthWords, payslip.Fourteenth, 0}} {
			for _, p := range c.words {
				_, last, ok := l.find(p)
				if !ok {
					continue
				}
				for _, t := range l.toks[last+1:] {
					if y, err := strconv.Atoi(t.Text); err == nil && y >= 1990 && y <= 2099 {
						return y, c.month, c.kind
					}
				}
			}
		}
	}
	return 0, 0, ""
}
