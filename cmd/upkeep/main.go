// Command upkeep lists what is due to keep cedolo's built-in data current:
// the figures set by law once a year passes the one they were checked
// through, and the CNEL contract list once its export is old. Each line is
// a title, a tab and a body, for a scheduled job to open an issue with;
// nothing is printed when nothing is due.
//
//	go run ./cmd/upkeep
package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/danielededo/cedolo/internal/ccnl"
	"github.com/danielededo/cedolo/internal/report"
)

// listAge is how old the CNEL export may get before a refresh is due.
const listAge = 120 * 24 * time.Hour

func main() {
	if err := run(os.Stdout, time.Now()); err != nil {
		fmt.Fprintln(os.Stderr, "upkeep:", err)
		os.Exit(1)
	}
}

func run(out io.Writer, now time.Time) error {
	if y := now.Year(); y > report.FiguresCheckedThrough {
		fmt.Fprintf(out, "Check the figures set by law for %d\tThe report's figures set by law were last checked for %d. Check them against the law in force for %d, add a row to `yearly` in internal/report/analytics.go where one changed, and set `FiguresCheckedThrough` to %d.\n",
			y, report.FiguresCheckedThrough, y, y)
	}
	asOf, err := time.Parse("2006-01-02", ccnl.AsOf())
	if err != nil {
		return fmt.Errorf("date of the CNEL list: %w", err)
	}
	if now.Sub(asOf) > listAge {
		fmt.Fprintf(out, "Refresh the CNEL contract list\tThe built-in list of national collective agreements is the CNEL export of %s. Download the current one and rebuild internal/ccnl/ccnl.tsv as cmd/ccnlgen describes.\n",
			ccnl.AsOf())
	}
	return nil
}
