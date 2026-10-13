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
