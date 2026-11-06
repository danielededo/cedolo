package main

import (
	"strings"
	"testing"
)

func TestPin(t *testing.T) {
	page := []byte(`<meta http-equiv="Content-Security-Policy" content="default-src 'none'; script-src 'self' 'wasm-unsafe-eval'; connect-src 'self'"><p>'self' elsewhere</p>`)
	got, err := pin(page, "https://example.org/cedolo/", "a/0123456789ab")
	if err != nil {
		t.Fatal(err)
	}
	want := `<meta http-equiv="Content-Security-Policy" content="default-src 'none'; script-src https://example.org/cedolo/a/0123456789ab/ 'wasm-unsafe-eval'; connect-src https://example.org/cedolo/a/0123456789ab/"><p>'self' elsewhere</p>`
	if string(got) != want {
		t.Errorf("pinned\n%s\nwant\n%s", got, want)
	}
	for _, bad := range []string{"example.org/cedolo/", "https://example.org/cedolo", "https://example.org/a b/", "https://example.org/x';/"} {
		if _, err := pin(page, bad, "a/x"); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
	if _, err := pin([]byte("<p>no policy</p>"), "https://example.org/", "a/x"); err == nil || !strings.Contains(err.Error(), "Content-Security-Policy") {
		t.Errorf("a page without a policy: %v", err)
	}
}
