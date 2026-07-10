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
	const (
		addr          = ":5055"
		fecsDir       = "fecs"
		rulesFile     = "rules.yml"
		objectivesDoc = "docs/DAVAI_2030.md"
		estimateCache = "attio_estimate.json"
	)

	mux, err := handler.New(handler.Server{
		Compta:     service.Compta{FecsDir: fecsDir},
		Fees:       service.Fees{FecsDir: fecsDir, RulesPath: rulesFile},
		Objectives: service.Objectives{FecsDir: fecsDir, DocPath: objectivesDoc, CachePath: estimateCache, RulesPath: rulesFile},
		Customers:  service.Customers{FecsDir: fecsDir},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("BizTop demarre -> http://localhost" + addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
