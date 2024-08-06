package main

import (
	"fmt"
	"net/http"
	"strconv"
)

type apiConfig struct {
	fileserverHits int
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits++
		next.ServeHTTP(w, r)
	})
}


func main() {

	apiCfg := &apiConfig{}
	mux := http.NewServeMux()

	fileServerHandler := http.FileServer(http.Dir("."))

	mux.Handle("/app/", apiCfg.middlewareMetricsInc(
		http.StripPrefix("/app", fileServerHandler),
	))

	mux.Handle("/assets", http.FileServer(http.Dir("./assets")))

	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter,r *http.Request) {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
	})


	mux.HandleFunc("GET /api/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hits: " + strconv.Itoa(apiCfg.fileserverHits)))
	})

	mux.HandleFunc("/api/reset", func(w http.ResponseWriter, r *http.Request) {
		apiCfg.fileserverHits = 0
		w.WriteHeader(http.StatusOK)
	})


	server := &http.Server{
    Addr:    "localhost:8080",
    Handler: mux,
	}

	fmt.Printf("Server wird versucht zu starten... http://localhost:8080")
	err := server.ListenAndServe()	

	if err != nil {
		fmt.Printf("Server konnte nicht gestartet werden: %v", err)
	} 

}