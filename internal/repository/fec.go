// Package repository is the data access layer: FEC files, config files,
// the cached Attio estimate and the codex CLI.
package repository

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"biztop/internal/domain"
)

// LoadEntries reads every fecs/*.csv (";" separated, dd/mm/yyyy dates).
func LoadEntries(dir string) ([]domain.Entry, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var rows []domain.Entry
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".csv") {
			continue
		}
		f, err := os.Open(filepath.Join(dir, file.Name()))
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
			rows = append(rows, domain.Entry{
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

func parseAmount(s string) float64 {
	s = strings.TrimSpace(strings.ReplaceAll(s, ",", "."))
	if s == "" {
		return 0
	}
	v, _ := strconv.ParseFloat(s, 64)
	return v
}
