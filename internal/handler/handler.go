// Package handler wires the HTTP routes to the services and serves the gui.
package handler

import (
	"context"
	"encoding/json"
	"io/fs"
	"net/http"
	"strconv"
	"time"

	"biztop/internal/gui"
	"biztop/internal/service"
)

const codexTimeout = 10 * time.Minute

type Server struct {
	Compta     service.Compta
	Fees       service.Fees
	Objectives service.Objectives
	Customers  service.Customers
}

func New(s Server) (*http.ServeMux, error) {
	static, err := fs.Sub(gui.FS, ".")
	if err != nil {
		return nil, err
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.index)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(static))))
	mux.HandleFunc("/api/data", s.data)
	mux.HandleFunc("/api/transactions", s.transactions)
	mux.HandleFunc("/api/fees", s.fees)
	mux.HandleFunc("/api/customers", s.customers)
	mux.HandleFunc("/api/objectives", s.objectives)
	mux.HandleFunc("/api/objectives/refresh", s.objectivesRefresh)
	return mux, nil
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.Encode(v)
}

func fail(w http.ResponseWriter, err error) {
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func intParam(r *http.Request, name string) int {
	v, _ := strconv.Atoi(r.URL.Query().Get(name))
	return v
}

func (s Server) index(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	page, err := gui.FS.ReadFile("index.html")
	if err != nil {
		fail(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(page)
}

func (s Server) data(w http.ResponseWriter, r *http.Request) {
	out, err := s.Compta.Data(intParam(r, "year"))
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, out)
}

func (s Server) transactions(w http.ResponseWriter, r *http.Request) {
	year := intParam(r, "year")
	if year == 0 {
		http.Error(w, "year is required", http.StatusBadRequest)
		return
	}
	out, err := s.Compta.Transactions(year, r.URL.Query().Get("compte"), intParam(r, "month"))
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, out)
}

func (s Server) fees(w http.ResponseWriter, r *http.Request) {
	out, err := s.Fees.Compute(intParam(r, "year"))
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, out)
}

func (s Server) customers(w http.ResponseWriter, r *http.Request) {
	out, err := s.Customers.Compute()
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, out)
}

func (s Server) objectives(w http.ResponseWriter, r *http.Request) {
	out, err := s.Objectives.Overview()
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, out)
}

func (s Server) objectivesRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), codexTimeout)
	defer cancel()
	estimate, err := s.Objectives.Refresh(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, map[string]any{"estimate": estimate})
}
