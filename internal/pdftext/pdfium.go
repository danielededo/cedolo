package pdftext

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/klippa-app/go-pdfium"
	pdfiumerrors "github.com/klippa-app/go-pdfium/errors"
	"github.com/klippa-app/go-pdfium/references"
	"github.com/klippa-app/go-pdfium/requests"
	"github.com/klippa-app/go-pdfium/webassembly"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/experimental"

	"github.com/danielededo/cedolo/internal/layout"
)

// PDFium reads PDFs with PDFium, the PDF engine of Chromium, compiled to
// WebAssembly and run by a pure Go runtime (github.com/klippa-app/go-pdfium,
// MIT; PDFium is BSD and Apache licensed). It needs no external program and
// no cgo, at the cost of about 5 MB of binary size and, the first time, a
// start-up of a few seconds to compile it.
type PDFium struct{}

func (PDFium) Name() string { return "pdfium" }

var (
	pdfiumOnce sync.Once
	pdfiumPool pdfium.Pool
	pdfiumErr  error
)

func pdfiumInstance() (pdfium.Pdfium, error) {
	pdfiumOnce.Do(func() {
		start := func(rc wazero.RuntimeConfig) (pdfium.Pool, error) {
			return webassembly.Init(webassembly.Config{
				MinIdle: 1, MaxIdle: 1, MaxTotal: 1,
				RuntimeConfig: rc,
				// PDFium gets the PDF as bytes and needs no file: it sees
				// none. By default go-pdfium would let it read and write the
				// whole file system, and look up system fonts for a PDF that
				// does not embed its own, so that one PDF read differently
				// from one computer to another. Without them PDFium uses its
				// built-in fonts.
				FSConfig: wazero.NewFSConfig(),
			})
		}
		rc, cached := pdfiumRuntime()
		pdfiumPool, pdfiumErr = start(rc)
		// The cache must never stop a read: one that cannot be written or
		// read, perhaps emptied by another cedolo meanwhile, is left out.
		if pdfiumErr != nil && cached {
			pdfiumPool, pdfiumErr = start(pdfiumFeatures())
		}
	})
	if pdfiumErr != nil {
		return nil, fmt.Errorf("start pdfium: %w", pdfiumErr)
	}
	return pdfiumPool.GetInstance(30 * time.Second)
}

// pdfiumRuntime compiles PDFium once per version of cedolo and keeps the
// machine code in the user's cache directory, so that later runs start in a
// fraction of a second instead of several. The cache holds compiled PDFium
// only, never anything read from a PDF. Without a cache directory PDFium is
// compiled at every start, as before. The features are those go-pdfium sets
// by default.
func pdfiumRuntime() (rc wazero.RuntimeConfig, cached bool) {
	rc = pdfiumFeatures()
	dir, err := os.UserCacheDir()
	if err != nil {
		return rc, false
	}
	cache, err := wazero.NewCompilationCacheWithDir(pdfiumCache(filepath.Join(dir, "cedolo", "pdfium"), pdfiumBuild()))
	if err != nil {
		return rc, false
	}
	return rc.WithCompilationCache(cache), true
}

func pdfiumFeatures() wazero.RuntimeConfig {
	return wazero.NewRuntimeConfig().WithCoreFeatures(api.CoreFeaturesV2 | experimental.CoreFeaturesExceptionHandling)
}

// pdfiumBuild names the PDFium compiled by this build of cedolo: the
// versions of go-pdfium, which holds PDFium, and of wazero, which compiles
// it, such as "go-pdfium-v1.21.4-wazero-v1.12.0".
func pdfiumBuild() string {
	versions := map[string]string{"github.com/klippa-app/go-pdfium": "unknown", "github.com/tetratelabs/wazero": "unknown"}
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, d := range info.Deps {
			if _, ok := versions[d.Path]; ok {
				versions[d.Path] = d.Version
			}
		}
	}
	return "go-pdfium-" + versions["github.com/klippa-app/go-pdfium"] + "-wazero-" + versions["github.com/tetratelabs/wazero"]
}

// pdfiumCache returns the folder of the PDFium of this build within root,
// and removes the others: each version compiled, about 20 MB, would
// otherwise stay there for good. Only a build that knows its versions
// removes them, so that a development build leaves a release's alone. A
// folder that cannot be removed is left for next time.
func pdfiumCache(root, build string) string {
	if entries, err := os.ReadDir(root); err == nil && !strings.Contains(build, "unknown") {
		for _, e := range entries {
			if e.Name() != build {
				_ = os.RemoveAll(filepath.Join(root, e.Name()))
			}
		}
	}
	return filepath.Join(root, build)
}

func (p PDFium) ReadFile(path string) (layout.Document, error) {
	return p.ReadFileWithPassword(path, "")
}

