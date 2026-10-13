// Command piiscan walks the given paths and fails if any file contains a
// valid codice fiscale or IBAN. CI runs it over the whole repository as a
// safety net against committing personal data. Findings are reported masked.
package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/danielededo/cedolo/internal/pii"
)

func main() {
	roots := os.Args[1:]
	if len(roots) == 0 {
		roots = []string{"."}
	}
	found := 0
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if d.Name() == ".git" {
					return filepath.SkipDir
				}
				return nil
			}
			if !d.Type().IsRegular() {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if bytes.IndexByte(data, 0) >= 0 {
				fmt.Fprintf(os.Stderr, "%s: binary file, not scanned\n", path)
				return nil
			}
			for _, m := range pii.Find(string(data)) {
				line := 1 + strings.Count(string(data[:m.Offset]), "\n")
				fmt.Printf("%s:%d: %s %s\n", path, line, m.Kind, m.Masked)
				found++
			}
			return nil
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, "piiscan:", err)
			os.Exit(2)
		}
	}
	if found > 0 {
		fmt.Fprintf(os.Stderr, "piiscan: %d personal identifier(s) found; remove them before committing\n", found)
		os.Exit(1)
	}
}
