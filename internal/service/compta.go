// Package service implements the business logic on top of the repository.
package service

import (
	"sort"

	"biztop/internal/domain"
	"biztop/internal/repository"
)

// Compta serves the compte de resultat and the transaction lists.
type Compta struct {
	FecsDir string
}

type Monthly struct {
	CA       [12]float64 `json:"ca"`
	Charges  [12]float64 `json:"charges"`
	Resultat [12]float64 `json:"resultat"`
}

type Totals struct {
	CA       float64 `json:"ca"`
	Charges  float64 `json:"charges"`
	Resultat float64 `json:"resultat"`
}

type YearData struct {
	Years   []int               `json:"years"`
	Year    int                 `json:"year"`
	Revenue []domain.AccountRow `json:"revenue"`
	Charges []domain.AccountRow `json:"charges"`
	Monthly Monthly             `json:"monthly"`
	Totals  Totals              `json:"totals"`
}

type TxList struct {
	Compte       string      `json:"compte"`
	Libelle      string      `json:"libelle"`
	Year         int         `json:"year"`
	Month        int         `json:"month"`
	Transactions []domain.Tx `json:"transactions"`
	TotalDebit   float64     `json:"total_debit"`
	TotalCredit  float64     `json:"total_credit"`
}

// Years lists the exercices present in the FEC entries, ascending.
func Years(entries []domain.Entry) []int {
	seen := map[int]bool{}
	for _, e := range entries {
		seen[e.Year] = true
	}
	var ys []int
	for y := range seen {
		ys = append(ys, y)
	}
	sort.Ints(ys)
	return ys
}

// ResolveYear maps 0 ("not specified") to the latest exercice.
func ResolveYear(ys []int, year int) int {
	if year != 0 || len(ys) == 0 {
		return year
	}
	return ys[len(ys)-1]
}

// Data aggregates one exercice into the compte de resultat.
func (c Compta) Data(year int) (*YearData, error) {
	entries, err := repository.LoadEntries(c.FecsDir)
	if err != nil {
		return nil, err
	}
	ys := Years(entries)
	year = ResolveYear(ys, year)
	revenue, charges := aggregateYear(entries, year)

	out := &YearData{Years: ys, Year: year, Revenue: revenue, Charges: charges}
	for _, row := range revenue {
		for m, v := range row.Months {
			out.Monthly.CA[m] += v
		}
		out.Totals.CA += row.Total
	}
	for _, row := range charges {
		for m, v := range row.Months {
			out.Monthly.Charges[m] += v
		}
		out.Totals.Charges += row.Total
	}
	for m := range out.Monthly.Resultat {
		out.Monthly.CA[m] = domain.Round2(out.Monthly.CA[m])
		out.Monthly.Charges[m] = domain.Round2(out.Monthly.Charges[m])
		out.Monthly.Resultat[m] = domain.Round2(out.Monthly.CA[m] - out.Monthly.Charges[m])
	}
	out.Totals.CA = domain.Round2(out.Totals.CA)
	out.Totals.Charges = domain.Round2(out.Totals.Charges)
	out.Totals.Resultat = domain.Round2(out.Totals.CA - out.Totals.Charges)
	return out, nil
}

func aggregateYear(entries []domain.Entry, year int) (revenue, charges []domain.AccountRow) {
	revenue, charges = []domain.AccountRow{}, []domain.AccountRow{}
	revIdx := map[string]int{}
	expIdx := map[string]int{}
	add := func(list *[]domain.AccountRow, idx map[string]int, e domain.Entry, amount float64) {
		i, ok := idx[e.Compte]
		if !ok {
			i = len(*list)
			idx[e.Compte] = i
			*list = append(*list, domain.AccountRow{Compte: e.Compte, Libelle: e.CompteLabel})
		}
		(*list)[i].Months[e.Month-1] += amount
	}
	for _, e := range entries {
		if e.Year != year {
			continue
		}
		switch {
		case e.IsIncome():
			add(&revenue, revIdx, e, e.IncomeAmount())
		case e.IsExpense():
			add(&charges, expIdx, e, e.ExpenseAmount())
		}
	}
	for _, list := range [][]domain.AccountRow{revenue, charges} {
		for i := range list {
			total := 0.0
			for m := range list[i].Months {
				list[i].Months[m] = domain.Round2(list[i].Months[m])
				total += list[i].Months[m]
			}
			list[i].Total = domain.Round2(total)
		}
	}
	sort.SliceStable(revenue, func(a, b int) bool { return revenue[a].Total > revenue[b].Total })
	sort.SliceStable(charges, func(a, b int) bool { return charges[a].Total > charges[b].Total })
	return revenue, charges
}

// Transactions lists an exercice's entries, optionally restricted to one
// account and one month (0 means whole year), ordered by date.
func (c Compta) Transactions(year int, compte string, month int) (*TxList, error) {
	entries, err := repository.LoadEntries(c.FecsDir)
	if err != nil {
		return nil, err
	}
	var filtered []domain.Entry
	for _, e := range entries {
		if e.Year == year && (compte == "" || e.Compte == compte) && (month == 0 || e.Month == month) {
			filtered = append(filtered, e)
		}
	}
	sort.SliceStable(filtered, func(a, b int) bool { return filtered[a].Date.Before(filtered[b].Date) })

	out := &TxList{Compte: compte, Year: year, Month: month, Transactions: []domain.Tx{}}
	for _, e := range filtered {
		out.Transactions = append(out.Transactions, domain.NewTx(e))
		out.TotalDebit += e.Debit
		out.TotalCredit += e.Credit
	}
	if compte != "" && len(filtered) > 0 {
		out.Libelle = filtered[0].CompteLabel
	}
	out.TotalDebit = domain.Round2(out.TotalDebit)
	out.TotalCredit = domain.Round2(out.TotalCredit)
	return out, nil
}
