package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/danielededo/cedolo/internal/layout"
)

func synthFile(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := runSynth([]string{"-o", path}); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestInspectPrintsNoContent(t *testing.T) {
	path := synthFile(t, "cedolino_BIANCHI_ANNA.pdf")
	var out bytes.Buffer
	if err := runInspect([]string{path}, &out); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "text layer readable") {
		t.Errorf("unexpected verdict:\n%s", got)
	}
	for _, secret := range []string{"BIANCHI", "ANNA", "1.667,40", path} {
		if strings.Contains(got, secret) {
			t.Errorf("inspect output leaks %q:\n%s", secret, got)
		}
	}
}

func TestInspectMissingFileHidesPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ROSSI_MARIO.pdf")
	var out bytes.Buffer
	if err := runInspect([]string{path}, &out); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "ROSSI") {
		t.Errorf("error output leaks the file name:\n%s", out.String())
	}
}

func TestLayoutRedact(t *testing.T) {
	path := synthFile(t, "example.pdf")
	var out, warn bytes.Buffer
	if err := runLayout([]string{"-redact", path}, &out, &warn); err != nil {
		t.Fatal(err)
	}
	if warn.Len() != 0 {
		t.Errorf("unexpected warning for redacted output: %s", warn.String())
	}
	var doc layout.Document
	if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	if doc.TokenCount() == 0 || strings.Contains(out.String(), "BIANCHI") {
		t.Errorf("redacted layout is empty or leaks content")
	}

	out.Reset()
	if err := runLayout([]string{path}, &out, &warn); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(warn.String(), "personal data") {
		t.Error("unredacted layout printed without a warning")
	}
}

func TestSynthWritesPDF(t *testing.T) {
	data, err := os.ReadFile(synthFile(t, "s.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Error("not a PDF")
	}
}

func TestVerdict(t *testing.T) {
	cases := []struct {
		s    docStats
		want string
	}{
		{docStats{pages: 1}, "no text layer"},
		{docStats{tokens: 10, chars: 100, unreadable: 6, amounts: 3}, "cannot be decoded"},
		{docStats{tokens: 10, chars: 100}, "no amounts"},
		{docStats{tokens: 10, chars: 100, amounts: 2}, "text layer readable"},
	}
	for _, c := range cases {
		if got := c.s.verdict(); !strings.Contains(got, c.want) {
			t.Errorf("verdict(%+v) = %q, want it to contain %q", c.s, got, c.want)
		}
	}
}

func TestExtractEveryStyle(t *testing.T) {
	for _, style := range []string{"classic", "signed", "grid"} {
		path := filepath.Join(t.TempDir(), style+".pdf")
		if err := runSynth([]string{"-style", style, "-o", path}); err != nil {
			t.Fatal(err)
		}
		var out bytes.Buffer
		if err := runExtract([]string{path}, &out); err != nil {
			t.Fatal(err)
		}
		got := out.String()
		if !strings.Contains(got, "October 2026 (monthly)") || strings.Contains(got, "FAIL") || strings.Contains(got, "inconsistent") {
			t.Errorf("%s: unexpected extraction:\n%s", style, got)
		}
		out.Reset()
		if err := runExtract([]string{"-json", path}, &out); err != nil {
			t.Fatal(err)
		}
		var ps []map[string]any
		if err := json.Unmarshal(out.Bytes(), &ps); err != nil || len(ps) != 1 {
			t.Errorf("%s: JSON output: %v, %d payslips", style, err, len(ps))
		}
	}
}

func TestAnonymizeCommand(t *testing.T) {
	path := synthFile(t, "anon.pdf")
	var out, info bytes.Buffer
	if err := runAnonymize([]string{path}, &out, &info); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "BIANCHI") || strings.Contains(out.String(), "1.667,40") {
		t.Error("anonymized output leaks personal data")
	}
	if !strings.Contains(out.String(), "TOTALE") || !strings.Contains(info.String(), "Review the output") {
		t.Errorf("unexpected output; info: %s", info.String())
	}
	var doc layout.Document
	if err := json.Unmarshal(out.Bytes(), &doc); err != nil || doc.TokenCount() == 0 {
		t.Errorf("output is not a layout: %v", err)
	}
}

func TestCompareCommand(t *testing.T) {
	dir := t.TempDir()
	// Two months from the synthetic generator; the second in another style
	// with the same content apart from the period.
	a := filepath.Join(dir, "a.pdf")
	b := filepath.Join(dir, "b.pdf")
	if err := runSynth([]string{"-style", "classic", "-o", a}); err != nil {
		t.Fatal(err)
	}
	if err := runSynth([]string{"-style", "signed", "-o", b}); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := runCompare([]string{"-lang", "en", a, b}, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "October 2026") {
		t.Errorf("unexpected output:\n%s", out.String())
	}
	out.Reset()
	if err := runCompare([]string{"-json", a}, &out); err != nil {
		t.Fatal(err)
	}
	var periods []map[string]any
	if err := json.Unmarshal(out.Bytes(), &periods); err != nil || len(periods) != 1 {
		t.Errorf("JSON output: %v, %d periods", err, len(periods))
	}
	if err := runCompare([]string{"-lang", "fr", a}, &out); err == nil {
		t.Error("unknown language accepted")
	}
}

func TestReportCommand(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "a.pdf")
	out := filepath.Join(dir, "r.html")
	if err := runSynth([]string{"-o", in}); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	if err := runReport([]string{"-lang", "en", "-o", out, in}, &stderr); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(out)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Errorf("report readable by others: %v", info.Mode())
	}
	page, _ := os.ReadFile(out)
	if !strings.Contains(string(page), "October 2026") {
		t.Error("report lacks the period")
	}
	if !strings.Contains(stderr.String(), "keep it private") {
		t.Errorf("no privacy warning: %q", stderr.String())
	}
	if err := runReport([]string{"-o", out}, &stderr); err == nil {
		t.Error("no files accepted")
	}
}
