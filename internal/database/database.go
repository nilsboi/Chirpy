package database

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type DB struct {
	path string
	mux  *sync.RWMutex
}

type DBStructure struct {
	Chirps map[int]Chirp    `json:"chirps"`
	Users  map[int]User     `json:"users"`
	Tokens map[string]Token `json:"tokens"`
}

type Chirp struct {
	ID     int    `json:"id"`
	Body   string `json:"body"`
	Author int    `json:"author_id"`
}

type Token struct {
	TokenString string    `json:"tokenString"`
	Expires     time.Time `json:"expires"`
	UserID      int       `json:"user"`
}

type User struct {
	ID           int     `json:"id"`
	Email        string  `json:"email"`
	Password     *string `json:"password,omitempty"`
	Token        string `json:"token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Premium bool `json:"is_chirpy_red"`
}

// CreateChirp creates a new chirp and saves it to disk
func (db *DB) CreateChirp(body string, token string) (Chirp, error) {

	users, err := db.GetUsers()

	if err != nil {
		log.Printf("Error fetching users in database: %v", err)
		return Chirp{}, err
	}

	for _, user := range users {
		
		if user.Token == token {
			chirps, err1 := db.GetChirps("", "")

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

			id := max + 1

			chirp := Chirp{
				ID:   id,
				Body: body,
				Author: user.ID,
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
	}
	return Chirp{}, errors.New("unauthorized")
}

// GetChirps returns all chirps in the database
func (db *DB) GetChirps(id string, s string) ([]Chirp, error) {
	db.mux.RLock()
	defer db.mux.RUnlock()
	dbStructure, err := db.loadDB()

	if err != nil {
		log.Printf("Error fetching chirps in GetChirps: %v", err)
		return nil, err
	}

	chirps := []Chirp{}

	if id == "" {

		for _, chirp := range dbStructure.Chirps {
			chirps = append(chirps, chirp)
		}
	} else {

		i, err := strconv.Atoi(id)

		if err != nil {
			log.Printf("Casting int: %v", err)
			return nil, err
		}

		for _, chirp := range dbStructure.Chirps {
			if i == chirp.Author {
				chirps = append(chirps, chirp)
			}
		}
	}

	if s== "desc" {
		sort.Slice(chirps, func(i, j int) bool { return chirps[i].ID < chirps[j].ID })
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

func (db *DB) DeleteChirp(tokenString string) (bool, error) {

	users, err := db.GetUsers()

	if err != nil {
		log.Printf("Error loading tokens: %v", err)
		return false, err
	}
	
	for _, user := range(users) {
		if user.Token == tokenString {
			
			dbStructure, err := db.loadDB()

			if err != nil {
				log.Printf("Error loading db: %v", err)
				return false, err
			}
		
			if dbStructure.Chirps[len(dbStructure.Chirps)].Author == user.ID { 
			delete(dbStructure.Chirps , len(dbStructure.Chirps) + 1)

			db.writeDB(dbStructure)
			return true, nil
			} else {
				return false, nil
			}
		}
	}

	return false, nil 
}

// ensureDB creates a new database file if it doesn't exist
func (db *DB) ensureDB() error {
	db.mux.Lock()
	defer db.mux.Unlock()
	err := os.WriteFile(db.path, []byte(`{ "chirps": {}, "users": {}, "tokens": {} }`), 0666)

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
		mux:  &sync.RWMutex{},
	}

	_, err := os.ReadFile(path)

	if errors.Is(err, os.ErrNotExist) {
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

	id := max + 1

	password, err4 := HashPassword(password)

	if err4 != nil {
		log.Printf("Error hashing password: %v", err4)
		return User{}, err4
	}

	user := User{
		ID:       id,
		Email:    email,
		Password: &password,
		Premium: false,
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

func (db *DB) GetTokens() ([]Token, error) {
	db.mux.RLock()
	defer db.mux.RUnlock()

	dbStructure, err := db.loadDB()

	if err != nil {
		log.Printf("Error fetching users in GetUsers: %v", err)
		return nil, err
	}

	tokens := make([]Token, 0, len(dbStructure.Tokens))

	for _, token := range dbStructure.Tokens {
		tokens = append(tokens, token)
	}

	return tokens, nil

}

func (db *DB) Login(email string, password string, key string) (User, error) {

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

			claims := &jwt.RegisteredClaims{
				IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
				ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(time.Second * time.Duration(3600))), //TODO und Expires richtig implementiren
				Issuer:    "chirpy",
				Subject:   strconv.Itoa(user.ID),
			}

			mySigningKey := []byte(key)

			token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
			ss, err := token.SignedString(mySigningKey)
			if err != nil {
				log.Print("Error signing token")
				return User{}, errors.New("Problem with Token")
			}

			dbStructure, err2 := db.loadDB()

			if err2 != nil {
				log.Print("Error Loadingdb")
				return User{}, errors.New("Problem with loading DB")
			}

			newToken := Token{}
			bytes := make([]byte, 32) // 256 bits
			_, err5 := rand.Read(bytes)
			if err5 != nil {
				return User{}, err5
			}

			newToken.TokenString = hex.EncodeToString(bytes)
			newToken.Expires = time.Now().Add(time.Hour * time.Duration(1440))
			newToken.UserID = user.ID

			dbStructure.Tokens[hex.EncodeToString(bytes)] = newToken

			user.Token = ss
			user.RefreshToken = newToken.TokenString
			
			dbStructure.Users[user.ID] = user

			err3 := db.writeDB(dbStructure)

			if err3 != nil {
				log.Printf("Error writing database file: %v", err3)
				return User{}, err3
			}

			user.Token = ss
			user.Password = nil
			user.RefreshToken = newToken.TokenString

			return user, nil

		}
	}

	return User{}, errors.New("User not found")
}

func (db *DB) UpdateUser(email string, password string, id int) (User, error) {

	users, err1 := db.GetUsers()

	if err1 != nil {
		log.Printf("Error fetching users in database: %v", err1)
		return User{}, err1
	}

	for _, user := range users {
		if user.ID == id {

			dbStructure, err2 := db.loadDB()

			if err2 != nil {
				log.Printf("Error reading database file: %v", err2)
				return User{}, err2
			}

			password, err4 := HashPassword(password)

			if err4 != nil {
				log.Printf("Error hashing password: %v", err4)
				return User{}, err4
			}

			user.Email = email
			user.Password = &password
			dbStructure.Users[id] = user

			err3 := db.writeDB(dbStructure)

			if err3 != nil {
				log.Printf("Error writing database file: %v", err3)
				return User{}, err3
			}

			user.Password = nil
			return user, nil

		}
	}
	return User{}, errors.New("problem with updating credentials")
}

func (db *DB) RefreshToken(refreshToken string, key string) (string, error) {
	tokens, err := db.GetTokens()
	if err != nil {
		log.Printf("Error fetching tokens: %v", err)
		return "", err
	}

	for _, token := range tokens {
		if token.TokenString == refreshToken {

			if time.Now().UTC().After(token.Expires) {
				return "", errors.New("expired")
			}

			claims := &jwt.RegisteredClaims{
				IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
				ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(time.Second * time.Duration(3600))),
				Issuer:    "chirpy",
				Subject:   strconv.Itoa(token.UserID),
			}

			mySigningKey := []byte(key)

			tokenA := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
			ss, err := tokenA.SignedString(mySigningKey)
			if err != nil {
				log.Print("Error signing token")
				return "", errors.New("Problem with access token")
			}

			dbStructure, err := db.loadDB()

			if err != nil {
				log.Printf("Error fetching DB: %v", err)
				return "", err
			}

			user := dbStructure.Users[token.UserID]

			user.Token = ss

			dbStructure.Users[token.UserID] = user

			err2 := db.writeDB(dbStructure)

			if err2 != nil {
				log.Print("Error saving access token")
				return "", errors.New("error saving access token")
			}

			return ss, nil
		}

	}
	return "", errors.New("invalid refresh token")
}

func (db *DB) RevokeToken(token string) (bool, error) {

	dbStructure, err := db.loadDB()

	if err != nil {
		log.Print("Error saving access token")
		return false, errors.New("error revoking token")
	}

	val, ok := dbStructure.Tokens[token]

	if ok {
		val.Expires = time.Now().Add(time.Hour * time.Duration(1440))
		dbStructure.Tokens[token] = val

		err2 := db.writeDB(dbStructure)

		if err2 != nil {
			log.Print("Error saving access token")
			return false, errors.New("error revoking token")
		}
		return true, nil
	}

	return false, nil
}

func (db *DB) UpdatePremium(user int) (bool, error) {


	dbStructure, err := db.loadDB()

	if err != nil {
		log.Print("Error saving access token")
		return false, errors.New("error revoking token")
	}

	val, ok := dbStructure.Users[user]

	if ok {
		val.Premium = true
		dbStructure.Users[user] = val

		err2 := db.writeDB(dbStructure)

		if err2 != nil {
			log.Print("Error saving user")
			return false, errors.New("error upgrading")
		}
		return true, nil
	}

	return false, nil 
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
