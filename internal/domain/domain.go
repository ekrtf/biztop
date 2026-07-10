// Package domain holds the core entities and accounting rules of BizTop.
package domain

import (
	"math"
	"time"
)

// Entry is one FEC line.
type Entry struct {
	Date        time.Time
	Year        int
	Month       int // 1-12
	Libelle     string
	Compte      string
	CompteLabel string
	Debit       float64
	Credit      float64
}

// IsExpense reports whether the account is an operating charge:
// class 6 except 695 (impot sur les benefices).
func (e Entry) IsExpense() bool {
	return len(e.Compte) > 0 && e.Compte[0] == '6' && !hasPrefix(e.Compte, "695")
}

// IsIncome reports whether the account is chiffre d'affaires (706/707).
func (e Entry) IsIncome() bool {
	return hasPrefix(e.Compte, "706") || hasPrefix(e.Compte, "707")
}

// ExpenseAmount is the net charge (debit minus credit, avoirs deducted).
func (e Entry) ExpenseAmount() float64 { return e.Debit - e.Credit }

// IncomeAmount is the net revenue (credit minus debit).
func (e Entry) IncomeAmount() float64 { return e.Credit - e.Debit }

// AccountRow is one line of the compte de resultat: an account with its
// monthly net amounts.
type AccountRow struct {
	Compte  string      `json:"compte"`
	Libelle string      `json:"libelle"`
	Months  [12]float64 `json:"months"`
	Total   float64     `json:"total"`
}

// Tx is a transaction as exposed by the API.
type Tx struct {
	Date        string  `json:"date"`
	Libelle     string  `json:"libelle"`
	Compte      string  `json:"compte"`
	CompteLabel string  `json:"compte_label"`
	Debit       float64 `json:"debit"`
	Credit      float64 `json:"credit"`
}

// FeeTx is an expense counted (fully or partially) as management fees.
type FeeTx struct {
	Tx
	Amount float64 `json:"amount"`
	Ratio  float64 `json:"ratio"`
	Fee    float64 `json:"fee"`
}

// Objective is one row of the 5-year plan.
type Objective struct {
	Year       int     `json:"year"`
	RevenueMin float64 `json:"revenue_min"`
	RevenueMax float64 `json:"revenue_max"`
	MarginMin  float64 `json:"margin_min"`
	MarginMax  float64 `json:"margin_max"`
	ProfitMin  float64 `json:"profit_min"`
	ProfitMax  float64 `json:"profit_max"`
	Team       string  `json:"team"`
}

// Rules is the content of rules.yml, the single source of truth for the
// business rules.
type Rules struct {
	ManagementFees FeesConfig  `yaml:"management_fees" json:"management_fees"`
	AttioTypes     []AttioType `yaml:"attio_types" json:"attio_types"`
}

// FeesConfig describes which expenses count as management fees and for
// which portion (ratio 0..1). ExcludePatterns veto a transaction even when
// another rule matches it.
type FeesConfig struct {
	LibellePatterns []string     `yaml:"libelle_patterns" json:"libelle_patterns"` // case-insensitive regexes on the libelle, counted in full
	Comptes         []CompteRule `yaml:"comptes" json:"comptes"`
	ExcludePatterns []string     `yaml:"exclude_patterns" json:"exclude_patterns"`
}

type CompteRule struct {
	Compte string  `yaml:"compte" json:"compte"` // whole plan comptable account
	Ratio  float64 `yaml:"ratio" json:"ratio"`
}

// AttioType is one Davai revenue type used to classify CRM deals.
// Billing is "one-shot" (amount = deal value) or "mrr" (amount = the
// monthly recurring revenue from the Attio MRR field).
type AttioType struct {
	Name        string `yaml:"name" json:"name"`
	Billing     string `yaml:"billing" json:"billing"`
	Description string `yaml:"description" json:"description"`
}

func Round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func NewTx(e Entry) Tx {
	return Tx{
		Date:        e.Date.Format("02/01/2006"),
		Libelle:     e.Libelle,
		Compte:      e.Compte,
		CompteLabel: e.CompteLabel,
		Debit:       e.Debit,
		Credit:      e.Credit,
	}
}
