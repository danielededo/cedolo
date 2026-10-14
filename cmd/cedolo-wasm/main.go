//go:build js && wasm

// Command cedolo-wasm exposes the cedolo core to a web page. It is a spike:
// the page extracts the positioned text of a PDF and passes it as JSON.
package main

import (
	"bytes"
	"encoding/json"
	"syscall/js"

	"github.com/danielededo/cedolo/internal/extract"
	"github.com/danielededo/cedolo/internal/i18n"
	"github.com/danielededo/cedolo/internal/layout"
	"github.com/danielededo/cedolo/internal/payslip"
	"github.com/danielededo/cedolo/internal/report"
)

// report(layoutsJSON, lang) returns the HTML report for the documents, each a
// layout.Document as JSON.
func renderReport(_ js.Value, args []js.Value) any {
	var docs []layout.Document
	if err := json.Unmarshal([]byte(args[0].String()), &docs); err != nil {
		return map[string]any{"error": err.Error()}
	}
	l, err := i18n.Parse(args[1].String())
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	var slips []payslip.Payslip
	for _, d := range docs {
		slips = append(slips, extract.Document(d)...)
	}
	var b bytes.Buffer
	if err := report.Render(&b, l, slips, "wasm"); err != nil {
		return map[string]any{"error": err.Error()}
	}
	out, _ := json.Marshal(slips)
	return map[string]any{"html": b.String(), "payslips": string(out)}
}

func main() {
	js.Global().Set("cedoloReport", js.FuncOf(renderReport))
	select {}
}
