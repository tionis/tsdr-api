package data

import (
	"database/sql"
	"errors"
	"os"
	"sync"
	"time"

	_ "github.com/go-kivik/couchdb/v4" // The CouchDB driver
	kivik "github.com/go-kivik/kivik/v4"
	_ "github.com/heroku/x/hmetrics/onload" // Heroku advanced go metrics
	"github.com/keybase/go-logging"
	_ "github.com/lib/pq" // The PostgreSQL Driver
)

// DB represents an postgres DB object
// TODO: to be removed
var db *sql.DB

var couch *kivik.Client

var dataLog = logging.MustGetLogger("data")

var tmpDataLock sync.RWMutex
var tmpData map[string]map[string]tmpDataObject

type tmpDataObject struct {
	data       string
	validUntil time.Time
}

// DBInit initializes the DB connection and tests it
func DBInit() {
	// Init RAM Store
	tmpData = make(map[string]map[string]tmpDataObject)

	// Init postgres
	if os.Getenv("DATABASE_URL") == "" {
		dataLog.Info("Database: " + os.Getenv("DATABASE_URL"))
		dataLog.Fatal("Fatal Error getting Database Information!")
	}
	var err error
	db, err = sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		dataLog.Fatal("PostgreSQL Server Connection failed: ", err)
	}
	db.SetMaxOpenConns(19) // Heroku free plan limit - 1 debug connection
	err = db.Ping()
	if err != nil {
		dataLog.Fatal("PostgreSQL Server Ping failed: ", err)
		err = db.Close()
		if err != nil {
			dataLog.Warning("PostgreSQL Error closing Postgres Session")
		}
		return
	}

	// Init Couchdb
	if os.Getenv("COUCHDB_URL") == "" {
		dataLog.Info("Database: " + os.Getenv("COUCHDB_URL"))
		dataLog.Fatal("Fatal Error getting CouchDB Database Information!")
	}
	couch, err = kivik.New("couch", "https://localhost:5984/")
	if err != nil {
		panic(err)
	}

	// start go routine that cleans cache hourly
	go startCacheCleaner(time.Hour)

	// Init the Database
	// Quotator Database
	_, err = db.Query(`CREATE TABLE IF NOT EXISTS quotes(id SERIAL PRIMARY KEY, quote text, author text, language text, universe text)`)
	if err != nil {
		dataLog.Fatal("Error creating table quotes: ", err)
	}
}

// startCacheCleaner cleans the cache and then waits the specified time until it cleans the cache again
func startCacheCleaner(waitingTime time.Duration) {
	for {
		cleanCache()
		time.Sleep(waitingTime)
	}
}

// cleanCache cleans the whole cache by iterating over it and deleting stale values
func cleanCache() {
	tmpDataLock.Lock()
	defer tmpDataLock.Unlock()
	for _, bucket := range tmpData {
		for key, data := range bucket {
			if !data.isValid() {
				delete(bucket, key)
			}
		}
	}
}

// isValid returns true if the tmpDataObject is still within its validity time range
func (t tmpDataObject) isValid() bool {
	return t.validUntil.After(time.Now())
}

// SetTmp Sets an temporary in memory key value store value
func SetTmp(bucket string, key string, value string, duration time.Duration) {
	var dataToSave tmpDataObject
	dataToSave.data = value
	dataToSave.validUntil = time.Now().Add(duration)
	tmpDataLock.Lock()
	defer tmpDataLock.Unlock()
	if tmpData[bucket] == nil {
		tmpData[bucket] = make(map[string]tmpDataObject)
	}
	tmpData[bucket][key] = dataToSave
}

// GetTmp gets an temporary in memory key value store value
func GetTmp(bucket string, key string) string {
	tmpDataLock.Lock()
	defer tmpDataLock.Unlock()
	if tmpData[bucket] == nil {
		return ""
	}
	dataToLoad := tmpData[bucket][key]
	if !dataToLoad.isValid() {
		delete(tmpData[bucket], key)
		return ""
	}
	return dataToLoad.data
}

// DelTmp deletes an temporary in memory key value store value
func DelTmp(bucket string, key string) {
	tmpDataLock.Lock()
	defer tmpDataLock.Unlock()
	if tmpData[bucket] == nil {
		return
	}
	delete(tmpData[bucket], key)
}

// GetUserIDFromDiscordID returns a userID assigned to a given discord ID
func GetUserIDFromDiscordID(discordUserID string) (string, error) {
	// TODO
	return "@tionis:tasadar.net", nil
}

func GetUserIDFromTelegramID(telegramUserID string) (string, error) {
	// TODO
	return "@tionis:tasadar.net", nil
}

// SetUserData sets the key in the bucket in the data of a user to the data from value
func SetUserData(userID, bucket, key string, value interface{}) error {
	// TODO
	return errors.New("not implemented")
}

// GetUserData gets the key in the bucket in the data of a user
func GetUserData(userID, bucket, key string) (interface{}, error) {
	// TODO
	return nil, errors.New("not implemented")
}

/*
package persist

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"sync"
)

var lock sync.Mutex

// Marshal is a function that marshals the object into an
// io.Reader.
// By default, it uses the JSON marshaller.
var Marshal = func(v interface{}) (io.Reader, error) {
	b, err := json.MarshalIndent(v, "", "\t")
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(b), nil
}

// Unmarshal is a function that unmarshals the data from the
// reader into the specified value.
// By default, it uses the JSON unmarshaller.
var Unmarshal = func(r io.Reader, v interface{}) error {
	return json.NewDecoder(r).Decode(v)
}

// Save saves a representation of v to the file at path.
func Save(path string, v interface{}) error {
	lock.Lock()
	defer lock.Unlock()
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	r, err := Marshal(v)
	if err != nil {
		return err
	}
	_, err = io.Copy(f, r)
	return err
}

// Load loads the file at path into v.
// Use os.IsNotExist() to see if the returned error is due
// to the file being missing.
func Load(path string, v interface{}) error {
	lock.Lock()
	defer lock.Unlock()
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return Unmarshal(f, v)
}*/
