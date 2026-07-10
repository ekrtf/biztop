package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"biztop/internal/service"
)

func TestRoutes(t *testing.T) {
	server := newTestServer(t)
	mux, err := New(server)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name   string
		method string
		path   string
		status int
		body   string
	}{
		{name: "index", method: http.MethodGet, path: "/", status: http.StatusOK, body: "<html"},
		{name: "not found", method: http.MethodGet, path: "/missing", status: http.StatusNotFound, body: "404"},
		{name: "static", method: http.MethodGet, path: "/static/style.css", status: http.StatusOK, body: "body"},
		{name: "data", method: http.MethodGet, path: "/api/data", status: http.StatusOK, body: `"year":2025`},
		{name: "transactions require year", method: http.MethodGet, path: "/api/transactions", status: http.StatusBadRequest, body: "year is required"},
		{name: "transactions", method: http.MethodGet, path: "/api/transactions?year=2025&compte=613600&month=1", status: http.StatusOK, body: `"total_debit":100`},
		{name: "fees", method: http.MethodGet, path: "/api/fees?year=2025", status: http.StatusOK, body: `"resultat_ajuste":870`},
		{name: "objectives", method: http.MethodGet, path: "/api/objectives", status: http.StatusOK, body: `"actuals"`},
		{name: "mission", method: http.MethodGet, path: "/api/mission", status: http.StatusOK, body: `"reste"`},
		{name: "refresh wrong method", method: http.MethodGet, path: "/api/objectives/refresh", status: http.StatusMethodNotAllowed, body: "POST only"},
		{name: "refresh", method: http.MethodPost, path: "/api/objectives/refresh", status: http.StatusOK, body: `"estimate"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			if rec.Code != tt.status {
				t.Fatalf("%s %s status = %d, want %d; body=%s", tt.method, tt.path, rec.Code, tt.status, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), tt.body) {
				t.Fatalf("%s %s body does not contain %q:\n%s", tt.method, tt.path, tt.body, rec.Body.String())
			}
		})
	}
}

func TestRouteErrors(t *testing.T) {
	rulesPath := writeHandlerRules(t)
	mux, err := New(Server{
		Compta:     service.Compta{FecsDir: filepath.Join(t.TempDir(), "missing")},
		Fees:       service.Fees{FecsDir: filepath.Join(t.TempDir(), "missing"), RulesPath: rulesPath},
		Objectives: service.Objectives{FecsDir: filepath.Join(t.TempDir(), "missing"), RulesPath: rulesPath},
		Mission:    service.Mission{FecsDir: filepath.Join(t.TempDir(), "missing"), RulesPath: rulesPath},
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{"/api/data", "/api/transactions?year=2025", "/api/fees", "/api/objectives", "/api/mission"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("%s status = %d, want 500", path, rec.Code)
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/api/objectives/refresh", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("refresh status = %d, want 502", rec.Code)
	}
}

func TestWriteJSONDoesNotEscapeHTML(t *testing.T) {
	rec := httptest.NewRecorder()
	writeJSON(rec, map[string]string{"html": "<b>"})
	if rec.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("Content-Type = %q", rec.Header().Get("Content-Type"))
	}
	var out map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out["html"] != "<b>" || strings.Contains(rec.Body.String(), "\\u003c") {
		t.Fatalf("writeJSON escaped HTML unexpectedly: %s", rec.Body.String())
	}
}

func newTestServer(t *testing.T) Server {
	t.Helper()
	dir := t.TempDir()
	installHandlerFakeCodex(t, dir, `{"deals":[],"by_type":{"Projects":5}}`)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	fecsDir := filepath.Join(dir, "fecs")
	if err := os.Mkdir(fecsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	fec := "Date;Libelle;Compte;Libelle du compte;Debit;Credit\n" +
		"01/01/2025;Invoice;706000;Prestations;0;1000\n" +
		"02/01/2025;Software;613600;Logiciels;100;0\n" +
		"03/01/2025;Lunch;625700;Restaurants;100;0\n"
	if err := os.WriteFile(filepath.Join(fecsDir, "2025.csv"), []byte(fec), 0o644); err != nil {
		t.Fatal(err)
	}
	rulesPath := filepath.Join(dir, "rules.yml")
	if err := os.WriteFile(rulesPath, []byte(`management_fees:
  comptes:
    - compte: "625700"
      ratio: 0.7
attio_types:
  - name: Projects
    billing: one-shot
    description: custom projects
`), 0o644); err != nil {
		t.Fatal(err)
	}
	docPath := filepath.Join(dir, "objectives.md")
	if err := os.WriteFile(docPath, []byte("YearRevenue TargetNet Profit MarginApprox. Net ProfitTeam Size2026650 – 800k€25–30%160 – 240k€6–7"), 0o644); err != nil {
		t.Fatal(err)
	}
	return Server{
		Compta:     service.Compta{FecsDir: fecsDir},
		Fees:       service.Fees{FecsDir: fecsDir, RulesPath: rulesPath},
		Objectives: service.Objectives{FecsDir: fecsDir, DocPath: docPath, CachePath: filepath.Join(dir, "estimate.json"), RulesPath: rulesPath},
		Mission:    service.Mission{FecsDir: fecsDir, DocPath: docPath, CachePath: filepath.Join(dir, "estimate.json"), RulesPath: rulesPath},
	}
}

func writeHandlerRules(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "rules.yml")
	if err := os.WriteFile(path, []byte("management_fees: {}\nattio_types: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func installHandlerFakeCodex(t *testing.T, dir string, output string) {
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
