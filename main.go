package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	"github.com/nilsboi/Chirpy/internal/database"
)

type apiConfig struct {
	fileserverHits int
	jwt            string
}

type returnError struct {
	// the key will be the name of struct field unless you give it an explicit JSON tag
	Error string `json:"error"`
}

type returnValid struct {
	// the key will be the name of struct field unless you give it an explicit JSON tag
	Cleaned_body string `json:"cleaned_body"`
}

type returnToken struct {
	// the key will be the name of struct field unless you give it an explicit JSON tag
	Token string `json:"token"`
}

type parameters struct {
	// these tags indicate how the keys in the JSON should be mapped to the struct fields
	// the struct fields must be exported (start with a capital letter) if you want them parsed
	Body             string `json:"body"`
	Email            string `json:"email"`
	Password         string `json:"password"`
	ExpiresInSeconds int    `json:"expires_in_seconds,omitempty"`
	Event string `json:"event"`
	Data data `json:"data"`
}

type data struct {
	User int `json:"user_id"`
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

func checkWords(msg string) string {

	msgSlice := strings.Split(msg, " ")

	for i := 0; i < len(msgSlice); i++ {
		if strings.ToLower(msgSlice[i]) == "kerfuffle" ||
			strings.ToLower(msgSlice[i]) == "sharbert" ||
			strings.ToLower(msgSlice[i]) == "fornax" {
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

	godotenv.Load()
	jwtSecret := os.Getenv("JWT_SECRET")
	polkaSecret := os.Getenv("POLKA_SECRET")

	apiCfg := &apiConfig{jwt: jwtSecret}
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
		s := r.URL.Query().Get("author_id")
		sort := r.URL.Query().Get("sort")


		db, err := database.NewDB("database.json")

		if err != nil {
			respondWithError(w, 400, "Fehler beim Erstellen der DB: "+err.Error())
			return
		}

		chirps, err := db.GetChirps(s, sort)
		if err != nil {
			respondWithError(w, 400, "Fehler beim Abrufen der Chirps: "+err.Error())
			return
		}

		respondWithJSON(w, 200, chirps)

	})

	mux.HandleFunc("GET /api/chirps/{id}", func(w http.ResponseWriter, r *http.Request) {

		db, err := database.NewDB("database.json")

		if err != nil {
			respondWithError(w, 400, "Fehler beim Erstellen der DB: "+err.Error())
			return
		}

		chirp, err := db.GetChirp(r.PathValue("id"))

		if err != nil {
			respondWithError(w, 404, "Fehler beim Abrufen der Chirps: "+err.Error())
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

		tokenEx := r.Header.Get("Authorization")
		tokenString := strings.Split(tokenEx, " ")[1]

		db, err := database.NewDB("database.json")

		if err != nil {
			respondWithError(w, 400, "Fehler beim Erstellen der DB: "+err.Error())
			return
		}

		body := checkWords(params.Body)
		chirp, err := db.CreateChirp(body, tokenString)

		if err != nil {
			respondWithError(w, 400, "Fehler beim Erstellen des Chrip: "+err.Error())
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
			respondWithError(w, 400, "Fehler beim Erstellen der DB: "+err.Error())
			return
		}

		log.Print(params.Email)
		user, err := db.CreateUser(params.Email, params.Password)

		if err != nil {
			respondWithError(w, 400, "Fehler beim Erstellen des User: "+err.Error())
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
			respondWithError(w, 400, "Fehler beim Erstellen der DB: "+err.Error())
			return
		}

		user, err := db.Login(params.Email, params.Password, apiCfg.jwt)

		if err != nil {
			respondWithError(w, 401, "Fehler beim Erstellen des User: "+err.Error())
			return
		}

		respondWithJSON(w, 200, user)

	})

	mux.HandleFunc("PUT /api/users", func(w http.ResponseWriter, r *http.Request) {

		decoder := json.NewDecoder(r.Body)
		params := parameters{}

		err := decoder.Decode(&params)

		if err != nil {
			respondWithError(w, 400, "Something went wrong")
			return
		}

		db, err := database.NewDB("database.json")

		if err != nil {
			respondWithError(w, 400, "Fehler beim Erstellen der DB: "+err.Error())
			return
		}

		tokenEx := r.Header.Get("Authorization")
		tokenString := strings.Split(tokenEx, " ")[1]

		claim, err2 := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(apiCfg.jwt), nil
		})

		if err2 != nil {
			respondWithError(w, 401, "Unauthorized: "+err2.Error())
			return
		}

		id, err4 := claim.Claims.GetSubject()

		if err4 != nil {
			respondWithError(w, 401, "Unauthorized: "+err4.Error())
		}

		idCast, _ := strconv.Atoi(id)

		user, err := db.UpdateUser(params.Email, params.Password, idCast)

		if err != nil {
			respondWithError(w, 401, "Fehler beim Erstellen des User: "+err.Error())
			return
		}

		respondWithJSON(w, 200, user)

	})

	mux.HandleFunc("POST /api/refresh", func(w http.ResponseWriter, r *http.Request) {

		decoder := json.NewDecoder(r.Body)
		params := parameters{}

		err := decoder.Decode(&params)

		if err != io.EOF {
			w.WriteHeader(401)
			return
		}

		db, err := database.NewDB("database.json")

		if err != nil {
			respondWithError(w, 400, "Fehler beim Erstellen der DB-Verbindung: "+err.Error())
			return
		}

		tokenEx := r.Header.Get("Authorization")
		tokenString := strings.Split(tokenEx, " ")[1]

		newToken, err := db.RefreshToken(tokenString, apiCfg.jwt)

		if err != nil {
			respondWithError(w, 401, "Unauthorized: "+err.Error())
			return
		}

		respondWithJSON(w, 200, returnToken{Token: newToken})
	})

	mux.HandleFunc("POST /api/revoke", func(w http.ResponseWriter, r *http.Request) {

		db, err := database.NewDB("database.json")

		if err != nil {
			respondWithError(w, 400, "Fehler beim Erstellen der DB-Verbindung: "+err.Error())
			return
		}

		tokenEx := r.Header.Get("Authorization")
		tokenString := strings.Split(tokenEx, " ")[1]

		success, err := db.RevokeToken(tokenString)

		if err != nil {
			respondWithError(w, 400, "Fehler "+err.Error())
			return
		}

		if success {
			w.WriteHeader(204)
		} else {
			w.WriteHeader(401)
		}
	})

	mux.HandleFunc("DELETE /api/chirps/{ID}", func(w http.ResponseWriter, r *http.Request) {
		db, err := database.NewDB("database.json")

		if err != nil {
			respondWithError(w, 400, "Fehler beim Erstellen der DB-Verbindung: "+err.Error())
			return
		}

		tokenEx := r.Header.Get("Authorization")
		tokenString := strings.Split(tokenEx, " ")[1]

		success, err := db.DeleteChirp(tokenString)

		if err != nil {
			respondWithError(w, 400, "Fehler "+err.Error())
			return
		}

		if success {
			w.WriteHeader(204)
		} else {
			w.WriteHeader(403)
		}

	})

	mux.HandleFunc("POST /api/polka/webhooks", func(w http.ResponseWriter, r *http.Request) {
		tokenEx := r.Header.Get("Authorization")

		if tokenEx == "" {
			w.WriteHeader(401)
			return
		}

		tokenString := strings.Split(tokenEx, " ")[1]
		
		if tokenString != polkaSecret {
			log.Print(tokenString)
			log.Print(polkaSecret)
			w.WriteHeader(401)
			return
		}

		decoder := json.NewDecoder(r.Body)
		params := parameters{}

		err := decoder.Decode(&params)

		if err != nil {
			w.WriteHeader(401)
			return
		}

		if params.Event != "user.upgraded" {
			w.WriteHeader(204)
			return
		} 
			
		db, err := database.NewDB("database.json")

		if err != nil {
			respondWithError(w, 400, "Fehler beim Erstellen der DB-Verbindung: "+err.Error())
			return
		}

		succ, err := db.UpdatePremium(params.Data.User)

		if err != nil {
			respondWithError(w, 404, "Fehler "+err.Error())
			return
		}

		if succ {
			w.WriteHeader(204)
		} else {
			w.WriteHeader(404)
		}

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
