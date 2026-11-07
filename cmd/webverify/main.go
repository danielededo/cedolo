// Command webverify checks that the web app online is the one built from a
// release. Each release lists the SHA-256 of every file of the site, built
// as GitHub Pages builds it; webverify fetches each file from the site and
// compares.
//
//	go run ./cmd/webverify -tag v0.12.0
//	go run ./cmd/webverify -tag v0.12.0 -site https://example.org/cedolo/
//	go run ./cmd/webverify -sums cedolo-web_v0.12.0.sha256 -site dist/web
//	go run ./cmd/webverify -list dist/web > cedolo-web_v0.12.0.sha256
//
// -site is an address or a folder: a folder is a build of your own, made
// with the release's go.mod toolchain and web/package-lock.json:
//
//	go run ./cmd/webbuild -o dist/web -version v0.12.0 -url https://danielededo.github.io/cedolo/
//
// The page names every other file it loads, and is itself in the list, so
// a site that matches it serves nothing else to the app.
package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
)

const (
	defaultSite = "https://danielededo.github.io/cedolo/"
	releases    = "https://github.com/danielededo/cedolo/releases/download/"
	// maxFile bounds what is read of one file of the site.
	maxFile = 64 << 20
)

func main() {
	tag := flag.String("tag", "", "the release to check against, such as v0.12.0")
	sums := flag.String("sums", "", "the list of hashes, a file or an address, instead of the release's")
	site := flag.String("site", defaultSite, "the site to check, an address or a folder")
	list := flag.String("list", "", "print the list of hashes of the files in this folder, and stop")
	flag.Parse()

	if *list != "" {
		if err := listDir(os.Stdout, *list); err != nil {
			fail(err)
		}
		return
	}
	src := *sums
	if src == "" {
		if !regexp.MustCompile(`^v\d+\.\d+\.\d+$`).MatchString(*tag) {
			fail(errors.New("give -tag, such as v0.12.0, or -sums"))
		}
		src = releases + *tag + "/cedolo-web_" + *tag + ".sha256"
	}
	client := &http.Client{Timeout: time.Minute}
	want, err := readSums(client, src)
	if err != nil && *sums == "" && strings.Contains(err.Error(), "404") {
		err = fmt.Errorf("%w: %s has no list of the site's hashes (releases before v0.12.0 have none)", err, *tag)
	}
	if err != nil {
		fail(err)
	}
	bad, err := check(os.Stdout, client, want, *site)
	if err != nil {
		fail(err)
	}
	if bad > 0 {
		fail(fmt.Errorf("%d of %d files differ from the release", bad, len(want)))
	}
	fmt.Printf("all %d files match\n", len(want))
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "webverify:", err)
	os.Exit(1)
}

// entry is one line of a list of hashes.
type entry struct {
	sum, path string
}

// listDir writes the SHA-256 of every file under dir, by path, in the
// format of sha256sum.
func listDir(out io.Writer, dir string) error {
	var paths []string
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type().IsRegular() {
			rel, err := filepath.Rel(dir, p)
			if err != nil {
				return err
			}
			paths = append(paths, filepath.ToSlash(rel))
		} else if !d.IsDir() {
			return fmt.Errorf("%s: not a regular file", p)
		}
		return nil
	})
	if err != nil {
		return err
	}
	slices.Sort(paths)
	for _, p := range paths {
		sum, err := fileSum(filepath.Join(dir, filepath.FromSlash(p)))
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "%s  %s\n", sum, p)
	}
	return nil
}

func fileSum(name string) (string, error) {
	f, err := os.Open(name)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

var line = regexp.MustCompile(`^([0-9a-f]{64})  ([A-Za-z0-9_][A-Za-z0-9._/-]*)$`)

// readSums reads a list of hashes from a file or an address. Every path
// must be a plain relative one, so it names a file of the site only.
func readSums(client *http.Client, src string) ([]entry, error) {
	var r io.Reader
	if isURL(src) {
		body, err := fetch(client, src)
		if err != nil {
			return nil, fmt.Errorf("list of hashes: %w", err)
		}
		r = strings.NewReader(string(body))
	} else {
		f, err := os.Open(src)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		r = f
	}
	var out []entry
	seen := map[string]bool{}
	sc := bufio.NewScanner(r)
	for n := 1; sc.Scan(); n++ {
		m := line.FindStringSubmatch(sc.Text())
		// A path starts with a letter, a digit or _, so not with / or ..,
		// and has no . or .. inside.
		if m == nil || path.Clean(m[2]) != m[2] || strings.Contains(m[2], "/..") || seen[m[2]] {
			return nil, fmt.Errorf("list of hashes, line %d: not a hash and a file of the site", n)
		}
		seen[m[2]] = true
		out = append(out, entry{m[1], m[2]})
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if !seen["index.html"] {
		return nil, errors.New("list of hashes: no index.html")
	}
	return out, nil
}

// check compares every file of the list with the site's, writing one line
// for each that differs or is missing, and returns how many did.
func check(out io.Writer, client *http.Client, want []entry, site string) (int, error) {
	remote := isURL(site)
	if remote && !strings.HasSuffix(site, "/") {
		site += "/"
	}
	bad := 0
	for _, e := range want {
		var got string
		if remote {
			body, err := fetch(client, site+e.path)
			if err != nil {
				fmt.Fprintf(out, "missing  %s (%v)\n", e.path, err)
				bad++
				continue
			}
			s := sha256.Sum256(body)
			got = hex.EncodeToString(s[:])
		} else {
			s, err := fileSum(filepath.Join(site, filepath.FromSlash(e.path)))
			if errors.Is(err, fs.ErrNotExist) {
				fmt.Fprintf(out, "missing  %s\n", e.path)
				bad++
				continue
			} else if err != nil {
				return 0, err
			}
			got = s
		}
		if got != e.sum {
			fmt.Fprintf(out, "differs  %s\n", e.path)
			bad++
		}
	}
	return bad, nil
}

func isURL(s string) bool {
	return strings.HasPrefix(s, "https://") || strings.HasPrefix(s, "http://")
}

func fetch(client *http.Client, url string) ([]byte, error) {
	res, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, errors.New(res.Status)
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, maxFile+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxFile {
		return nil, errors.New("too large")
	}
	return body, nil
}
