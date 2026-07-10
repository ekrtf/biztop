package main

import (
	"errors"
	"net/http"
	"testing"
)

func TestNewMux(t *testing.T) {
	mux, err := newMux()
	if err != nil {
		t.Fatal(err)
	}
	if mux == nil {
		t.Fatal("newMux() returned nil mux")
	}
	req, err := http.NewRequest(http.MethodGet, "http://localhost"+addr+"/", nil)
	if err != nil {
		t.Fatal(err)
	}
	if req.URL.Host != "localhost"+addr {
		t.Fatalf("unexpected addr: %s", req.URL.Host)
	}
}

func TestRun(t *testing.T) {
	wantErr := errors.New("stop")
	err := run(func(gotAddr string, handler http.Handler) error {
		if gotAddr != addr {
			t.Fatalf("addr = %q, want %q", gotAddr, addr)
		}
		if handler == nil {
			t.Fatal("handler is nil")
		}
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("run() error = %v, want %v", err, wantErr)
	}
}
