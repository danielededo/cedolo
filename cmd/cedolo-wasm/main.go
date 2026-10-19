//go:build js && wasm

// Command cedolo-wasm is the cedolo core for the web app. The page extracts
// the positioned text of each PDF with pdf.js and passes it here as JSON;
// everything runs in the browser.
//
// It defines a global object cedolo with:
//
//	analyze(layoutsJSON, lang) -> {payslips, perDocument, report} or {error};
//	    perDocument has, for each document, {read, repeated, periods} or
//	    {error}, periods naming the payslips read, such as "marzo 2026"
//	anonymize(layoutJSON, keepAmounts) -> {text, pdf} or {error}: an
//	    anonymized copy of one document (ADR 0008), its text for review and
//	    the rebuilt PDF as a Uint8Array
//	texts(lang) -> JSON object of the web app's texts
//	version -> string
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"syscall/js"

	"github.com/danielededo/cedolo/internal/anonymize"
	"github.com/danielededo/cedolo/internal/extract"
	"github.com/danielededo/cedolo/internal/i18n"
	"github.com/danielededo/cedolo/internal/insights"
	"github.com/danielededo/cedolo/internal/layout"
	"github.com/danielededo/cedolo/internal/payslip"
	"github.com/danielededo/cedolo/internal/report"
)

// version is set by the web build (-ldflags "-X main.version=v1.2.3").
var version = "dev"

func failure(err error) any { return map[string]any{"error": err.Error()} }

// analyze extracts the payslips of every document and renders the report.
// It also returns how many payslips each document held, so the page can
// say which file held none.
func analyze(_ js.Value, args []js.Value) (result any) {
	// A panic would stop the Go program and leave the page without a core;
	// report it as an error instead.
	defer func() {
		if r := recover(); r != nil {
			result = map[string]any{"error": fmt.Sprint("internal error: ", r)}
		}
	}()
	if len(args) != 2 {
		return map[string]any{"error": "analyze(layoutsJSON, lang)"}
	}
	var docs []layout.Document
	if err := json.Unmarshal([]byte(args[0].String()), &docs); err != nil {
		return failure(err)
	}
	l, err := i18n.Parse(args[1].String())
	if err != nil {
		return failure(err)
	}
	// Each document is read on its own, so one that trips the extractor is
	// reported and the others are kept. Payslips repeating an earlier one
	// (the same file twice) are counted once.
	perDoc := make([]any, len(docs))
	var slips []payslip.Payslip
	for i, d := range docs {
		found, err := extractOne(d)
		if err != nil {
			perDoc[i] = map[string]any{"error": err.Error()}
			continue
		}
		kept, dropped := insights.Dedupe(append(append([]payslip.Payslip(nil), slips...), found...))
		periods := make([]any, len(found))
		for k, p := range found {
			periods[k] = insights.Label(l, p)
		}
		perDoc[i] = map[string]any{"read": len(found), "repeated": dropped, "periods": periods}
		slips = kept
	}
	var page bytes.Buffer
	if len(slips) > 0 {
		if err := report.Render(&page, l, slips, version); err != nil {
			return failure(err)
		}
	}
	data, err := json.Marshal(slips)
	if err != nil {
		return failure(err)
	}
	return map[string]any{"payslips": string(data), "perDocument": perDoc, "report": page.String()}
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
		"anonymize": js.FuncOf(anonymizeOne),
		"texts":     js.FuncOf(texts),
		"version":   version,
	}))
	select {}
}
