package service

// Objectives: the 5-year plan parsed from the DAVAI brief, reconciled with
// the FEC actuals, plus the Attio pipeline estimate refreshed through codex.

import (
	"context"
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"time"

	"biztop/internal/domain"
	"biztop/internal/repository"
)

type Objectives struct {
	FecsDir   string
	DocPath   string
	CachePath string
}

type Actual struct {
	CA       float64 `json:"ca"`
	Charges  float64 `json:"charges"`
	Resultat float64 `json:"resultat"`
}

type ObjectivesResult struct {
	Objectives []domain.Objective `json:"objectives"`
	Actuals    map[string]*Actual `json:"actuals"`
	Estimate   json.RawMessage    `json:"estimate"`
}

func (o Objectives) Overview() (*ObjectivesResult, error) {
	entries, err := repository.LoadEntries(o.FecsDir)
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
		Objectives: ParseObjectives(repository.ReadObjectivesDoc(o.DocPath)),
		Actuals:    actuals,
		Estimate:   repository.LoadEstimate(o.CachePath),
	}, nil
}

// Refresh queries Attio through codex and caches the estimate.
func (o Objectives) Refresh(ctx context.Context) (json.RawMessage, error) {
	estimate, err := repository.FetchAttioEstimate(ctx)
	if err != nil {
		return nil, err
	}
	estimate["fetched_at"] = time.Now().Format(time.RFC3339)
	return repository.SaveEstimate(o.CachePath, estimate)
}

var (
	// "650 – 800k€" or "900k – 1.4M€" or "4.5 – 6.0M€+"
	rangeRe  = regexp.MustCompile(`([\d.]+)\s*([kM]?)\s*[\x{2013}\x{2014}-]\s*([\d.]+)\s*([kM])\x{20ac}\+?`)
	marginRe = regexp.MustCompile(`([\d.]+)\s*[\x{2013}\x{2014}-]\s*([\d.]+)%`)
	teamRe   = regexp.MustCompile(`^\s*(\d+(?:\s*[\x{2013}\x{2014}-]\s*\d+)?(?:\s*max)?)`)
	yearRe   = regexp.MustCompile(`20\d\d`)
)

func toEuros(num, unit, fallbackUnit string) float64 {
	v, _ := strconv.ParseFloat(num, 64)
	if unit == "" {
		unit = fallbackUnit
	}
	switch unit {
	case "k":
		return v * 1e3
	case "M":
		return v * 1e6
	}
	return v
}

// ParseObjectives extracts the 5-year plan table from the brief. The table
// is one long line ("YearRevenue Target..." then "2026650 – 800k€25–30%...");
// rows are located by their consecutive years, then each chunk is parsed.
func ParseObjectives(doc string) []domain.Objective {
	var table string
	for _, line := range strings.Split(doc, "\n") {
		if strings.Contains(line, "Revenue Target") {
			table = line
			break
		}
	}
	if table == "" {
		return nil
	}
	first := yearRe.FindString(table)
	if first == "" {
		return nil
	}
	startYear, _ := strconv.Atoi(first)

	// Chunk boundaries: each row starts at its year, years are consecutive.
	starts := []int{strings.Index(table, first)}
	yearsFound := []int{startYear}
	for y := startYear + 1; ; y++ {
		i := strings.Index(table[starts[len(starts)-1]:], strconv.Itoa(y))
		if i < 0 {
			break
		}
		starts = append(starts, starts[len(starts)-1]+i)
		yearsFound = append(yearsFound, y)
	}

	var objectives []domain.Objective
	for i, start := range starts {
		end := len(table)
		if i+1 < len(starts) {
			end = starts[i+1]
		}
		chunk := table[start+4 : end] // skip the year itself
		ranges := rangeRe.FindAllStringSubmatch(chunk, 2)
		margin := marginRe.FindStringSubmatch(chunk)
		if len(ranges) < 2 || margin == nil {
			continue
		}
		o := domain.Objective{
			Year:       yearsFound[i],
			RevenueMin: toEuros(ranges[0][1], ranges[0][2], ranges[0][4]),
			RevenueMax: toEuros(ranges[0][3], ranges[0][4], ""),
			ProfitMin:  toEuros(ranges[1][1], ranges[1][2], ranges[1][4]),
			ProfitMax:  toEuros(ranges[1][3], ranges[1][4], ""),
		}
		o.MarginMin, _ = strconv.ParseFloat(margin[1], 64)
		o.MarginMax, _ = strconv.ParseFloat(margin[2], 64)
		rangePos := rangeRe.FindAllStringIndex(chunk, 2)
		if team := teamRe.FindStringSubmatch(chunk[rangePos[1][1]:]); team != nil {
			o.Team = strings.TrimSpace(team[1])
		}
		objectives = append(objectives, o)
	}
	return objectives
}
