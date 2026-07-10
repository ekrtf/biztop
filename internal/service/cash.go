package service

// Cash reconciliation: which part of the invoiced CA actually hit the bank.
// The simplified FEC has no lettrage column, so lines are grouped into
// ecritures by (date, libelle), then classified:
//   invoice     411 debit + 706/707 credit (HT = income, TTC = 411 debit)
//   settlement  411 credit + 512 debit (GoCardless fees are netted inside)
//   direct cash 706/707 credit + 512 debit, no 411: invoiced and paid at once
// Avoirs (411 credit + 706 debit, no bank line) are skipped: their HT already
// nets the CA, so the aggregate CA - encaisse stays right without them.
// Settlements are matched to invoices first by the invoice reference digits
// found in the settlement libelle, then by exact amount, oldest first.

import (
	"sort"
	"strings"
	"time"

	"biztop/internal/domain"
)

type cashInvoice struct {
	date    time.Time
	year    int
	month   int
	ht      float64
	ttc     float64
	settled float64
	ref     string // digits of the libelle, e.g. "202601133"
}

type cashSettlement struct {
	date   time.Time
	year   int
	month  int
	amount float64
	digits string
}

// reconcileCash returns the collected HT per invoice year, plus the monthly
// collected HT for chartYear (by settlement date, invoices of that year only).
func reconcileCash(entries []domain.Entry, chartYear int) (map[int]float64, [12]float64) {
	type ecriture struct {
		date                              time.Time
		year, month                       int
		libelle                           string
		income, debit411, credit411, bank float64
	}
	type key struct {
		date    time.Time
		libelle string
	}
	byKey := map[key]*ecriture{}
	var order []key
	for _, e := range entries {
		k := key{e.Date, e.Libelle}
		ec := byKey[k]
		if ec == nil {
			ec = &ecriture{date: e.Date, year: e.Year, month: e.Month, libelle: e.Libelle}
			byKey[k] = ec
			order = append(order, k)
		}
		switch {
		case e.IsIncome():
			ec.income += e.IncomeAmount()
		case strings.HasPrefix(e.Compte, "411"):
			ec.debit411 += e.Debit
			ec.credit411 += e.Credit
		case strings.HasPrefix(e.Compte, "512"):
			ec.bank += e.Debit
		}
	}

	var invoices []*cashInvoice
	var settlements []*cashSettlement
	var monthly [12]float64
	cash := map[int]float64{}
	for _, k := range order {
		ec := byKey[k]
		net411 := ec.credit411 - ec.debit411
		switch {
		case ec.income > 0 && ec.debit411 > 0:
			invoices = append(invoices, &cashInvoice{
				date: ec.date, year: ec.year, month: ec.month,
				ht: ec.income, ttc: ec.debit411, ref: invoiceRef(ec.libelle),
			})
		case ec.income > 0 && ec.bank > 0:
			// Paid on the spot, no receivable: fully collected.
			cash[ec.year] += ec.income
			if ec.year == chartYear {
				monthly[ec.month-1] += ec.income
			}
		case net411 > 0 && ec.bank > 0:
			settlements = append(settlements, &cashSettlement{
				date: ec.date, year: ec.year, month: ec.month,
				amount: net411, digits: digitsOf(ec.libelle),
			})
		}
	}
	sort.SliceStable(invoices, func(a, b int) bool { return invoices[a].date.Before(invoices[b].date) })
	sort.SliceStable(settlements, func(a, b int) bool { return settlements[a].date.Before(settlements[b].date) })

	allocate := func(s *cashSettlement, inv *cashInvoice, amount float64) {
		inv.settled += amount
		s.amount -= amount
		if inv.year == chartYear && s.year == chartYear {
			monthly[s.month-1] += amount / inv.ttc * inv.ht
		}
	}
	// Pass 1: the settlement libelle carries the invoice number.
	for _, s := range settlements {
		for _, inv := range invoices {
			if s.amount <= 0 {
				break
			}
			if inv.ref == "" || inv.settled >= inv.ttc || !strings.Contains(s.digits, inv.ref) {
				continue
			}
			allocate(s, inv, min(s.amount, inv.ttc-inv.settled))
		}
	}
	// Pass 2: exact amount against the oldest open invoice.
	for _, s := range settlements {
		if s.amount <= 0 {
			continue
		}
		for _, inv := range invoices {
			if inv.settled < inv.ttc && abs(inv.ttc-inv.settled-s.amount) < 0.01 {
				allocate(s, inv, s.amount)
				break
			}
		}
	}

	for _, inv := range invoices {
		cash[inv.year] += min(inv.settled, inv.ttc) / inv.ttc * inv.ht
	}
	for y := range cash {
		cash[y] = domain.Round2(cash[y])
	}
	for i := range monthly {
		monthly[i] = domain.Round2(monthly[i])
	}
	return cash, monthly
}

// invoiceRef is the digits of the invoice number ("Facturation 202601-133 -
// CHAUVIN PARIS" gives "202601133"). The client name after the last " - " is
// dropped so a digit in it cannot pollute the reference. Empty when too
// short to be a reference.
func invoiceRef(libelle string) string {
	if i := strings.LastIndex(libelle, " - "); i >= 0 {
		libelle = libelle[:i]
	}
	d := digitsOf(libelle)
	if len(d) < 6 {
		return ""
	}
	return d
}

func digitsOf(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
