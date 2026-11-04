package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/danielededo/cedolo/internal/ccnl"
	"github.com/danielededo/cedolo/internal/report"
)

func TestRun(t *testing.T) {
	asOf, err := time.Parse("2006-01-02", ccnl.AsOf())
	if err != nil {
		t.Fatal(err)
	}
	due := func(now time.Time) string {
		var b bytes.Buffer
		if err := run(&b, now); err != nil {
			t.Fatal(err)
		}
		return b.String()
	}
	if got := due(asOf); got != "" {
		t.Errorf("nothing due the day of the export, got %q", got)
	}
	later := time.Date(report.FiguresCheckedThrough+1, 1, 2, 0, 0, 0, 0, time.UTC)
	if !strings.Contains(due(later), "Check the figures set by law") || strings.Contains(due(later.AddDate(-1, 0, 0)), "Check the figures") {
		t.Errorf("figures due only once a year passes the one checked")
	}
	got := due(time.Date(max(report.FiguresCheckedThrough, asOf.Year())+1, 12, 31, 0, 0, 0, 0, time.UTC))
	if !strings.Contains(got, "Check the figures set by law") || !strings.Contains(got, "Refresh the CNEL contract list") {
		t.Errorf("both due at the end of the next year, got %q", got)
	}
	for _, line := range strings.Split(strings.TrimSpace(got), "\n") {
		if len(strings.Split(line, "\t")) != 2 {
			t.Errorf("line is not a title and a body: %q", line)
		}
	}
}