func (PDFium) ReadFileWithPassword(path, password string) (layout.Document, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return layout.Document{}, err
	}
	inst, err := pdfiumInstance()
	if err != nil {
		return layout.Document{}, err
	}
	defer inst.Close()

	open := &requests.OpenDocument{File: &data}
	if password != "" {
		open.Password = &password
	}
	opened, err := inst.OpenDocument(open)
	switch {
	case errors.Is(err, pdfiumerrors.ErrPassword):
		return layout.Document{}, ErrPassword
	case errors.Is(err, pdfiumerrors.ErrFormat), errors.Is(err, pdfiumerrors.ErrFile):
		return layout.Document{}, ErrDamaged
	case err != nil:
		return layout.Document{}, fmt.Errorf("open PDF: %w", err)
	}
	defer inst.FPDF_CloseDocument(&requests.FPDF_CloseDocument{Document: opened.Document})

	count, err := inst.FPDF_GetPageCount(&requests.FPDF_GetPageCount{Document: opened.Document})
	if err != nil {
		return layout.Document{}, err
	}
	var doc layout.Document
	for i := 0; i < count.PageCount; i++ {
		page, err := readPDFiumPage(inst, opened.Document, i)
		if err != nil {
			return layout.Document{}, fmt.Errorf("page %d: %w", i+1, err)
		}
		doc.Pages = append(doc.Pages, page)
	}
	return doc, nil
}

func readPDFiumPage(inst pdfium.Pdfium, d references.FPDF_DOCUMENT, index int) (layout.Page, error) {
	loaded, err := inst.FPDF_LoadPage(&requests.FPDF_LoadPage{Document: d, Index: index})
	if err != nil {
		return layout.Page{}, err
	}
	defer inst.FPDF_ClosePage(&requests.FPDF_ClosePage{Page: loaded.Page})
	ref := requests.Page{ByReference: &loaded.Page}

	w, err := inst.FPDF_GetPageWidthF(&requests.FPDF_GetPageWidthF{Page: ref})
	if err != nil {
		return layout.Page{}, err
	}
	h, err := inst.FPDF_GetPageHeightF(&requests.FPDF_GetPageHeightF{Page: ref})
	if err != nil {
		return layout.Page{}, err
	}
	height := float64(h.PageHeight)

	text, err := inst.FPDFText_LoadPage(&requests.FPDFText_LoadPage{Page: ref})
	if err != nil {
		return layout.Page{}, err
	}
	defer inst.FPDFText_ClosePage(&requests.FPDFText_ClosePage{TextPage: text.TextPage})
	n, err := inst.FPDFText_CountChars(&requests.FPDFText_CountChars{TextPage: text.TextPage})
	if err != nil {
		return layout.Page{}, err
	}

	glyphs := make([]glyph, 0, n.Count)
	for i := 0; i < n.Count; i++ {
		u, err := inst.FPDFText_GetUnicode(&requests.FPDFText_GetUnicode{TextPage: text.TextPage, Index: i})
		if err != nil {
			return layout.Page{}, err
		}
		r := rune(u.Unicode)
		if r == 0 {
			r = unicode.ReplacementChar
		}
		// Characters PDFium generates (spaces and line breaks it infers)
		// carry no box; keep them as word separators.
		gen, err := inst.FPDFText_IsGenerated(&requests.FPDFText_IsGenerated{TextPage: text.TextPage, Index: i})
		if err != nil {
			return layout.Page{}, err
		}
		if gen.IsGenerated || unicode.IsSpace(r) {
			glyphs = append(glyphs, glyph{r: ' '})
			continue
		}
		box, err := inst.FPDFText_GetLooseCharBox(&requests.FPDFText_GetLooseCharBox{TextPage: text.TextPage, Index: i})
		if err != nil {
			return layout.Page{}, err
		}
		size, err := inst.FPDFText_GetFontSize(&requests.FPDFText_GetFontSize{TextPage: text.TextPage, Index: i})
		if err != nil {
			return layout.Page{}, err
		}
		angle, err := inst.FPDFText_GetCharAngle(&requests.FPDFText_GetCharAngle{TextPage: text.TextPage, Index: i})
		if err != nil {
			return layout.Page{}, err
		}
		// The angle is in radians; a quarter turn either way is vertical.
		quarter := math.Abs(math.Mod(float64(angle.CharAngle), math.Pi) - math.Pi/2)
		glyphs = append(glyphs, glyph{
			r:        r,
			x0:       float64(box.Rect.Left),
			x1:       float64(box.Rect.Right),
			top:      height - float64(box.Rect.Top),
			bottom:   height - float64(box.Rect.Bottom),
			size:     size.FontSize,
			vertical: quarter < 0.2,
		})
	}
	return layout.Page{
		Number: index + 1,
		Width:  float64(w.PageWidth),
		Height: height,
		Tokens: groupWords(glyphs),
	}, nil
}
