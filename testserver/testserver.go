package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	r := chi.NewRouter()

	r.Use(middleware.Logger)

	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "helloooooo")
	})

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ok")
	})

	log.Println("started")

	http.ListenAndServe(os.Args[1], r)
}
