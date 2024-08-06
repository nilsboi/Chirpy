package main

import (
	"encoding/json"
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

	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
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

	mux.HandleFunc("GET /admin/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		html := fmt.Sprintf("<html><body><h1>Welcome, Chirpy Admin</h1><p>Chirpy has been visited %d times!</p></body></html>", apiCfg.fileserverHits)
		w.Write([]byte(html))
	})

	mux.HandleFunc("POST /api/validate_chirp", func(w http.ResponseWriter, r *http.Request) {
		type parameters struct {
			// these tags indicate how the keys in the JSON should be mapped to the struct fields
			// the struct fields must be exported (start with a capital letter) if you want them parsed
			Body string `json:"body"`
		}

		type returnError struct {
			// the key will be the name of struct field unless you give it an explicit JSON tag
			Error string `json:"error"`
		}

		type returnValid struct {
			// the key will be the name of struct field unless you give it an explicit JSON tag
			Valid bool `json:"valid"`
		}

		w.Header().Set("Content-Type", "application/json")

		decoder := json.NewDecoder(r.Body)
		params := parameters{}

		err := decoder.Decode(&params)
		
		if err != nil {
			respError := returnError{
				Error: "Something went wrong",
			}

			dat, err := json.Marshal(respError)
			if err != nil {
				w.WriteHeader(400)
				return
			}

			w.WriteHeader(400)
			w.Write(dat)
			return
		}

		if len(params.Body) > 140 {
			respError := returnError{
				Error: "Chirp is too long",
			}

			dat, err := json.Marshal(respError)
			if err != nil {
				w.WriteHeader(400)
				return
			}

			w.WriteHeader(400)
			w.Write(dat)
			return
		}

		respVal := returnValid{
			Valid: true,
		}

		dat, err := json.Marshal(respVal)
		if err != nil {
			w.WriteHeader(400)
			return
		}

		w.WriteHeader(200)
		w.Write(dat)

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
