package database

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"strconv"
	"sync"

	"golang.org/x/crypto/bcrypt"
)

type DB struct {
	path string
	mux  *sync.RWMutex
}

type DBStructure struct {
	Chirps map[int]Chirp `json:"chirps"`
	Users map[int]User `json:"users"`
}

type Chirp struct {
	ID   int    `json:"id"`
	Body string `json:"body"`
}


type User struct {
	ID   int    `json:"id"`
	Email string `json:"email"`
	Password *string `json:"password,omitempty"`
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

// Get chrips bei id

func (db *DB) GetChirp(id string) (Chirp, error) {
	db.mux.RLock()
	defer db.mux.RUnlock()
	dbStructure, err := db.loadDB()

	if err != nil {
		log.Printf("Error fetching chirp in GetChir: %v", err)
		return Chirp{}, err
	}

	find, err := strconv.Atoi(id) 
	
	if err != nil {
		log.Printf("Error casting id in GetChir: %v", err)
		return Chirp{}, err
	}

	for _, chirp := range dbStructure.Chirps {
		
   if chirp.ID == find {
		return chirp, nil
	 }
	}

	return Chirp{}, errors.New("ID not found")
}

// ensureDB creates a new database file if it doesn't exist
func (db *DB) ensureDB() error {
	db.mux.Lock()
defer db.mux.Unlock()
	err := os.WriteFile(db.path,[]byte(`{ "chirps": {}, "users": {} }`),0666)

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

func (db *DB) CreateUser(email string, password string) (User, error) {
	users, err1 := db.GetUsers()

	if err1 != nil {
		log.Printf("Error fetching users in database: %v", err1)
		return User{}, err1
	}
	
	max := 0

	for _, user := range users {
    if user.ID > max {
        max = user.ID
    }

		if user.Email == email {
			return User{}, errors.New("User already registered")
		}
	}

	id := max +1

	password, err4 := HashPassword(password)

	if err4 != nil {
		log.Printf("Error hashing password: %v", err4)
		return User{}, err4
	}

	user := User{
		ID: id,
		Email: email,
		Password: &password,
	}

	dbStructure, err2 := db.loadDB()

	if err2 != nil {
		log.Printf("Error reading database file: %v", err2)
		return User{}, err2
	}

	dbStructure.Users[id] = user

	err3 := db.writeDB(dbStructure) 

	if err3 != nil {
		log.Printf("Error writing database file: %v", err3)
		return User{}, err3
	}
	
	user.Password = nil

	return user, nil 
	
}

func (db *DB) GetUsers() ([]User, error) {
	db.mux.RLock()
	defer db.mux.RUnlock()
	dbStructure, err := db.loadDB()

	if err != nil {
		log.Printf("Error fetching users in GetUsers: %v", err)
		return nil, err
	}

	users := make([]User, 0, len(dbStructure.Chirps))

	for _, user := range dbStructure.Users {
    users = append(users, user)
	}

	return users, nil
}

func (db *DB) Login(email string, password string) (User, error) {
	
	users, err1 := db.GetUsers()

	if err1 != nil {
		log.Printf("Error fetching users in database: %v", err1)
		return User{}, err1
	}
	
	for _, user := range users {
    if user.Email == email {
			check := CheckPasswordHash(password, *user.Password)
			
			if !check {
				log.Print("Credentials not valid")
				return User{}, errors.New("Problem with login")
			}

       user.Password = nil
			 return user, nil
			 
    }
	}
	return User{}, errors.New("User not found")
}


func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
