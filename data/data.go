package data

import (
	"database/sql"
	"errors"
	"os"
	"time"

	_ "github.com/go-kivik/couchdb/v4" // The CouchDB driver
	kivik "github.com/go-kivik/kivik/v4"
	"github.com/keybase/go-logging"
	_ "github.com/lib/pq" // The PostgreSQL Driver
)

// DB represents an postgres DB object
// TODO: to be removed
var DB *sql.DB

var couch *kivik.Client

var dataLog = logging.MustGetLogger("data")

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
	DB, err = sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		dataLog.Fatal("PostgreSQL Server Connection failed: ", err)
	}
	DB.SetMaxOpenConns(19) // Heroku free plan limit - 1 debug connection
	err = DB.Ping()
	if err != nil {
		dataLog.Fatal("PostgreSQL Server Ping failed: ", err)
		err = DB.Close()
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

	// Init the Database
	// Quotator Database
	_, err = DB.Query(`CREATE TABLE IF NOT EXISTS quotes(id SERIAL PRIMARY KEY, quote text, author text, language text, universe text)`)
	if err != nil {
		dataLog.Fatal("Error creating table quotes: ", err)
	}
}

// SetTmp Sets an temporary in memory key value store value
func SetTmp(bucket string, key string, value string, duration time.Duration) {
	var dataToSave tmpDataObject
	dataToSave.data = value
	dataToSave.validUntil = time.Now().Add(duration)
	if tmpData[bucket] == nil {
		tmpData[bucket] = make(map[string]tmpDataObject)
	}
	tmpData[bucket][key] = dataToSave
	// TODO init job to delete old values
}

// GetTmp gets an temporary in memory key value store value
func GetTmp(bucket string, key string) string {
	if tmpData[bucket] == nil {
		return ""
	}
	dataToLoad := tmpData[bucket][key]
	if dataToLoad.validUntil.Before(time.Now()) {
		delete(tmpData[bucket], key)
		return ""
	}
	return dataToLoad.data
}

// DelTmp deletes an temporary in memory key value store value
func DelTmp(bucket string, key string) {
	if tmpData[bucket] == nil {
		return
	}
	delete(tmpData[bucket], key)
}

// GetUserIDFromDiscordID returns a userID assigned to a given discord ID
func GetUserIDFromDiscordID(discordUserID string) (string, error) {
	return "tionis", nil
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
