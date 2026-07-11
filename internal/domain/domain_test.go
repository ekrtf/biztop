package domain

import (
	"testing"
	"time"
)

func TestEntryAccountingClassificationAndAmounts(t *testing.T) {
	tests := []struct {
		name      string
		entry     Entry
		expense   bool
		income    bool
		expenseAm float64
		incomeAm  float64
	}{
		{
			name:      "operating expense",
			entry:     Entry{Compte: "613600", Debit: 120, Credit: 20},
			expense:   true,
			expenseAm: 100,
			incomeAm:  -100,
		},
		{
			name:      "corporate tax is not operating expense",
			entry:     Entry{Compte: "695000", Debit: 50},
			expense:   false,
			expenseAm: 50,
			incomeAm:  -50,
		},
		{
			name:      "service revenue",
			entry:     Entry{Compte: "706000", Debit: 10, Credit: 250},
			income:    true,
			expenseAm: -240,
			incomeAm:  240,
		},
		{
			name:      "goods revenue",
			entry:     Entry{Compte: "707100", Credit: 80},
			income:    true,
			expenseAm: -80,
			incomeAm:  80,
		},
		{
			name:      "empty account",
			entry:     Entry{Debit: 1, Credit: 2},
			expenseAm: -1,
			incomeAm:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.entry.IsExpense(); got != tt.expense {
				t.Fatalf("IsExpense() = %v, want %v", got, tt.expense)
			}
			if got := tt.entry.IsIncome(); got != tt.income {
				t.Fatalf("IsIncome() = %v, want %v", got, tt.income)
			}
			if got := tt.entry.ExpenseAmount(); got != tt.expenseAm {
				t.Fatalf("ExpenseAmount() = %v, want %v", got, tt.expenseAm)
			}
			if got := tt.entry.IncomeAmount(); got != tt.incomeAm {
				t.Fatalf("IncomeAmount() = %v, want %v", got, tt.incomeAm)
			}
		})
	}
}

func TestTaxConfigTax(t *testing.T) {
	// The scale from docs/IS_CHEAT_SHEET.md: 15% up to 42 500, then 25%,
	// reduced rate reserved to companies with a CA under 10M.
	scale := TaxConfig{ReducedRate: 0.15, ReducedCap: 42500, StandardRate: 0.25, RevenueCap: 10000000}
	tests := []struct {
		profit, ca, want float64
	}{
		{profit: 100000, ca: 650000, want: 20750},   // 42500*15% + 57500*25%
		{profit: 30000, ca: 650000, want: 4500},     // all under the cap
		{profit: 100000, ca: 12000000, want: 25000}, // CA too high, no reduced rate
		{profit: -5000, ca: 650000, want: 0},        // a loss owes nothing
		{profit: 0, ca: 650000, want: 0},
	}
	for _, tt := range tests {
		if got := scale.Tax(tt.profit, tt.ca); got != tt.want {
			t.Fatalf("Tax(%v, %v) = %v, want %v", tt.profit, tt.ca, got, tt.want)
		}
	}
}

func TestRound2(t *testing.T) {
	if got := Round2(12.345); got != 12.35 {
		t.Fatalf("Round2(12.345) = %v, want 12.35", got)
	}
	if got := Round2(12.344); got != 12.34 {
		t.Fatalf("Round2(12.344) = %v, want 12.34", got)
	}
}

func TestNewTx(t *testing.T) {
	entry := Entry{
		Date:        time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC),
		Libelle:     "Invoice",
		Compte:      "706000",
		CompteLabel: "Prestations",
		Debit:       1,
		Credit:      2,
	}

	tx := NewTx(entry)
	if tx.Date != "10/07/2026" || tx.Libelle != entry.Libelle || tx.Compte != entry.Compte || tx.CompteLabel != entry.CompteLabel || tx.Debit != 1 || tx.Credit != 2 {
		t.Fatalf("NewTx() = %+v", tx)
	}
}
