package database

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"sync"
)

type DB struct {
	path string
	mux  *sync.RWMutex
}

type DBStructure struct {
	Chirps map[int]Chirp `json:"chirps"`
}

type Chirp struct {
	ID   int    `json:"id"`
	Body string `json:"body"`
}

// CreateChirp creates a new chirp and saves it to disk
func (db *DB) CreateChirp(body string) (Chirp, error) {
	

	chirps, err1 := db.GetChirps()

	if err1 != nil {
		log.Printf("Error fetching chirps in database: %v", err1)
		return Chirp{}, err1
	}
	max := 0

	for _, chirp := range chirps {
    if chirp.ID > max {
        max = chirp.ID
    }
	}

	id := max +1

	chirp := Chirp{
		ID: id,
		Body: body,
	}

	dbStructure, err2 := db.loadDB()

	if err2 != nil {
		log.Printf("Error reading database file: %v", err2)
		return Chirp{}, err2
	}

	dbStructure.Chirps[id] = chirp

	err3 := db.writeDB(dbStructure) 

	if err3 != nil {
		log.Printf("Error writing database file: %v", err3)
		return Chirp{}, err3
	}
	

	return chirp, nil 
}

// GetChirps returns all chirps in the database
func (db *DB) GetChirps() ([]Chirp, error) {
	db.mux.RLock()
defer db.mux.RUnlock()
	dbStructure, err := db.loadDB()

	if err != nil {
		log.Printf("Error fetching chirps in GetChirps: %v", err)
		return nil, err
	}

	chirps := make([]Chirp, 0, len(dbStructure.Chirps))

	for _, chirp := range dbStructure.Chirps {
    chirps = append(chirps, chirp)
	}

	return chirps, nil
}

// ensureDB creates a new database file if it doesn't exist
func (db *DB) ensureDB() error {
	db.mux.Lock()
defer db.mux.Unlock()
	err := os.WriteFile(db.path,[]byte(`{ "chirps": {} }`),0666)

		if err != nil {
			return err
		}

	return nil 
}

// loadDB reads the database file into memory
func (db *DB) loadDB() (DBStructure, error) {
	db.mux.RLock()
defer db.mux.RUnlock()

	data, err1 := os.ReadFile(db.path)
	
	if err1 != nil {
		log.Printf("Error reading File in loadDB: %v", err1)
		return DBStructure{}, err1
	}

	var dbStructure DBStructure
	err2 := json.Unmarshal(data, &dbStructure)
	
	if err2 != nil {
			log.Printf("Error unmarshal DB in  loadDB: %v", err2)
			return DBStructure{}, err2
	}

	return dbStructure, nil
}

// writeDB writes the database file to disk
func (db *DB) writeDB(dbStructure DBStructure) error {
	db.mux.Lock()
defer db.mux.Unlock()
	

	data, err := json.Marshal(dbStructure)
	if err != nil {
			return err
	}

	err = os.WriteFile(db.path, data, 0666)
	if err != nil {
			return err
	}

	return nil
}

// NewDB creates a new database connection
// and creates the database file if it doesn't exist
func NewDB(path string) (*DB, error) {

	newDatabase := DB{
		path: path, 
		mux: &sync.RWMutex{}, 
	}
	
	_, err := os.ReadFile(path)

	if errors.Is(err, os.ErrNotExist ) {
		err := newDatabase.ensureDB()
		if err != nil {
			return nil, err
		}

	}

	return &newDatabase, nil
}
