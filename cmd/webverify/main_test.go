package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// site writes a small build: a page and one asset in a folder.
func site(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "a", "b1"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{"index.html": "<p>page</p>", "a/b1/app.js": "app()"} {
		if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(name)), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func sums(t *testing.T, dir string) string {
	t.Helper()
	var b bytes.Buffer
	if err := listDir(&b, dir); err != nil {
		t.Fatal(err)
	}
	name := filepath.Join(t.TempDir(), "sums")
	if err := os.WriteFile(name, b.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return name
}

func TestListDir(t *testing.T) {
	var b bytes.Buffer
	if err := listDir(&b, site(t)); err != nil {
		t.Fatal(err)
	}
	// Sorted by path, in sha256sum's format, with paths of the site.
	lines := strings.Split(strings.TrimSpace(b.String()), "\n")
	if len(lines) != 2 || !strings.HasSuffix(lines[0], "  a/b1/app.js") || !strings.HasSuffix(lines[1], "  index.html") {
		t.Fatalf("list:\n%s", b.String())
	}
	if !line.MatchString(lines[0]) {
		t.Errorf("not in the format read back: %q", lines[0])
	}
}

func TestCheck(t *testing.T) {
	dir := site(t)
	want, err := readSums(http.DefaultClient, sums(t, dir))
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.FileServer(http.Dir(dir)))
	defer srv.Close()
	for _, s := range []string{dir, srv.URL + "/", srv.URL} {
		var out bytes.Buffer
		if bad, err := check(&out, srv.Client(), want, s); err != nil || bad != 0 {
			t.Errorf("%s: %d bad, %v\n%s", s, bad, err, out.String())
		}
	}

	// A changed file and a missing one are both reported.
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<p>other</p>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, "a", "b1", "app.js")); err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{dir, srv.URL} {
		var out bytes.Buffer
		bad, err := check(&out, srv.Client(), want, s)
		if err != nil || bad != 2 || !strings.Contains(out.String(), "differs  index.html") || !strings.Contains(out.String(), "missing  a/b1/app.js") {
			t.Errorf("%s: %d bad, %v\n%s", s, bad, err, out.String())
		}
	}
}

func TestReadSums(t *testing.T) {
	sum := strings.Repeat("ab", 32)
	for _, bad := range []string{
		"",                 // no page
		sum + "  app.js\n", // no page either
		sum + "  index.html\n" + sum + "  ../secret\n",
		sum + "  index.html\n" + sum + "  /etc/passwd\n",
		sum + "  index.html\n" + sum + "  a/./b\n",
		sum + "  index.html\n" + sum + "  index.html\n",
		"xyz  index.html\n",
	} {
		name := filepath.Join(t.TempDir(), "sums")
		if err := os.WriteFile(name, []byte(bad), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := readSums(http.DefaultClient, name); err == nil {
			t.Errorf("accepted %q", bad)
		}
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sums" {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte(sum + "  index.html\n"))
	}))
	defer srv.Close()
	if got, err := readSums(srv.Client(), srv.URL+"/sums"); err != nil || len(got) != 1 {
		t.Errorf("from an address: %v, %v", got, err)
	}
	if _, err := readSums(srv.Client(), srv.URL+"/none"); err == nil {
		t.Error("a missing list was accepted")
	}
}
