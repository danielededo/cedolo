package main

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/danielededo/cedolo/internal/i18n"
	"github.com/danielededo/cedolo/internal/taxonomy"
)

func runGlossary(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("glossary", flag.ContinueOnError)
	lang := fs.String("lang", "it", "language: it or en")
	if err := fs.Parse(args); err != nil {
		return err
	}
	l, err := i18n.Parse(*lang)
	if err != nil {
		return err
	}
	query := taxonomy.Normalize(strings.Join(fs.Args(), " "))
	// matches tells whether every word of the query appears in one of the
	// texts, so "addizionale" finds both surcharges and "inps" finds the
	// contributions and the contribution base.
	matches := func(texts ...string) bool {
		if query == "" {
			return true
		}
		all := " " + taxonomy.Normalize(strings.Join(texts, " ")) + " "
		for _, w := range strings.Fields(query) {
			if !strings.Contains(all, " "+w) {
				return false
			}
		}
		return true
	}

	keys := []string{}
	for _, e := range taxonomy.Entries {
		keys = append(keys, e.Key)
	}
	keys = append(keys, taxonomy.OtherEarning, taxonomy.OtherDeduction, taxonomy.OtherInfo)
	printed := 0
	entry := func(label, text string) {
		if printed > 0 {
			fmt.Fprintln(out)
		}
		printed++
		fmt.Fprintln(out, label)
		fmt.Fprint(out, wrap(text, "  ", 78))
	}
	for _, k := range keys {
		e, _ := taxonomy.Lookup(k)
		if !matches(append([]string{e.LabelEN, e.LabelIT}, e.Keywords...)...) {
			continue
		}
		label := i18n.ItemLabel(l, k)
		if l != i18n.IT && e.LabelIT != label {
			label += " (" + e.LabelIT + ")"
		}
		entry(label, i18n.ItemText(l, k))
	}
	for _, term := range i18n.Terms {
		label := i18n.T(l, "term."+term)
		if !matches(label, i18n.T(i18n.IT, "term."+term), i18n.T(i18n.EN, "term."+term)) {
			continue
		}
		entry(label, i18n.T(l, "term."+term+".text"))
	}
	if printed == 0 {
		fmt.Fprintln(out, i18n.T(l, "glossary.no_match"))
		return nil
	}
	fmt.Fprintf(out, "\n%s\n", wrap(i18n.T(l, "glossary.intro"), "", 78))
	return nil
}

// wrap breaks text into lines of at most width runes, each starting with
// indent, and ends it with a newline.
func wrap(text, indent string, width int) string {
	var b strings.Builder
	line := indent
	for _, w := range strings.Fields(text) {
		if line != indent && len([]rune(line))+1+len([]rune(w)) > width {
			b.WriteString(line + "\n")
			line = indent
		}
		if line != indent {
			line += " "
		}
		line += w
	}
	b.WriteString(line + "\n")
	return b.String()
}
