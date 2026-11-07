//go:build js && wasm

// Command cedolo-wasm is the cedolo core for the web app. The page extracts
// the positioned text of each PDF with pdf.js and passes it here as JSON;
// everything runs in the browser.
//
// It defines a global object cedolo with:
//
//	read(layoutJSON) -> {doc} or {error}: reads the payslips of one
//	    document once and keeps them, under the number doc, for analyze
//	analyze(docsJSON, lang) -> {perDocument, report} or {error}: the
//	    report of the documents read under those numbers, in that order;
//	    perDocument has, for each, {read, repeated, periods, latest,
//	    people} or {error}: periods name the payslips read, such as
//	    "marzo 2026", latest is the latest of them as year*100+month for
//	    sorting, and people names whose payslips they are
//	forget(docsJSON) -> {}: drops what read kept for those documents
//	anonymize(layoutJSON, keepAmounts) -> {text, pdf} or {error}: an
//	    anonymized copy of one document (ADR 0008), its text for review and
//	    the rebuilt PDF as a Uint8Array
//	payslips(layoutsJSON) -> {payslips}: the payslips read, as JSON, for
//	    the tests that compare them with the CLI's
//	example(year) -> {layouts}: the layouts of a year of invented payslips,
//	    as JSON, for the page to read as files
//	texts(lang) -> JSON object of the web app's texts
//	version -> string
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"syscall/js"

	"github.com/danielededo/cedolo/internal/anonymize"
	"github.com/danielededo/cedolo/internal/extract"
	"github.com/danielededo/cedolo/internal/i18n"
	"github.com/danielededo/cedolo/internal/insights"
	"github.com/danielededo/cedolo/internal/layout"
	"github.com/danielededo/cedolo/internal/payslip"
	"github.com/danielededo/cedolo/internal/report"
	"github.com/danielededo/cedolo/internal/synth"
)

// version is set by the web build (-ldflags "-X main.version=v1.2.3").
var version = "dev"

func failure(err error) any { return map[string]any{"error": err.Error()} }

// readDocs holds the payslips of each document read, by number, so that adding
// a file or changing the language reads no document again.
var (
	readDocs = map[int]docResult{}
	nextDoc  = 1
)

type docResult struct {
	slips []payslip.Payslip
	err   error
}

// readDoc extracts the payslips of one document and keeps them.
func readDoc(_ js.Value, args []js.Value) (result any) {
	// A panic would stop the Go program and leave the page without a core;
	// report it as an error instead.
	defer func() {
		if r := recover(); r != nil {
			result = map[string]any{"error": fmt.Sprint("internal error: ", r)}
		}
	}()
	if len(args) != 1 {
		return map[string]any{"error": "read(layoutJSON)"}
	}
	var d layout.Document
	if err := json.Unmarshal([]byte(args[0].String()), &d); err != nil {
		return failure(err)
	}
	// A document that trips the extractor is kept with its error, so the
	// others are still read.
	slips, err := extractOne(d)
	id := nextDoc
	nextDoc++
	readDocs[id] = docResult{slips, err}
	return map[string]any{"doc": id}
}

// forget drops the payslips kept for some documents.
func forget(_ js.Value, args []js.Value) any {
	if len(args) != 1 {
		return map[string]any{"error": "forget(docsJSON)"}
	}
	var ids []int
	if err := json.Unmarshal([]byte(args[0].String()), &ids); err != nil {
		return failure(err)
	}
	for _, id := range ids {
		delete(readDocs, id)
	}
	return map[string]any{}
}

// analyze renders the report of documents already read. It also returns
// how many payslips each document held, so the page can say which file
// held none.
func analyze(_ js.Value, args []js.Value) (result any) {
	defer func() {
		if r := recover(); r != nil {
			result = map[string]any{"error": fmt.Sprint("internal error: ", r)}
		}
	}()
	if len(args) != 2 {
		return map[string]any{"error": "analyze(docsJSON, lang)"}
	}
	var ids []int
	if err := json.Unmarshal([]byte(args[0].String()), &ids); err != nil {
		return failure(err)
	}
	l, err := i18n.Parse(args[1].String())
	if err != nil {
		return failure(err)
	}
	// Payslips repeating one of an earlier document (the same file twice)
	// are counted once.
	perDoc := make([]any, len(ids))
	var slips []payslip.Payslip
	var seen insights.Deduper
	for i, id := range ids {
		r, ok := readDocs[id]
		if !ok {
			perDoc[i] = map[string]any{"error": fmt.Sprintf("document %d was not read", id)}
			continue
		}
		if r.err != nil {
			perDoc[i] = map[string]any{"error": r.err.Error()}
			continue
		}
		kept, dropped := seen.Keep(r.slips)
		periods := make([]any, len(r.slips))
		var latest int
		var people []any
		for k, p := range r.slips {
			periods[k] = insights.Label(l, p)
			latest = max(latest, p.Year*100+p.Month)
			if name := report.PersonLabel(l, p.Person); !slices.Contains(people, any(name)) {
				people = append(people, name)
			}
		}
		perDoc[i] = map[string]any{"read": len(r.slips), "repeated": dropped, "periods": periods, "latest": latest, "people": people}
		slips = append(slips, kept...)
	}
	var page bytes.Buffer
	if len(slips) > 0 {
		if err := report.Render(&page, l, slips, version); err != nil {
			return failure(err)
		}
	}
	return map[string]any{"perDocument": perDoc, "report": page.String()}
}

