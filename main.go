package main

// BizTop - Business monitoring dashboard
// French accounting, compte de resultat par exercice, drill-down transactions

import (
	"embed"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

//go:embed gui
var guiFS embed.FS

const fecsDir = "fecs"

type entry struct {
	Date        time.Time
	Year        int
	Month       int // 1-12
	Libelle     string
	Compte      string
	CompteLabel string
	Debit       float64
	Credit      float64
}

func isExpense(compte string) bool {
	return strings.HasPrefix(compte, "6") && !strings.HasPrefix(compte, "695")
}

func isIncome(compte string) bool {
	return strings.HasPrefix(compte, "706") || strings.HasPrefix(compte, "707")
}

func parseAmount(s string) float64 {
	s = strings.TrimSpace(strings.ReplaceAll(s, ",", "."))
	if s == "" {
		return 0
	}
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func loadFecs() ([]entry, error) {
	files, err := os.ReadDir(fecsDir)
	if err != nil {
		return nil, err
	}
	var rows []entry
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".csv") {
			continue
		}
		f, err := os.Open(filepath.Join(fecsDir, file.Name()))
		if err != nil {
			return nil, err
		}
		reader := csv.NewReader(f)
		reader.Comma = ';'
		reader.FieldsPerRecord = -1
		records, err := reader.ReadAll()
		f.Close()
		if err != nil {
			return nil, err
		}
		if len(records) == 0 {
			continue
		}
		col := map[string]int{}
		for i, name := range records[0] {
			col[strings.TrimSpace(strings.TrimPrefix(name, "\ufeff"))] = i
		}
		get := func(rec []string, name string) string {
			if i, ok := col[name]; ok && i < len(rec) {
				return strings.TrimSpace(rec[i])
			}
			return ""
		}
		for _, rec := range records[1:] {
			d, err := time.Parse("02/01/2006", get(rec, "Date"))
			if err != nil {
				continue
			}
			rows = append(rows, entry{
				Date:        d,
				Year:        d.Year(),
				Month:       int(d.Month()),
				Libelle:     get(rec, "Libelle"),
				Compte:      get(rec, "Compte"),
				CompteLabel: get(rec, "Libelle du compte"),
				Debit:       parseAmount(get(rec, "Debit")),
				Credit:      parseAmount(get(rec, "Credit")),
			})
		}
	}
	return rows, nil
}

func years(rows []entry) []int {
	seen := map[int]bool{}
	for _, r := range rows {
		seen[r.Year] = true
	}
	var ys []int
	for y := range seen {
		ys = append(ys, y)
	}
	sort.Ints(ys)
	return ys
}

// accountRow is one line of the compte de resultat: an account with its
// monthly net amounts (expenses: debit - credit, revenue: credit - debit).
type accountRow struct {
	Compte  string      `json:"compte"`
	Libelle string      `json:"libelle"`
	Months  [12]float64 `json:"months"`
	Total   float64     `json:"total"`
}

func aggregateYear(rows []entry, year int) (revenue, charges []accountRow) {
	revIdx := map[string]int{}
	expIdx := map[string]int{}
	for _, r := range rows {
		if r.Year != year {
			continue
		}
		switch {
		case isIncome(r.Compte):
			i, ok := revIdx[r.Compte]
			if !ok {
				i = len(revenue)
				revIdx[r.Compte] = i
				revenue = append(revenue, accountRow{Compte: r.Compte, Libelle: r.CompteLabel})
			}
			revenue[i].Months[r.Month-1] += r.Credit - r.Debit
		case isExpense(r.Compte):
			i, ok := expIdx[r.Compte]
			if !ok {
				i = len(charges)
				expIdx[r.Compte] = i
				charges = append(charges, accountRow{Compte: r.Compte, Libelle: r.CompteLabel})
			}
			charges[i].Months[r.Month-1] += r.Debit - r.Credit
		}
	}
	for _, list := range [][]accountRow{revenue, charges} {
		for i := range list {
			total := 0.0
			for m := range list[i].Months {
				list[i].Months[m] = round2(list[i].Months[m])
				total += list[i].Months[m]
			}
			list[i].Total = round2(total)
		}
	}
	sort.SliceStable(revenue, func(a, b int) bool { return revenue[a].Total > revenue[b].Total })
	sort.SliceStable(charges, func(a, b int) bool { return charges[a].Total > charges[b].Total })
	return revenue, charges
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.Encode(v)
}

