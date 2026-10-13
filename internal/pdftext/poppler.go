package pdftext

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/danielededo/cedolo/internal/layout"
)

// Poppler reads PDFs by running poppler's pdftotext with bounding box output.
// It needs the pdftotext program installed (package poppler-utils on most
// Linux distributions, poppler on Homebrew). Poppler is GPL licensed; running
// it as a separate program keeps it out of cedolo's own binary.
type Poppler struct{}

func (Poppler) Name() string { return "poppler" }

func (Poppler) ReadFile(path string) (layout.Document, error) {
	bin, err := exec.LookPath("pdftotext")
	if err != nil {
		return layout.Document{}, errors.New("pdftotext not found; install poppler-utils or use the pdfium reader")
	}
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(bin, "-bbox", "-enc", "UTF-8", path, "-")
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return layout.Document{}, fmt.Errorf("pdftotext: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return parseBBox(stdout.Bytes())
}

type bboxHTML struct {
	Pages []struct {
		Width  float64 `xml:"width,attr"`
		Height float64 `xml:"height,attr"`
		Words  []struct {
			XMin float64 `xml:"xMin,attr"`
			YMin float64 `xml:"yMin,attr"`
			XMax float64 `xml:"xMax,attr"`
			YMax float64 `xml:"yMax,attr"`
			Text string  `xml:",chardata"`
		} `xml:"word"`
	} `xml:"body>doc>page"`
}

// parseBBox parses the XHTML produced by pdftotext -bbox, whose coordinates
// already use a top-left origin.
func parseBBox(data []byte) (layout.Document, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = false
	dec.Entity = xml.HTMLEntity
	var h bboxHTML
	if err := dec.Decode(&h); err != nil {
		return layout.Document{}, fmt.Errorf("parse pdftotext output: %w", err)
	}
	doc := layout.Document{Pages: make([]layout.Page, len(h.Pages))}
	for i, p := range h.Pages {
		lp := layout.Page{Number: i + 1, Width: p.Width, Height: p.Height}
		for _, w := range p.Words {
			lp.Tokens = append(lp.Tokens, layout.Token{
				Text: w.Text,
				X:    w.XMin,
				Y:    w.YMin,
				W:    w.XMax - w.XMin,
				H:    w.YMax - w.YMin,
			})
		}
		doc.Pages[i] = lp
	}
	return doc, nil
}
