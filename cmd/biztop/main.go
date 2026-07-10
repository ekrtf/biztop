// BizTop - Business monitoring dashboard for Davai.
// Config and wiring only; the app lives in internal/.
package main

import (
	"fmt"
	"log"
	"net/http"

	"biztop/internal/handler"
	"biztop/internal/service"
)

func main() {
	if err := run(http.ListenAndServe); err != nil {
		log.Fatal(err)
	}
}

func run(listenAndServe func(string, http.Handler) error) error {
	mux, err := newMux()
	if err != nil {
		return err
	}
	fmt.Println("BizTop demarre -> http://localhost" + addr)
	return listenAndServe(addr, mux)
}

const (
	addr          = ":5055"
	fecsDir       = "fecs"
	rulesFile     = "rules.yml"
	objectivesDoc = "docs/DAVAI_2030.md"
	estimateCache = "attio_estimate.json"
)

func newMux() (*http.ServeMux, error) {
	return handler.New(handler.Server{
		Compta:     service.Compta{FecsDir: fecsDir},
		Fees:       service.Fees{FecsDir: fecsDir, RulesPath: rulesFile},
		Objectives: service.Objectives{FecsDir: fecsDir, DocPath: objectivesDoc, CachePath: estimateCache, RulesPath: rulesFile},
		Customers:  service.Customers{FecsDir: fecsDir, RulesPath: rulesFile},
		Mission:    service.Mission{FecsDir: fecsDir, DocPath: objectivesDoc, CachePath: estimateCache, RulesPath: rulesFile},
	})
}