// payslips returns the payslips of every document as JSON, for the tests
// that compare them with the CLI's. The page never asks for them: they
// would be one more copy of the payslips' data in memory.
func payslips(_ js.Value, args []js.Value) (result any) {
	defer func() {
		if r := recover(); r != nil {
			result = map[string]any{"error": fmt.Sprint("internal error: ", r)}
		}
	}()
	if len(args) != 1 {
		return map[string]any{"error": "payslips(layoutsJSON)"}
	}
	var docs []layout.Document
	if err := json.Unmarshal([]byte(args[0].String()), &docs); err != nil {
		return failure(err)
	}
	var slips []payslip.Payslip
	for _, d := range docs {
		found, err := extractOne(d)
		if err != nil {
			return failure(err)
		}
		slips = append(slips, found...)
	}
	slips, _ = insights.Dedupe(slips)
	data, err := json.Marshal(slips)
	if err != nil {
		return failure(err)
	}
	return map[string]any{"payslips": string(data)}
}

// extractOne reads the payslips of one document, turning a panic into an
// error.
func extractOne(d layout.Document) (slips []payslip.Payslip, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("internal error: %v", r)
		}
	}()
	return extract.Document(d), nil
}

// anonymizeOne makes the anonymized copy of one document. Nothing is
// returned when a masked word or an identifier would survive.
func anonymizeOne(_ js.Value, args []js.Value) (result any) {
	defer func() {
		if r := recover(); r != nil {
			result = map[string]any{"error": fmt.Sprint("internal error: ", r)}
		}
	}()
	if len(args) != 2 {
		return map[string]any{"error": "anonymize(layoutJSON, keepAmounts)"}
	}
	var doc layout.Document
	if err := json.Unmarshal([]byte(args[0].String()), &doc); err != nil {
		return failure(err)
	}
	anon, _ := anonymize.Layout(doc, anonymize.Options{KeepAmounts: args[1].Truthy()})
	if leaks := anonymize.Leaks(doc, anon); len(leaks) > 0 {
		return map[string]any{"error": "leak"}
	}
	var pdf bytes.Buffer
	if err := anonymize.WritePDF(&pdf, anon); err != nil {
		return failure(err)
	}
	var text []string
	for _, p := range anon.Pages {
		for _, l := range p.Lines(2) {
			text = append(text, l.Text())
		}
		text = append(text, "")
	}
	data := js.Global().Get("Uint8Array").New(pdf.Len())
	js.CopyBytesToJS(data, pdf.Bytes())
	return map[string]any{"text": strings.TrimSpace(strings.Join(text, "\n")), "pdf": data}
}

// example returns the layouts of a year of invented payslips, for the page
// to read as if they were files: a report to look at before reading one's
// own.
func example(_ js.Value, args []js.Value) (result any) {
	defer func() {
		if r := recover(); r != nil {
			result = map[string]any{"error": fmt.Sprint("internal error: ", r)}
		}
	}()
	if len(args) != 1 {
		return map[string]any{"error": "example(year)"}
	}
	var docs []layout.Document
	for _, p := range synth.Year(args[0].Int()) {
		docs = append(docs, synth.Layout(p, synth.StyleClassic))
	}
	data, err := json.Marshal(docs)
	if err != nil {
		return failure(err)
	}
	return map[string]any{"layouts": string(data)}
}

// texts returns the web app's texts and the report texts it reuses.
func texts(_ js.Value, args []js.Value) any {
	l := i18n.IT
	if len(args) > 0 {
		if parsed, err := i18n.Parse(args[0].String()); err == nil {
			l = parsed
		}
	}
	out := map[string]string{}
	for _, k := range i18n.Keys(l) {
		if strings.HasPrefix(k, "web.") || k == "report.disclaimer" {
			out[k] = i18n.Template(l, k)
		}
	}
	data, _ := json.Marshal(out)
	return string(data)
}

func main() {
	js.Global().Set("cedolo", js.ValueOf(map[string]any{
		"analyze":   js.FuncOf(analyze),
		"forget":    js.FuncOf(forget),
		"read":      js.FuncOf(readDoc),
		"anonymize": js.FuncOf(anonymizeOne),
		"example":   js.FuncOf(example),
		"payslips":  js.FuncOf(payslips),
		"texts":     js.FuncOf(texts),
		"version":   version,
	}))
	select {}
}
