package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/nilsboi/Chirpy/internal/database"
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
	Email string `json:"email"`
	Password string `json:"password"`
}


func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits++
		next.ServeHTTP(w, r)
	})
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	respError := returnError{
		Error: msg,
	}

	dat, err := json.Marshal(respError)
	
	if err != nil {
		w.WriteHeader(400)
		return
	}

	
	w.WriteHeader(code)
	w.Write(dat)
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")

	dat, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(400)
		return
	}

	
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

	dbg := flag.Bool("debug", false, "Enable debug mode")
	flag.Parse()

	if *dbg {
		log.Print("Debugging enabled")
		e := os.Remove("database.json") 
    if e != nil { 
				log.Print(e)
    } 

	}

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

	mux.HandleFunc("GET /api/chirps", func(w http.ResponseWriter, r *http.Request) {

		db, err := database.NewDB("database.json")

		if err != nil {
			respondWithError(w, 400, "Fehler beim Erstellen der DB: " + err.Error())
			return
		}

		chirps, err := db.GetChirps()
    if err != nil {
        respondWithError(w, 400, "Fehler beim Abrufen der Chirps: " + err.Error())
        return
    }

    respondWithJSON(w, 200, chirps)

	})

	mux.HandleFunc("GET /api/chirps/{id}", func(w http.ResponseWriter, r *http.Request) {

		db, err := database.NewDB("database.json")

		if err != nil {
			respondWithError(w, 400, "Fehler beim Erstellen der DB: " + err.Error())
			return
		}

		chirp, err := db.GetChirp(r.PathValue("id"))

    if err != nil {
        respondWithError(w, 404, "Fehler beim Abrufen der Chirps: " + err.Error())
        return
    }

    respondWithJSON(w, 200, chirp)

	})

	

	mux.HandleFunc("POST /api/chirps", func(w http.ResponseWriter, r *http.Request) {
	
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

	
		db, err := database.NewDB("database.json")

		if err != nil {
			respondWithError(w, 400, "Fehler beim Erstellen der DB: " + err.Error())
			return
		}

		body := checkWords(params.Body)
		chirp, err := db.CreateChirp(body)

		if err != nil {
			respondWithError(w, 400, "Fehler beim Erstellen des Chrip: " + err.Error())
			return
		}

		respondWithJSON(w, 201, chirp)

	})


	mux.HandleFunc("POST /api/users", func(w http.ResponseWriter, r *http.Request) {
	
		decoder := json.NewDecoder(r.Body)
		params := parameters{}

		err := decoder.Decode(&params)

		if err != nil {
			respondWithError(w, 400, "Something went wrong")
			return
		}

		db, err := database.NewDB("database.json")

		if err != nil {
			respondWithError(w, 400, "Fehler beim Erstellen der DB: " + err.Error())
			return
		}

		log.Print(params.Email)
		user, err := db.CreateUser(params.Email, params.Password)

		if err != nil {
			respondWithError(w, 400, "Fehler beim Erstellen des User: " + err.Error())
			return
		}

		respondWithJSON(w, 201, user)

	})

	mux.HandleFunc("POST /api/login", func(w http.ResponseWriter, r *http.Request) {
	
		decoder := json.NewDecoder(r.Body)
		params := parameters{}

		err := decoder.Decode(&params)

		if err != nil {
			respondWithError(w, 400, "Something went wrong")
			return
		}

		db, err := database.NewDB("database.json")

		if err != nil {
			respondWithError(w, 400, "Fehler beim Erstellen der DB: " + err.Error())
			return
		}

		user, err := db.Login(params.Email, params.Password)

		if err != nil {
			respondWithError(w, 401, "Fehler beim Erstellen des User: " + err.Error())
			return
		}

		respondWithJSON(w, 200, user)

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
