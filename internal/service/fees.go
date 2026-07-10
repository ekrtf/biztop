package service

// Management fees: expenses that are legally company charges but are really
// part of the manager's compensation, so they are added back to the result
// to see the true business profit. Each matching rule carries a ratio: only
// that portion of the expense counts as management fees.

import (
	"fmt"
	"regexp"
	"sort"

	"biztop/internal/domain"
	"biztop/internal/repository"
)

type Fees struct {
	FecsDir   string
	RulesPath string
}

type FeesResult struct {
	Years          []int             `json:"years"`
	Year           int               `json:"year"`
	Config         domain.FeesConfig `json:"config"`
	Transactions   []domain.FeeTx    `json:"transactions"`
	Monthly        [12]float64       `json:"monthly"`
	Total          float64           `json:"total"`
	Resultat       float64           `json:"resultat"`
	ResultatAjuste float64           `json:"resultat_ajuste"`
}

func (f Fees) Compute(year int) (*FeesResult, error) {
	entries, err := repository.LoadEntries(f.FecsDir)
	if err != nil {
		return nil, err
	}
	rules, err := repository.LoadRules(f.RulesPath)
	if err != nil {
		return nil, err
	}
	cfg := rules.ManagementFees
	type patternRule struct {
		re    *regexp.Regexp
		ratio float64
	}
	var patterns []patternRule
	for _, p := range cfg.LibellePatterns {
		re, err := regexp.Compile("(?i)" + p.Pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid pattern %q in %s: %w", p.Pattern, f.RulesPath, err)
		}
		patterns = append(patterns, patternRule{re, p.Ratio})
	}
	var excludes []*regexp.Regexp
	for _, p := range cfg.ExcludePatterns {
		re, err := regexp.Compile("(?i)" + p)
		if err != nil {
			return nil, fmt.Errorf("invalid exclude pattern %q in %s: %w", p, f.RulesPath, err)
		}
		excludes = append(excludes, re)
	}
	compteRatio := map[string]float64{}
	for _, c := range cfg.Comptes {
		compteRatio[c.Compte] = c.Ratio
	}
	// Excludes veto everything; otherwise the highest matching ratio wins.
	ratioFor := func(e domain.Entry) float64 {
		for _, re := range excludes {
			if re.MatchString(e.Libelle) {
				return 0
			}
		}
		ratio := 0.0
		if r, ok := compteRatio[e.Compte]; ok && r > ratio {
			ratio = r
		}
		for _, p := range patterns {
			if p.ratio > ratio && p.re.MatchString(e.Libelle) {
				ratio = p.ratio
			}
		}
		return ratio
	}

	ys := Years(entries)
	out := &FeesResult{Years: ys, Year: ResolveYear(ys, year), Config: cfg, Transactions: []domain.FeeTx{}}
	totalCA, totalCharges := 0.0, 0.0
	type match struct {
		entry domain.Entry
		fee   domain.FeeTx
	}
	var matched []match
	for _, e := range entries {
		if e.Year != out.Year {
			continue
		}
		if e.IsIncome() {
			totalCA += e.IncomeAmount()
		}
		if !e.IsExpense() {
			continue
		}
		totalCharges += e.ExpenseAmount()
		if ratio := ratioFor(e); ratio > 0 {
			fee := e.ExpenseAmount() * ratio
			matched = append(matched, match{e, domain.FeeTx{
				Tx:     domain.NewTx(e),
				Amount: domain.Round2(e.ExpenseAmount()),
				Ratio:  ratio,
				Fee:    domain.Round2(fee),
			}})
			out.Monthly[e.Month-1] += fee
			out.Total += fee
		}
	}
	sort.SliceStable(matched, func(a, b int) bool {
		return matched[a].entry.Date.Before(matched[b].entry.Date)
	})
	for _, m := range matched {
		out.Transactions = append(out.Transactions, m.fee)
	}

	for m := range out.Monthly {
		out.Monthly[m] = domain.Round2(out.Monthly[m])
	}
	out.Total = domain.Round2(out.Total)
	out.Resultat = domain.Round2(totalCA - totalCharges)
	out.ResultatAjuste = domain.Round2(out.Resultat + out.Total)
	return out, nil
}
