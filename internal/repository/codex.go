package repository

// Attio CRM access goes through the `codex` CLI, which has the Attio
// plugin enabled and handles authentication. The prompt and the output
// schema are built from the attio_types rules (rules.yml).

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"biztop/internal/domain"
)

func buildPrompt(types []domain.AttioType) string {
	var b strings.Builder
	b.WriteString(`Use the Attio CRM tools to review every deal / opportunity in my CRM.
Keep only deals that are still open (not won and billed, not lost) and estimate the upcoming revenue for Davai.
For each deal give: name, amount_eur, the month the revenue is expected (YYYY-MM), a probability between 0 and 1,
and classify it into exactly one of these Davai revenue types:
`)
	for _, t := range types {
		if t.Billing == "mrr" {
			fmt.Fprintf(&b, "- %q (%s): recurring billing, amount_eur is the monthly recurring revenue, parse it from the Attio MRR field.\n", t.Name, t.Description)
		} else {
			fmt.Fprintf(&b, "- %q (%s): one-shot billing, amount_eur is the deal value still expected to be invoiced.\n", t.Name, t.Description)
		}
	}
	b.WriteString(`When a needed amount is missing in Attio, estimate it from the deal context.
by_type must contain the sum of amount_eur per type (0 when none).`)
	return b.String()
}

func buildSchema(types []domain.AttioType) ([]byte, error) {
	names := make([]string, len(types))
	byType := map[string]any{}
	for i, t := range types {
		names[i] = t.Name
		byType[t.Name] = map[string]any{"type": "number"}
	}
	schema := map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"deals", "by_type"},
		"properties": map[string]any{
			"deals": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type":                 "object",
					"additionalProperties": false,
					"required":             []string{"name", "type", "amount_eur", "expected_month", "probability", "notes"},
					"properties": map[string]any{
						"name":           map[string]any{"type": "string"},
						"type":           map[string]any{"type": "string", "enum": names},
						"amount_eur":     map[string]any{"type": "number"},
						"expected_month": map[string]any{"type": "string"},
						"probability":    map[string]any{"type": "number"},
						"notes":          map[string]any{"type": "string"},
					},
				},
			},
			"by_type": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             names,
				"properties":           byType,
			},
		},
	}
	return json.Marshal(schema)
}

// FetchAttioEstimate asks codex to estimate the upcoming revenue across all
// open Attio deals, classified by the configured Davai types.
func FetchAttioEstimate(ctx context.Context, types []domain.AttioType) (map[string]any, error) {
	if len(types) == 0 {
		return nil, fmt.Errorf("no attio_types configured in rules.yml")
	}
	schema, err := buildSchema(types)
	if err != nil {
		return nil, err
	}
	tmp, err := os.MkdirTemp("", "biztop-codex")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)
	schemaPath := filepath.Join(tmp, "schema.json")
	outPath := filepath.Join(tmp, "estimate.json")
	if err := os.WriteFile(schemaPath, schema, 0o644); err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, "codex", "exec",
		"--color", "never",
		"-s", "read-only",
		"--output-schema", schemaPath,
		"-o", outPath,
		buildPrompt(types))
	logs, err := cmd.CombinedOutput()
	if err != nil {
		tail := string(logs)
		if len(tail) > 2000 {
			tail = tail[len(tail)-2000:]
		}
		return nil, fmt.Errorf("codex failed: %w\n%s", err, tail)
	}
	raw, err := os.ReadFile(outPath)
	if err != nil {
		return nil, fmt.Errorf("codex produced no output file: %w", err)
	}
	var estimate map[string]any
	if err := json.Unmarshal(raw, &estimate); err != nil {
		return nil, fmt.Errorf("codex output is not valid JSON: %w", err)
	}
	return estimate, nil
}