func apiData(w http.ResponseWriter, r *http.Request) {
	rows, err := loadFecs()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ys := years(rows)
	if len(ys) == 0 {
		http.Error(w, "no FEC data", http.StatusInternalServerError)
		return
	}
	year := ys[len(ys)-1]
	if y, err := strconv.Atoi(r.URL.Query().Get("year")); err == nil {
		year = y
	}
	revenue, charges := aggregateYear(rows, year)
	if revenue == nil {
		revenue = []accountRow{}
	}
	if charges == nil {
		charges = []accountRow{}
	}

	var caMonthly, chMonthly, resMonthly [12]float64
	totalCA, totalCharges := 0.0, 0.0
	for _, row := range revenue {
		for m, v := range row.Months {
			caMonthly[m] += v
		}
		totalCA += row.Total
	}
	for _, row := range charges {
		for m, v := range row.Months {
			chMonthly[m] += v
		}
		totalCharges += row.Total
	}
	for m := range resMonthly {
		caMonthly[m] = round2(caMonthly[m])
		chMonthly[m] = round2(chMonthly[m])
		resMonthly[m] = round2(caMonthly[m] - chMonthly[m])
	}

	writeJSON(w, map[string]any{
		"years":   ys,
		"year":    year,
		"revenue": revenue,
		"charges": charges,
		"monthly": map[string]any{
			"ca":       caMonthly,
			"charges":  chMonthly,
			"resultat": resMonthly,
		},
		"totals": map[string]float64{
			"ca":       round2(totalCA),
			"charges":  round2(totalCharges),
			"resultat": round2(totalCA - totalCharges),
		},
	})
}

func apiTransactions(w http.ResponseWriter, r *http.Request) {
	rows, err := loadFecs()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	q := r.URL.Query()
	year, _ := strconv.Atoi(q.Get("year"))
	month, _ := strconv.Atoi(q.Get("month")) // 0 means whole year
	compte := q.Get("compte")
	if compte == "" || year == 0 {
		http.Error(w, "compte and year are required", http.StatusBadRequest)
		return
	}

	type tx struct {
		Date    string  `json:"date"`
		Libelle string  `json:"libelle"`
		Debit   float64 `json:"debit"`
		Credit  float64 `json:"credit"`
	}
	txs := []tx{}
	libelle := ""
	totalDebit, totalCredit := 0.0, 0.0
	var filtered []entry
	for _, e := range rows {
		if e.Compte == compte && e.Year == year && (month == 0 || e.Month == month) {
			filtered = append(filtered, e)
		}
	}
	sort.SliceStable(filtered, func(a, b int) bool { return filtered[a].Date.Before(filtered[b].Date) })
	for _, e := range filtered {
		libelle = e.CompteLabel
		totalDebit += e.Debit
		totalCredit += e.Credit
		txs = append(txs, tx{
			Date:    e.Date.Format("02/01/2006"),
			Libelle: e.Libelle,
			Debit:   e.Debit,
			Credit:  e.Credit,
		})
	}

	writeJSON(w, map[string]any{
		"compte":       compte,
		"libelle":      libelle,
		"year":         year,
		"month":        month,
		"transactions": txs,
		"total_debit":  round2(totalDebit),
		"total_credit": round2(totalCredit),
	})
}

func index(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	page, err := guiFS.ReadFile("gui/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(page)
}

func main() {
	static, err := fs.Sub(guiFS, "gui")
	if err != nil {
		log.Fatal(err)
	}
	http.HandleFunc("/", index)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(static))))
	http.HandleFunc("/api/data", apiData)
	http.HandleFunc("/api/transactions", apiTransactions)
	fmt.Println("BizTop demarre -> http://localhost:5055")
	log.Fatal(http.ListenAndServe(":5055", nil))
}
