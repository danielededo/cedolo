package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
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
	if err := runAnonymize([]string{"-amounts", "mask", path}, &out, &info); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "BIANCHI") || strings.Contains(out.String(), "1.667,40") {
		t.Error("anonymized output leaks personal data")
	}
	if !strings.Contains(out.String(), "TOTALE") || !strings.Contains(info.String(), "Review it") || strings.Contains(info.String(), "amounts are kept") {
		t.Errorf("unexpected output; info: %s", info.String())
	}
	var doc layout.Document
	if err := json.Unmarshal(out.Bytes(), &doc); err != nil || doc.TokenCount() == 0 {
		t.Errorf("output is not a layout: %v", err)
	}

	// A rebuilt PDF keeps the amounts by default and reads as the same payslip.
	pdf := filepath.Join(t.TempDir(), "shared.pdf")
	out.Reset()
	info.Reset()
	if err := runAnonymize([]string{"-o", pdf, path}, &out, &info); err != nil {
		t.Fatal(err)
	}
	if out.Len() != 0 || !strings.Contains(info.String(), "amounts are kept") {
		t.Errorf("stdout %q, info %q", out.String(), info.String())
	}
	data, err := os.ReadFile(pdf)
	if err != nil || bytes.Contains(data, []byte("BIANCHI")) || !bytes.Contains(data, []byte("1.667,40")) {
		t.Fatalf("rebuilt PDF: %v", err)
	}
	out.Reset()
	if err := runExtract([]string{pdf}, &out); err != nil || !strings.Contains(out.String(), "1.667,40") {
		t.Errorf("rebuilt PDF does not extract: %v\n%s", err, out.String())
	}

	if err := runAnonymize([]string{"-o", path, path}, &out, &info); err == nil {
		t.Error("overwrote the payslip")
	}
	if err := runAnonymize([]string{"-amounts", "some", path}, &out, &info); err == nil {
		t.Error("accepted -amounts some")
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

func TestCCNLCommand(t *testing.T) {
	var out bytes.Buffer
	if err := runCCNL([]string{"-lang", "en", "h011"}, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "H011") || !strings.Contains(out.String(), "CNEL") {
		t.Errorf("unexpected output:\n%s", out.String())
	}
	out.Reset()
	if err := runCCNL([]string{"-search", "studi", "professionali"}, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "non sono univoci") {
		t.Errorf("search output lacks the warning:\n%s", out.String())
	}
	if err := runCCNL(nil, &out); err == nil {
		t.Error("no arguments accepted")
	}
}

func TestReadingIsResilient(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "good.pdf")
	locked := filepath.Join(dir, "locked.pdf")
	locked2 := filepath.Join(dir, "locked2.pdf")
	damaged := filepath.Join(dir, "damaged.pdf")
	if err := runSynth([]string{"-o", good}); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{locked, locked2} {
		if err := runSynth([]string{"-style", "grid", "-password", "segreto", "-o", p}); err != nil {
			t.Fatal(err)
		}
	}
	os.WriteFile(damaged, []byte("not a pdf"), 0o600)

	var warn bytes.Buffer
	warnings = &warn
	defer func() { warnings = os.Stderr }()

	// A damaged file is skipped; the others are read.
	asked := 0
	askPassword = func(string) (string, error) { asked++; return "segreto", nil }
	slips, err := readAll("pdfium", []string{good, damaged, locked, locked2})
	if err != nil || len(slips) != 3 {
		t.Fatalf("readAll = %d payslips, %v", len(slips), err)
	}
	if !strings.Contains(warn.String(), "damaged.pdf: the file is not a PDF or is damaged (skipped)") {
		t.Errorf("warnings: %q", warn.String())
	}
	// The password is asked once and reused for the next protected file.
	if asked != 1 {
		t.Errorf("asked for a password %d times", asked)
	}

	// Without a terminal, the environment variable is used.
	askPassword = func(string) (string, error) { return "", errors.New("no terminal") }
	t.Setenv(passwordEnv, "segreto")
	if slips, err := readAll("pdfium", []string{locked}); err != nil || len(slips) != 1 {
		t.Errorf("with %s: %d payslips, %v", passwordEnv, len(slips), err)
	}
	t.Setenv(passwordEnv, "")
	warn.Reset()
	if _, err := readAll("pdfium", []string{locked}); err == nil || !strings.Contains(warn.String(), passwordEnv) {
		t.Errorf("no password available: %v, %q", err, warn.String())
	}

	// The same payslip given twice is counted once in the report.
	warn.Reset()
	out := filepath.Join(dir, "r.html")
	if err := runReport([]string{"-o", out, good, good}, io.Discard); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(warn.String(), "1 repeated payslip(s) ignored") {
		t.Errorf("warnings: %q", warn.String())
	}
}

func TestGlossaryCommand(t *testing.T) {
	var out bytes.Buffer
	if err := runGlossary(nil, &out); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"IRPEF\n", "Conguaglio\n", "Addizionale comunale\n"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("full glossary lacks %q", want)
		}
	}
	out.Reset()
	if err := runGlossary([]string{"-lang", "en", "addizionale"}, &out); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "Regional income tax surcharge (Addizionale regionale)\n") ||
		!strings.Contains(got, "Municipal income tax surcharge") || strings.Contains(got, "IRPEF)\n") {
		t.Errorf("search for addizionale:\n%s", got)
	}
	for _, line := range strings.Split(got, "\n") {
		if len([]rune(line)) > 78 {
			t.Errorf("line longer than 78: %q", line)
		}
	}
	out.Reset()
	if err := runGlossary([]string{"imponibile", "fiscale"}, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out.String(), "Imponibile fiscale\n") || strings.Contains(out.String(), "Imponibile previdenziale") {
		t.Errorf("search for imponibile fiscale:\n%s", out.String())
	}
	out.Reset()
	if err := runGlossary([]string{"xyz"}, &out); err != nil || !strings.Contains(out.String(), "Nessuna") {
		t.Errorf("no match: %q, %v", out.String(), err)
	}
}
