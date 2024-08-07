package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type apiConfig struct {
	fileserverHits int
}

type returnError struct {
	// the key will be the name of struct field unless you give it an explicit JSON tag
	Error string `json:"error"`
}

type returnValid struct {
	// the key will be the name of struct field unless you give it an explicit JSON tag
	Cleaned_body string `json:"cleaned_body"`
}

type parameters struct {
	// these tags indicate how the keys in the JSON should be mapped to the struct fields
	// the struct fields must be exported (start with a capital letter) if you want them parsed
	Body string `json:"body"`
}



func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits++
		next.ServeHTTP(w, r)
	})
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	
	respError := returnError{
		Error: msg,
	}

	dat, err := json.Marshal(respError)
	
	if err != nil {
		w.WriteHeader(400)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(dat)
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	
	dat, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(400)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(dat)
}

func checkWords (msg string) string {

	msgSlice := strings.Split(msg, " ") 

	for i := 0; i < len(msgSlice); i++ {
		if strings.ToLower(msgSlice[i]) == "kerfuffle" || 
		strings.ToLower(msgSlice[i]) == "sharbert" ||
		strings.ToLower(msgSlice[i])== "fornax" {
			msgSlice[i] = "****"
		}
	}

	msgCheck := strings.Join(msgSlice, " ")
	
	return msgCheck
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
	
		decoder := json.NewDecoder(r.Body)
		params := parameters{}

		err := decoder.Decode(&params)

		if err != nil {
			respondWithError(w, 400, "Something went wrong")
			return
		}

		if len(params.Body) > 140 {
			respondWithError(w, 400, "Chirp is too long")
			return
		}

		msg := checkWords(params.Body)

		respVal := returnValid{
			Cleaned_body: msg,
			}

		respondWithJSON(w, 200, respVal)

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
