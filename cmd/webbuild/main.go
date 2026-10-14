// Command webbuild assembles the cedolo web app as static files: the Go core
// compiled to WebAssembly, the pages in web/ and pdf.js from web/node_modules
// (run "npm ci" in web/ first). Serve the output directory with any static
// file server.
//
//	go run ./cmd/webbuild -o dist/web -version v0.2.0
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/danielededo/cedolo/internal/report"
)

func main() {
	out := flag.String("o", "dist/web", "output directory")
	version := flag.String("version", "dev", "version shown by the app")
	flag.Parse()
	if err := build(*out, *version); err != nil {
		fmt.Fprintln(os.Stderr, "webbuild:", err)
		os.Exit(1)
	}
}

func build(out, version string) error {
	pdfjs := filepath.Join("web", "node_modules", "pdfjs-dist")
	if _, err := os.Stat(pdfjs); err != nil {
		return errors.New("pdf.js is missing: run \"npm ci\" in web/ first")
	}
	if err := os.RemoveAll(out); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(out, "pdfjs"), 0o755); err != nil {
		return err
	}

	cmd := exec.Command("go", "build", "-trimpath", "-ldflags", "-s -w -X main.version="+version,
		"-o", filepath.Join(out, "cedolo.wasm"), "./cmd/cedolo-wasm")
	cmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("build the core: %w", err)
	}
	goroot, err := exec.Command("go", "env", "GOROOT").Output()
	if err != nil {
		return err
	}
	if err := copyFile(filepath.Join(strings.TrimSpace(string(goroot)), "lib", "wasm", "wasm_exec.js"), filepath.Join(out, "wasm_exec.js")); err != nil {
		return err
	}

	page, err := os.ReadFile(filepath.Join("web", "index.html"))
	if err != nil {
		return err
	}
	const placeholder = "{{REPORT_SCRIPT_HASH}}"
	if !bytes.Contains(page, []byte(placeholder)) {
		return errors.New("web/index.html lacks " + placeholder)
	}
	page = bytes.ReplaceAll(page, []byte(placeholder), []byte(report.ScriptHash()))
	if err := os.WriteFile(filepath.Join(out, "index.html"), page, 0o644); err != nil {
		return err
	}
	for _, name := range []string{"app.js", "layout.js", "style.css"} {
		if err := copyFile(filepath.Join("web", name), filepath.Join(out, name)); err != nil {
			return err
		}
	}

	for _, name := range []string{"build/pdf.min.mjs", "build/pdf.worker.min.mjs", "LICENSE"} {
		if err := copyFile(filepath.Join(pdfjs, name), filepath.Join(out, "pdfjs", filepath.Base(name))); err != nil {
			return err
		}
	}
	for _, dir := range []string{"standard_fonts", "cmaps"} {
		if err := copyDir(filepath.Join(pdfjs, dir), filepath.Join(out, "pdfjs", dir)); err != nil {
			return err
		}
	}
	fmt.Println("web app written to", out)
	return nil
}

func copyFile(from, to string) error {
	data, err := os.ReadFile(from)
	if err != nil {
		return err
	}
	return os.WriteFile(to, data, 0o644)
}

func copyDir(from, to string) error {
	return filepath.WalkDir(from, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(from, path)
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(to, rel), 0o755)
		}
		return copyFile(path, filepath.Join(to, rel))
	})
}
