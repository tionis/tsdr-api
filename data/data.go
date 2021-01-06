package data

import (
	"database/sql"
	"sync"
	"time"

	_ "github.com/heroku/x/hmetrics/onload" // Heroku advanced go metrics
	"github.com/keybase/go-logging"         // This unifies logging across components of the application
	_ "github.com/lib/pq"                   // The PostgreSQL Driver
)

// GlyphData represents a configured data backend
type GlyphData struct {
	db *sql.DB // DB represents an postgres database

	tmpDataLock *sync.RWMutex                       // This could be a performance bottleneck in the future. If the bot performs badly the cache logic should be rewritten.
	tmpData     map[string]map[string]tmpDataObject // This may not be passed by reference -> bug risk?

	logger *logging.Logger

	adapterMessageChannelsLock *sync.RWMutex
	adapterMessageChannels     map[string]chan AdapterMessage
}

type tmpDataObject struct {
	data       string
	validUntil time.Time
}

// DBInit initializes the DB connection and tests it
func DBInit(sqlURL string) *GlyphData {
	out := &GlyphData{
		db:                         nil,
		tmpDataLock:                &sync.RWMutex{},
		tmpData:                    make(map[string]map[string]tmpDataObject),
		logger:                     logging.MustGetLogger("data"),
		adapterMessageChannelsLock: &sync.RWMutex{},
		adapterMessageChannels:     make(map[string]chan AdapterMessage)}

	// Init postgres
	out.initPostgres(sqlURL)

	// start go routine that cleans cache hourly
	go out.startCacheCleaner(time.Hour)

	// Init the Database
	out.initDatabase()
	return out
}

func (d *GlyphData) initPostgres(sqlURL string) {
	var err error
	d.db, err = sql.Open("postgres", sqlURL)
	if err != nil {
		d.logger.Fatal("PostgreSQL Server Connection failed: ", err)
	}
	d.db.SetMaxOpenConns(19) // Heroku free plan limit - 1 debug connection
	err = d.db.Ping()
	if err != nil {
		d.logger.Fatal("PostgreSQL Server Ping failed: ", err)
		err = d.db.Close()
		if err != nil {
			d.logger.Warning("PostgreSQL Error closing Postgres Session")
		}
		return
	}
}

func (d *GlyphData) initDatabase() {
	// Quotator Table
	_, err := d.db.Query(`CREATE TABLE IF NOT EXISTS quotes(id SERIAL PRIMARY KEY, quote text, author text, language text, universe text, byUser text)`)
	if err != nil {
		d.logger.Fatal("Error creating table quotes: ", err)
	}

	// User Tables
	_, err = d.db.Query(`CREATE TABLE IF NOT EXISTS users(userID text PRIMARY KEY NOT NULL, email text, isAdmin boolean, preferredAdapters json)`)
	if err != nil {
		d.logger.Fatal("Error creating table users: ", err)
	}

	_, err = d.db.Query(`CREATE TABLE IF NOT EXISTS qotd(userID text PRIMARY KEY, quoteID SERIAL, validUntil timestamptz)`)
	if err != nil {
		d.logger.Fatal("Error creating table qotd: ", err)
	}

	// This may not be performance ideal, in the future creating an index may be helpful: CREATE UNIQUE INDEX userID ON userdata(userID);
	_, err = d.db.Query(`CREATE TABLE IF NOT EXISTS userdata(userID text references users (userID) on delete cascade UNIQUE, key text, value text, primary key(userID, key))`)
	if err != nil {
		d.logger.Fatal("Error creating table userdata: ", err)
	}

	_, err = d.db.Query(`CREATE TABLE IF NOT EXISTS authsessions(authToken text PRIMARY KEY, userID text references users (userID) on delete cascade UNIQUE, key text, value text, validUntil timestamptz)`)
	if err != nil {
		d.logger.Fatal("Error creating table userdata: ", err)
	}

	go d.startAuthSessionDBCleaner(time.Hour)
}

// startCacheCleaner cleans the cache and then waits the specified time until it cleans the cache again
func (d *GlyphData) startCacheCleaner(waitingTime time.Duration) {
	for {
		d.cleanCache()
		time.Sleep(waitingTime)
	}
}

// startAuthSessionDBCleaner cleans the authSessionDB and then waits the specified time until it cleans the DB again
func (d *GlyphData) startAuthSessionDBCleaner(waitingTime time.Duration) {
	for {
		d.cleanAuthSessionDB()
		time.Sleep(waitingTime)
	}
}

// cleanCache cleans the whole cache by iterating over it and deleting stale values
func (d *GlyphData) cleanCache() {
	d.tmpDataLock.Lock()
	defer d.tmpDataLock.Unlock()
	for _, bucket := range d.tmpData {
		for key, data := range bucket {
			if !data.isValid() {
				delete(bucket, key)
			}
		}
	}
}

// cleanAuthSessionDB cleans the DB by telling it to delete all values that are stale
func (d *GlyphData) cleanAuthSessionDB() {
	stmt, err := d.db.Prepare(`DELETE FROM authsessions WHERE validUntil < $1`)
	if err != nil {
		d.logger.Error("cleaning authSessionDB failed: ", err)
		return
	}
	_, err = stmt.Exec(time.Now())
	if err != nil {
		d.logger.Error("cleaning authSessionDB failed: ", err)
		return
	}
}

// isValid returns true if the tmpDataObject is still within its validity time range
func (t tmpDataObject) isValid() bool {
	return t.validUntil.After(time.Now())
}

// SetTmp Sets an temporary in memory key value store value
func (d *GlyphData) SetTmp(bucket, key, value string, duration time.Duration) {
	var dataToSave tmpDataObject
	dataToSave.data = value
	dataToSave.validUntil = time.Now().Add(duration)
	d.tmpDataLock.Lock()
	defer d.tmpDataLock.Unlock()
	if d.tmpData[bucket] == nil {
		d.tmpData[bucket] = make(map[string]tmpDataObject)
	}
	d.tmpData[bucket][key] = dataToSave
}

// GetTmp gets an temporary in memory key value store value
func (d *GlyphData) GetTmp(bucket, key string) string {
	d.tmpDataLock.Lock()
	defer d.tmpDataLock.Unlock()
	if d.tmpData[bucket] == nil {
		return ""
	}
	dataToLoad := d.tmpData[bucket][key]
	if !dataToLoad.isValid() {
		delete(d.tmpData[bucket], key)
		return ""
	}
	return dataToLoad.data
}

// DelTmp deletes an temporary in memory key value store value
func (d *GlyphData) DelTmp(bucket, key string) {
	d.tmpDataLock.Lock()
	defer d.tmpDataLock.Unlock()
	if d.tmpData[bucket] == nil {
		return
	}
	delete(d.tmpData[bucket], key)
}
