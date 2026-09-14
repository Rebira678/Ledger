// Package web renders the server-side dashboard (D-09): embedded html/template
// pages and a dependency-free inline SVG category chart.
package web

import (
	"embed"
	"fmt"
	"html/template"
	"math"
	"net/http"
	"sort"
)

//go:embed templates/*.html
var templatesFS embed.FS

var tmpl = template.Must(template.ParseFS(templatesFS, "templates/*.html"))

// TxnRow is one transaction row in the dashboard table.
type TxnRow struct {
	ID           string
	Amount       string
	Direction    string
	Counterparty string
	Category     string
	Source       string
	OccurredAt   string
}

// ClarifRow is one pending clarifying question.
type ClarifRow struct {
	ID            string
	TransactionID string
	Question      string
	Options       []string
}

// ReportRow is one weekly report in the list.
type ReportRow struct {
	ID     string
	Period string
	Status string
}

// PageData is the data contract for all templates.
type PageData struct {
	Title       string
	ErrorMsg    string
	Balance     string
	WeekSpend   string
	TopCategory string
	PendingCnt  int
	SVGChart    template.HTML
	Txns        []TxnRow
	Clarifs     []ClarifRow
	Reports     []ReportRow
	Narrative   string
	WoWRaw      map[string]string
	ReportMeta  string
}

// Render writes a template page; a failure is a plain 500 with context.
func Render(w http.ResponseWriter, name string, data PageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

// SpendByCategorySVG renders a simple donut chart of category totals —
// zero JS, server-generated (D-09).
func SpendByCategorySVG(totals map[string]float64) template.HTML {
	cats := make([]string, 0, len(totals))
	for c := range totals {
		cats = append(cats, c)
	}
	sort.Strings(cats)
	total := 0.0
	for _, v := range totals {
		total += v
	}
	if total <= 0 {
		return template.HTML(`<p class="muted">No spending recorded this week.</p>`)
	}

	const cx, cy, r, sw = 90.0, 90.0, 60.0, 36.0
	circ := 2 * math.Pi * r
	offset := 0.0
	palette := []string{"#2563eb", "#16a34a", "#dc2626", "#d97706", "#7c3aed", "#0891b2", "#db2777", "#65a30d"}
	svg := fmt.Sprintf(`<svg viewBox="0 0 360 200" xmlns="http://www.w3.org/2000/svg" role="img" aria-label="Spending by category"><g transform="translate(100,100)">`)
	i := 0
	legend := `<g transform="translate(210,30)" font-size="11" font-family="sans-serif">`
	for _, c := range cats {
		frac := totals[c] / total
		color := palette[i%len(palette)]
		svg += fmt.Sprintf(
			`<circle r="%g" fill="none" stroke="%s" stroke-width="%g" stroke-dasharray="%g %g" transform="rotate(%g)" stroke-dashoffset="0"></circle>`,
			r, color, sw, frac*circ, circ-frac*circ, -90+offset*360,
		)
		legend += fmt.Sprintf(`<rect x="0" y="%d" width="10" height="10" fill="%s"></rect><text x="16" y="%d" fill="#374151">%s — %.0f ETB</text>`,
			i*18, color, i*18+9, template.HTMLEscapeString(c), totals[c])
		offset += frac
		i++
	}
	legend += `</g>`
	svg += legend + `</g></svg>`
	return template.HTML(svg)
}
