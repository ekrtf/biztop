package service

// Mission Control: reconciles the three sources of truth - the 5-year plan
// (objectifs), the Attio pipeline and the FEC actuals (compta) - into clear
// "reste a faire" numbers. Pipeline amounts are weighted by probability;
// MRR deals are projected monthly from their expected month to the end of
// their year (rules.yml says which types bill as MRR).

import (
	"encoding/json"
	"strconv"
	"time"

	"biztop/internal/domain"
	"biztop/internal/repository"
)

type Mission struct {
	FecsDir   string
	DocPath   string
	CachePath string
	RulesPath string
	Now       func() time.Time // nil means time.Now, injected in tests
}

// MissionRow reconciles one plan year: objectif - CA realise - pipeline.
type MissionRow struct {
	Year          int     `json:"year"`
	ObjectiveMin  float64 `json:"objective_min"`
	ObjectiveMax  float64 `json:"objective_max"`
	CA            float64 `json:"ca"`           // realise (compta)
	Pipeline      float64 `json:"pipeline"`     // Attio, pondere
	Projection    float64 `json:"projection"`   // ca + pipeline
	ResteCompta   float64 `json:"reste_compta"` // objectif bas - ca
	Reste         float64 `json:"reste"`        // objectif bas - ca - pipeline
	ProfitMin     float64 `json:"profit_min"`
	ProfitMax     float64 `json:"profit_max"`
	Resultat      float64 `json:"resultat"`
	ResteResultat float64 `json:"reste_resultat"` // objectif resultat bas - resultat
}

type MissionResult struct {
	Year            int          `json:"year"`             // current year
	Month           int          `json:"month"`            // current month, 1-12
	MonthsLeft      int          `json:"months_left"`      // remaining months, current one included
	RunRate         float64      `json:"run_rate"`         // reste / months_left, current year
	MonthlyCA       [12]float64  `json:"monthly_ca"`       // realise, current year
	MonthlyPipeline [12]float64  `json:"monthly_pipeline"` // pondere, current year
	Rows            []MissionRow `json:"rows"`             // one per plan year
	HasEstimate     bool         `json:"has_estimate"`
	FetchedAt       string       `json:"fetched_at"`
}

// estimate is the subset of the cached Attio estimate the mission needs.
type estimate struct {
	Deals []struct {
		Type          string  `json:"type"`
		AmountEur     float64 `json:"amount_eur"`
		Probability   float64 `json:"probability"`
		ExpectedMonth string  `json:"expected_month"` // "2026-07"
	} `json:"deals"`
	FetchedAt string `json:"fetched_at"`
}

func (m Mission) Compute() (*MissionResult, error) {
	entries, err := repository.LoadEntries(m.FecsDir)
	if err != nil {
		return nil, err
	}
	rules, err := repository.LoadRules(m.RulesPath)
	if err != nil {
		return nil, err
	}

	now := time.Now
	if m.Now != nil {
		now = m.Now
	}
	out := &MissionResult{Year: now().Year(), Month: int(now().Month())}
	out.MonthsLeft = 12 - out.Month + 1

	caByYear := map[int]float64{}
	resultatByYear := map[int]float64{}
	for _, e := range entries {
		if e.IsIncome() {
			caByYear[e.Year] += e.IncomeAmount()
			resultatByYear[e.Year] += e.IncomeAmount()
			if e.Year == out.Year {
				out.MonthlyCA[e.Month-1] += e.IncomeAmount()
			}
		} else if e.IsExpense() {
			resultatByYear[e.Year] -= e.ExpenseAmount()
		}
	}
	for i := range out.MonthlyCA {
		out.MonthlyCA[i] = domain.Round2(out.MonthlyCA[i])
	}

	pipeByYear := m.pipeline(rules.AttioTypes, out)

	for _, o := range ParseObjectives(repository.ReadObjectivesDoc(m.DocPath)) {
		ca := domain.Round2(caByYear[o.Year])
		pipe := domain.Round2(pipeByYear[o.Year])
		resultat := domain.Round2(resultatByYear[o.Year])
		row := MissionRow{
			Year:          o.Year,
			ObjectiveMin:  o.RevenueMin,
			ObjectiveMax:  o.RevenueMax,
			CA:            ca,
			Pipeline:      pipe,
			Projection:    domain.Round2(ca + pipe),
			ResteCompta:   domain.Round2(o.RevenueMin - ca),
			Reste:         domain.Round2(o.RevenueMin - ca - pipe),
			ProfitMin:     o.ProfitMin,
			ProfitMax:     o.ProfitMax,
			Resultat:      resultat,
			ResteResultat: domain.Round2(o.ProfitMin - resultat),
		}
		if row.Year == out.Year && row.Reste > 0 {
			out.RunRate = domain.Round2(row.Reste / float64(out.MonthsLeft))
		}
		out.Rows = append(out.Rows, row)
	}
	return out, nil
}

// pipeline sums the weighted deals per expected year and fills the current
// year's monthly detail. One-shot deals count once at their expected month;
// MRR deals count every month from the expected one to December.
func (m Mission) pipeline(types []domain.AttioType, out *MissionResult) map[int]float64 {
	var est estimate
	raw := repository.LoadEstimate(m.CachePath)
	if raw == nil || json.Unmarshal(raw, &est) != nil {
		return nil
	}
	out.HasEstimate = true
	out.FetchedAt = est.FetchedAt

	mrr := map[string]bool{}
	for _, t := range types {
		mrr[t.Name] = t.Billing == "mrr"
	}

	byYear := map[int]float64{}
	for _, d := range est.Deals {
		year, month := parseMonth(d.ExpectedMonth)
		if year == 0 {
			continue
		}
		weighted := d.AmountEur * d.Probability
		months := 1
		if mrr[d.Type] {
			months = 12 - month + 1
		}
		byYear[year] += weighted * float64(months)
		if year == out.Year {
			for i := 0; i < months; i++ {
				out.MonthlyPipeline[month-1+i] += weighted
			}
		}
	}
	for i := range out.MonthlyPipeline {
		out.MonthlyPipeline[i] = domain.Round2(out.MonthlyPipeline[i])
	}
	return byYear
}

// parseMonth splits "2026-07"; year 0 means unparseable.
func parseMonth(s string) (year, month int) {
	if len(s) < 7 {
		return 0, 0
	}
	year, _ = strconv.Atoi(s[:4])
	month, _ = strconv.Atoi(s[5:7])
	if year < 2000 || month < 1 || month > 12 {
		return 0, 0
	}
	return year, month
}
