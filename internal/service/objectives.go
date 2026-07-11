package service

// Objectives: the 5-year plan objectives from rules.yml, reconciled with
// the FEC actuals, plus the Attio pipeline estimate refreshed through codex.

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"biztop/internal/domain"
	"biztop/internal/repository"
)

type Objectives struct {
	FecsDir   string
	CachePath string
	RulesPath string
}

type Actual struct {
	CA       float64 `json:"ca"`
	Charges  float64 `json:"charges"`
	Resultat float64 `json:"resultat"`
}

type ObjectivesResult struct {
	Objectives []domain.Objective `json:"objectives"`
	Actuals    map[string]*Actual `json:"actuals"`
	Types      []domain.AttioType `json:"types"`
	Estimate   json.RawMessage    `json:"estimate"`
}

func (o Objectives) Overview() (*ObjectivesResult, error) {
	entries, err := repository.LoadEntries(o.FecsDir)
	if err != nil {
		return nil, err
	}
	rules, err := repository.LoadRules(o.RulesPath)
	if err != nil {
		return nil, err
	}
	actuals := map[string]*Actual{}
	for _, e := range entries {
		key := strconv.Itoa(e.Year)
		if actuals[key] == nil {
			actuals[key] = &Actual{}
		}
		if e.IsIncome() {
			actuals[key].CA += e.IncomeAmount()
		} else if e.IsExpense() {
			actuals[key].Charges += e.ExpenseAmount()
		}
	}
	for _, a := range actuals {
		a.CA = domain.Round2(a.CA)
		a.Charges = domain.Round2(a.Charges)
		a.Resultat = domain.Round2(a.CA - a.Charges)
	}
	return &ObjectivesResult{
		Objectives: rules.Objectives,
		Actuals:    actuals,
		Types:      rules.AttioTypes,
		Estimate:   repository.LoadEstimate(o.CachePath),
	}, nil
}

// Refresh queries Attio through codex and caches the estimate.
func (o Objectives) Refresh(ctx context.Context) (json.RawMessage, error) {
	rules, err := repository.LoadRules(o.RulesPath)
	if err != nil {
		return nil, err
	}
	estimate, err := repository.FetchAttioEstimate(ctx, rules.AttioTypes)
	if err != nil {
		return nil, err
	}
	estimate["fetched_at"] = time.Now().Format(time.RFC3339)
	return repository.SaveEstimate(o.CachePath, estimate)
}
