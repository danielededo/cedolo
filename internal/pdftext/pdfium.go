package pdftext

import (
	"fmt"
	"os"
	"sync"
	"time"
	"unicode"

	"github.com/klippa-app/go-pdfium"
	"github.com/klippa-app/go-pdfium/references"
	"github.com/klippa-app/go-pdfium/requests"
	"github.com/klippa-app/go-pdfium/webassembly"

	"github.com/danielededo/cedolo/internal/layout"
)

// PDFium reads PDFs with PDFium, the PDF engine of Chromium, compiled to
// WebAssembly and run by a pure Go runtime (github.com/klippa-app/go-pdfium,
// MIT; PDFium is BSD and Apache licensed). It needs no external program and
// no cgo, at the cost of about 5 MB of binary size and a short start-up time.
type PDFium struct{}

func (PDFium) Name() string { return "pdfium" }

var (
	pdfiumOnce sync.Once
	pdfiumPool pdfium.Pool
	pdfiumErr  error
)

func pdfiumInstance() (pdfium.Pdfium, error) {
	pdfiumOnce.Do(func() {
		pdfiumPool, pdfiumErr = webassembly.Init(webassembly.Config{MinIdle: 1, MaxIdle: 1, MaxTotal: 1})
	})
	if pdfiumErr != nil {
		return nil, fmt.Errorf("start pdfium: %w", pdfiumErr)
	}
	return pdfiumPool.GetInstance(30 * time.Second)
}

func (PDFium) ReadFile(path string) (layout.Document, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return layout.Document{}, err
	}
	inst, err := pdfiumInstance()
	if err != nil {
		return layout.Document{}, err
	}
	defer inst.Close()

	opened, err := inst.OpenDocument(&requests.OpenDocument{File: &data})
	if err != nil {
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
		glyphs = append(glyphs, glyph{
			r:      r,
			x0:     float64(box.Rect.Left),
			x1:     float64(box.Rect.Right),
			top:    height - float64(box.Rect.Top),
			bottom: height - float64(box.Rect.Bottom),
			size:   size.FontSize,
		})
	}
	return layout.Page{
		Number: index + 1,
		Width:  float64(w.PageWidth),
		Height: height,
		Tokens: groupWords(glyphs),
	}, nil
}
