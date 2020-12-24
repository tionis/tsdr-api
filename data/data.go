package data

import (
	"database/sql"
	"errors"
	"os"
	"sync"
	"time"

	_ "github.com/heroku/x/hmetrics/onload" // Heroku advanced go metrics
	"github.com/keybase/go-logging"
	_ "github.com/lib/pq" // The PostgreSQL Driver
)

// DB represents an postgres database
var db *sql.DB

var dataLog = logging.MustGetLogger("data")

// This could be a performance bottleneck in the future.
// If the bot performs badly the cache logic should be rewritten.
var tmpDataLock sync.RWMutex
var tmpData map[string]map[string]tmpDataObject

// Define errors
var errUserNotFound = errors.New("user could not be found in the database")

type tmpDataObject struct {
	data       string
	validUntil time.Time
}

// DBInit initializes the DB connection and tests it
func DBInit() {
	// Init RAM Store
	tmpData = make(map[string]map[string]tmpDataObject)

	// Init postgres
	initPostgres()

	// start go routine that cleans cache hourly
	go startCacheCleaner(time.Hour)

	// Init the Database
	initDatabase()
}

func initPostgres() {
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
}

func initDatabase() {
	// Quotator Table
	_, err := db.Query(`CREATE TABLE IF NOT EXISTS quotes(id SERIAL PRIMARY KEY, quote text, author text, language text, universe text, byUser text)`)
	if err != nil {
		dataLog.Fatal("Error creating table quotes: ", err)
	}

	// User Tables
	_, err = db.Query(`CREATE TABLE IF NOT EXISTS users(id text PRIMARY KEY, email text, isAdmin boolean)`)
	if err != nil {
		dataLog.Fatal("Error creating table users: ", err)
	}

	_, err = db.Query(`CREATE TABLE IF NOT EXISTS qotd(userID text PRIMARY KEY, quoteID SERIAL, validUntil timestamptz)`)
	if err != nil {
		dataLog.Fatal("Error creating table qotd: ", err)
	}

	_, err = db.Query(`CREATE TABLE IF NOT EXISTS userdata(userID text PRIMARY KEY, key text, value text)`)
	if err != nil {
		dataLog.Fatal("Error creating table userdata: ", err)
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
