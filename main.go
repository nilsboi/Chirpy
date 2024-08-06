package main

import (
	"fmt"
	"net/http"
)


func main() {
	mux := http.NewServeMux()

	mux.Handle("/app/", http.StripPrefix("/app", http.FileServer(http.Dir("."))))
	mux.Handle("/assets", http.FileServer(http.Dir("./assets")))

	mux.HandleFunc("/healthz", func(w http.ResponseWriter,r *http.Request) {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
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