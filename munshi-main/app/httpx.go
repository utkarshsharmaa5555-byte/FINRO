package app

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
)

type apiError struct {
	Status int
	Msg    string
}

func (e *apiError) Error() string { return e.Msg }

func httpErr(status int, msg string) error { return &apiError{status, msg} }

func handle(h func(w http.ResponseWriter, r *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := h(w, r)
		if err == nil {
			return
		}
		var ae *apiError
		if errors.As(err, &ae) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(ae.Status)
			json.NewEncoder(w).Encode(map[string]string{"error": ae.Msg})
			return
		}
		log.Printf("%s %s: %v", r.Method, r.URL.Path, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(500)
		json.NewEncoder(w).Encode(map[string]string{"error": "something went wrong, please try again"})
	}
}

func writeJSON(w http.ResponseWriter, v any) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(v)
}

func readJSON(r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, 6<<20)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return httpErr(400, "invalid request body")
	}
	return nil
}
