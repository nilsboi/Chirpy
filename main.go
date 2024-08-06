package main

import (
	"fmt"
	"net/http"
)


func main() {
	mux := http.NewServeMux()

	mux.Handle("/", http.FileServer(http.Dir(".")))
	mux.Handle("/assets", http.FileServer(http.Dir("./assets")))


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