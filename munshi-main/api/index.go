package handler

import (
	"net/http"

	"finro/app"
)

func Handler(w http.ResponseWriter, r *http.Request) { app.Router().ServeHTTP(w, r) }
