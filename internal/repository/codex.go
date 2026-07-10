package repository

// Attio CRM access goes through the `codex` CLI, which has the Attio
// plugin enabled and handles authentication.

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const codexPrompt = `Use the Attio CRM tools to review every deal / opportunity in my CRM.
Keep only deals that are still open (not won and billed, not lost) and estimate the upcoming revenue for Davai.
For each deal give: name, the amount in EUR still expected to be invoiced, the month it is expected (YYYY-MM),
a probability between 0 and 1, and classify it into exactly one Davai revenue type:
"Projects" (custom software projects), "Maintenance & Hosting" (recurring maintenance, hosting, support contracts),
"Prezence" (the Prezence product) or "Bodacker" (the Bodacker product).
Base amounts on the deal values recorded in Attio when present, otherwise estimate from the deal context.
by_type must contain the sum of amount_eur per type (0 when none).`

const estimateSchema = `{
  "type": "object",
  "properties": {
    "deals": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "name": {"type": "string"},
          "type": {"type": "string", "enum": ["Projects", "Maintenance & Hosting", "Prezence", "Bodacker"]},
          "amount_eur": {"type": "number"},
          "expected_month": {"type": "string"},
          "probability": {"type": "number"},
          "notes": {"type": "string"}
        },
        "required": ["name", "type", "amount_eur", "expected_month", "probability", "notes"],
        "additionalProperties": false
      }
    },
    "by_type": {
      "type": "object",
      "properties": {
        "Projects": {"type": "number"},
        "Maintenance & Hosting": {"type": "number"},
        "Prezence": {"type": "number"},
        "Bodacker": {"type": "number"}
      },
      "required": ["Projects", "Maintenance & Hosting", "Prezence", "Bodacker"],
      "additionalProperties": false
    }
  },
  "required": ["deals", "by_type"],
  "additionalProperties": false
}`

// FetchAttioEstimate asks codex to estimate the upcoming revenue across all
// open Attio deals, structured by the schema above.
func FetchAttioEstimate(ctx context.Context) (map[string]any, error) {
	tmp, err := os.MkdirTemp("", "biztop-codex")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)
	schemaPath := filepath.Join(tmp, "schema.json")
	outPath := filepath.Join(tmp, "estimate.json")
	if err := os.WriteFile(schemaPath, []byte(estimateSchema), 0o644); err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, "codex", "exec",
		"--color", "never",
		"-s", "read-only",
		"--output-schema", schemaPath,
		"-o", outPath,
		codexPrompt)
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
