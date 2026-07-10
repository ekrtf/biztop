package repository

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"biztop/internal/domain"
)

func testTypes() []domain.AttioType {
	return []domain.AttioType{
		{Name: "Projects", Billing: "one-shot", Description: "custom projects"},
		{Name: "Maintenance", Billing: "mrr", Description: "recurring work"},
	}
}

func TestBuildPromptIncludesTypesAndBillingRules(t *testing.T) {
	prompt := buildPrompt(testTypes())
	for _, want := range []string{"Attio CRM", `"Projects"`, "one-shot billing", `"Maintenance"`, "monthly recurring revenue", "by_type"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt does not contain %q:\n%s", want, prompt)
		}
	}
}

func TestBuildSchema(t *testing.T) {
	raw, err := buildSchema(testTypes())
	if err != nil {
		t.Fatal(err)
	}
	var schema map[string]any
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatal(err)
	}
	if schema["type"] != "object" || schema["additionalProperties"] != false {
		t.Fatalf("unexpected top-level schema: %v", schema)
	}
	props := schema["properties"].(map[string]any)
	byType := props["by_type"].(map[string]any)
	required := byType["required"].([]any)
	if len(required) != 2 || required[0] != "Projects" || required[1] != "Maintenance" {
		t.Fatalf("required by_type names = %v", required)
	}
}

func TestFetchAttioEstimateRequiresTypes(t *testing.T) {
	if _, err := FetchAttioEstimate(context.Background(), nil); err == nil {
		t.Fatal("FetchAttioEstimate() error = nil, want missing types error")
	}
}

func TestFetchAttioEstimateUsesCodexOutput(t *testing.T) {
	dir := t.TempDir()
	installFakeCodex(t, dir, `{"deals":[],"by_type":{"Projects":100,"Maintenance":25}}`)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	estimate, err := FetchAttioEstimate(context.Background(), testTypes())
	if err != nil {
		t.Fatal(err)
	}
	byType := estimate["by_type"].(map[string]any)
	if byType["Projects"] != float64(100) || byType["Maintenance"] != float64(25) {
		t.Fatalf("unexpected estimate: %+v", estimate)
	}
}

func TestFetchAttioEstimateReportsCodexFailure(t *testing.T) {
	dir := t.TempDir()
	script := "#!/bin/sh\nprintf '%*s' 2101 '' | tr ' ' x >&2\nprintf 'failure log' >&2\nexit 7\n"
	if err := os.WriteFile(filepath.Join(dir, "codex"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	_, err := FetchAttioEstimate(context.Background(), testTypes())
	if err == nil || !strings.Contains(err.Error(), "failure log") {
		t.Fatalf("FetchAttioEstimate() error = %v, want codex log", err)
	}
}

func TestFetchAttioEstimateReportsMissingOutput(t *testing.T) {
	dir := t.TempDir()
	script := "#!/bin/sh\nexit 0\n"
	if err := os.WriteFile(filepath.Join(dir, "codex"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	_, err := FetchAttioEstimate(context.Background(), testTypes())
	if err == nil || !strings.Contains(err.Error(), "produced no output") {
		t.Fatalf("FetchAttioEstimate() error = %v, want missing output", err)
	}
}

func TestFetchAttioEstimateReportsInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	installFakeCodex(t, dir, `{bad json`)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	_, err := FetchAttioEstimate(context.Background(), testTypes())
	if err == nil || !strings.Contains(err.Error(), "not valid JSON") {
		t.Fatalf("FetchAttioEstimate() error = %v, want invalid JSON", err)
	}
}

func installFakeCodex(t *testing.T, dir string, output string) {
	t.Helper()
	script := `#!/bin/sh
out=""
while [ "$#" -gt 0 ]; do
  if [ "$1" = "-o" ]; then
    shift
    out="$1"
  fi
  shift
done
printf '%s\n' '` + output + `' > "$out"
`
	if err := os.WriteFile(filepath.Join(dir, "codex"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
}
