package service

// Management fees: expenses that are legally company charges but are really
// part of the manager's compensation, so they are added back to the result
// to see the true business profit. Libelle patterns count the whole expense;
// compte rules carry a ratio so only that portion counts.

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
	ratioFor, err := feeRatioFor(cfg, f.RulesPath)
	if err != nil {
		return nil, err
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

// feeRatioFor compiles the management-fees rules into a matcher returning
// the portion of an expense counted as fees (0..1). Excludes veto
// everything; a libelle pattern counts the expense in full, otherwise the
// compte ratio applies.
func feeRatioFor(cfg domain.FeesConfig, rulesPath string) (func(domain.Entry) float64, error) {
	var patterns []*regexp.Regexp
	for _, p := range cfg.LibellePatterns {
		re, err := regexp.Compile("(?i)" + p)
		if err != nil {
			return nil, fmt.Errorf("invalid pattern %q in %s: %w", p, rulesPath, err)
		}
		patterns = append(patterns, re)
	}
	var excludes []*regexp.Regexp
	for _, p := range cfg.ExcludePatterns {
		re, err := regexp.Compile("(?i)" + p)
		if err != nil {
			return nil, fmt.Errorf("invalid exclude pattern %q in %s: %w", p, rulesPath, err)
		}
		excludes = append(excludes, re)
	}
	compteRatio := map[string]float64{}
	for _, c := range cfg.Comptes {
		compteRatio[c.Compte] = c.Ratio
	}
	return func(e domain.Entry) float64 {
		for _, re := range excludes {
			if re.MatchString(e.Libelle) {
				return 0
			}
		}
		for _, re := range patterns {
			if re.MatchString(e.Libelle) {
				return 1
			}
		}
		return compteRatio[e.Compte]
	}, nil
}
