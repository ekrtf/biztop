package service

// Customers: revenue per client over the whole FEC history. The client name
// is the part of the entry libelle after the last " - " separator, e.g.
// "Facturation 202604-160 - CHAUVIN PARIS", remapped through the
// client_aliases table in rules.yml.

import (
	"fmt"
	"sort"
	"strings"

	"biztop/internal/domain"
	"biztop/internal/repository"
)

// chartSeries is the number of clients drawn individually on the chart;
// the remaining ones are folded into an "Autres" series.
const chartSeries = 8

type Customers struct {
	FecsDir   string
	RulesPath string
}

type CustomerRow struct {
	Name     string    `json:"name"`
	Invoices int       `json:"invoices"`
	ByYear   []float64 `json:"by_year"` // aligned with Years
	Total    float64   `json:"total"`
}

type CustomerSeries struct {
	Name string    `json:"name"`
	Data []float64 `json:"data"` // aligned with Months
}

type CustomersResult struct {
	Years     []int            `json:"years"`
	Months    []string         `json:"months"` // "2006-01", continuous range
	Customers []CustomerRow    `json:"customers"`
	Series    []CustomerSeries `json:"series"` // top clients + "Autres"
	Total     float64          `json:"total"`
}

// CustomerName extracts the client from a revenue entry libelle.
func CustomerName(libelle string) string {
	if i := strings.LastIndex(libelle, " - "); i >= 0 {
		if name := strings.TrimSpace(libelle[i+3:]); name != "" {
			return name
		}
	}
	return "(inconnu)"
}

// Compute aggregates every chiffre d'affaires entry by client: totals per
// exercice for the table and per month for the chart.
func (c Customers) Compute() (*CustomersResult, error) {
	entries, err := repository.LoadEntries(c.FecsDir)
	if err != nil {
		return nil, err
	}
	rules, err := repository.LoadRules(c.RulesPath)
	if err != nil {
		return nil, err
	}

	type acc struct {
		byYear   map[int]float64
		byMonth  map[int]float64 // key: year*12 + month-1
		invoices map[string]bool // distinct libelles
	}
	accs := map[string]*acc{}
	minKey, maxKey := 0, 0
	seen := map[int]bool{}
	var years []int
	for _, e := range entries {
		if !e.IsIncome() {
			continue
		}
		name := CustomerName(e.Libelle)
		if alias, ok := rules.ClientAliases[name]; ok {
			name = alias
		}
		a := accs[name]
		if a == nil {
			a = &acc{byYear: map[int]float64{}, byMonth: map[int]float64{}, invoices: map[string]bool{}}
			accs[name] = a
		}
		key := e.Year*12 + e.Month - 1
		if minKey == 0 || key < minKey {
			minKey = key
		}
		if key > maxKey {
			maxKey = key
		}
		a.byYear[e.Year] += e.IncomeAmount()
		a.byMonth[key] += e.IncomeAmount()
		a.invoices[e.Libelle] = true
		if !seen[e.Year] {
			seen[e.Year] = true
			years = append(years, e.Year)
		}
	}
	sort.Ints(years)

	out := &CustomersResult{Years: years, Months: []string{}, Customers: []CustomerRow{}, Series: []CustomerSeries{}}
	if len(accs) == 0 {
		return out, nil
	}
	for key := minKey; key <= maxKey; key++ {
		out.Months = append(out.Months, fmt.Sprintf("%04d-%02d", key/12, key%12+1))
	}

	for name, a := range accs {
		row := CustomerRow{Name: name, Invoices: len(a.invoices)}
		for _, y := range years {
			v := domain.Round2(a.byYear[y])
			row.ByYear = append(row.ByYear, v)
			row.Total += v
		}
		row.Total = domain.Round2(row.Total)
		out.Customers = append(out.Customers, row)
		out.Total += row.Total
	}
	out.Total = domain.Round2(out.Total)
	sort.SliceStable(out.Customers, func(a, b int) bool {
		if out.Customers[a].Total != out.Customers[b].Total {
			return out.Customers[a].Total > out.Customers[b].Total
		}
		return out.Customers[a].Name < out.Customers[b].Name
	})

	monthly := func(names []string) []float64 {
		data := make([]float64, maxKey-minKey+1)
		for _, n := range names {
			for key, v := range accs[n].byMonth {
				data[key-minKey] += v
			}
		}
		for i := range data {
			data[i] = domain.Round2(data[i])
		}
		return data
	}
	var rest []string
	for i, row := range out.Customers {
		if i < chartSeries {
			out.Series = append(out.Series, CustomerSeries{Name: row.Name, Data: monthly([]string{row.Name})})
		} else {
			rest = append(rest, row.Name)
		}
	}
	if len(rest) > 0 {
		out.Series = append(out.Series, CustomerSeries{Name: "Autres", Data: monthly(rest)})
	}
	return out, nil
}
