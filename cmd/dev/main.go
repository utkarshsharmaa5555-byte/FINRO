// Local dev server: loads .env.local, optionally migrates+seeds, serves the API on :8080.
package main

import (
	"bufio"
	"context"
	"log"
	"net/http"
	"os"
	"strings"

	"finro/app"
)

func main() {
	if f, err := os.Open(".env.local"); err == nil {
		s := bufio.NewScanner(f)
		for s.Scan() {
			if k, v, ok := strings.Cut(s.Text(), "="); ok && !strings.HasPrefix(k, "#") && os.Getenv(k) == "" {
				os.Setenv(strings.TrimSpace(k), strings.Trim(strings.TrimSpace(v), `"`))
			}
		}
		f.Close()
	}
	if len(os.Args) > 1 && os.Args[1] == "setup" {
		if err := app.Migrate(context.Background()); err != nil {
			log.Fatal(err)
		}
		if err := app.Seed(context.Background()); err != nil {
			log.Fatal(err)
		}
		log.Println("migrated + seeded")
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "reset-demo" {
		if err := app.ResetDemo(context.Background()); err != nil {
			log.Fatal(err)
		}
		log.Println("demo personas reset")
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "refresh" {
		log.Printf("%v", app.RefreshAll(context.Background()))
		return
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8787"
	}
	log.Println("API on :" + port)
	log.Fatal(http.ListenAndServe(":"+port, app.Router()))
}
