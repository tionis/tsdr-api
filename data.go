package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"os"
	"time"

	"github.com/keybase/go-logging"
	_ "github.com/lib/pq"
)

//var redclient *redis.Client
var db *sql.DB

var dataLog = logging.MustGetLogger("data")

var tmpData map[string]map[string]tmpDataObject

type tmpDataObject struct {
	data       string
	validUntil time.Time
}

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

func dbInit() {
	// Init RAM Store
	tmpData = make(map[string]map[string]tmpDataObject)

	// Init postgres
	if os.Getenv("DATABASE_URL") == "" || os.Getenv("REDIS_URL") == "" {
		dataLog.Info("Database: " + os.Getenv("DATABASE_URL") + "  |Redis:  " + os.Getenv("REDIS_URL"))
		dataLog.Fatal("Fatal Error getting Database Information!")
	}
	/*redisS1 := strings.Split(strings.TrimPrefix(os.Getenv("REDIS_URL"), "redis://"), "@")
	redisPass := ""
	if redisS1[0] != ":" {
		redisPass = strings.Split(redisS1[0], ":")[1]
	}
	redclient = redis.NewClient(&redis.Options{
		Addr:     redisS1[1],
		Password: redisPass,
		DB:       0, // use default DB
	})
	if _, err := redclient.Ping().Result(); err != nil {
		dataLog.Fatal("Fatal Error connecting to redis database! err: ", err)
	}*/
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

	// Init the Database
	// Quotator Database
	_, err = db.Query(`CREATE TABLE IF NOT EXISTS quotes(id SERIAL PRIMARY KEY, quote text, author text, language text, universe text)`)
	if err != nil {
		dataLog.Fatal("Error creating table quotes: ", err)
	}
}

// TODO This should handle saving arbitrary objects to key value store
// Save an object to the given path
/*func Save(path string, v interface{}) error {
	r, err := Marshal(v)
	if err != nil {
		return err
	}
	var b bytes.Buffer
	_, err = io.Copy(&b, r)
	return redclient.Set(path, b.String(), 0).Err()
}

// Load the object corresponding to a specific path
func Load(path string, v interface{}) error {
	reader := strings.NewReader(redclient.Get(path).Val())
	return Unmarshal(reader, v)
}*/

// Direct Database Interaction Functions
/*func setWithTimer(key, value string, time time.Duration) error {
	return redclient.Set(key, value, time).Err()
}*/

/* SET commands from redis
func setAdd(key, value string) error {
	return redclient.SAdd(key, value).Err()
}

func setIsMember(key, value string) (bool, error) {
	return redclient.SIsMember(key, value).Result()
}

func SetRemove(key, value string) error {
	return redclient.SRem(key, value).Err()
}*/

func setTmp(bucket string, key string, value string, duration time.Duration) {
	var dataToSave tmpDataObject
	dataToSave.data = value
	dataToSave.validUntil = time.Now().Add(duration)
	if tmpData[bucket] == nil {
		tmpData[bucket] = make(map[string]tmpDataObject)
	}
	tmpData[bucket][key] = dataToSave
	// TODO init job to delete old values
}

func getTmp(bucket string, key string) string {
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

func delTmp(bucket string, key string) {
	if tmpData[bucket] == nil {
		return
	}
	delete(tmpData[bucket], key)
}

/*func set(key string, value string) error {
	return redclient.Set(key, value, 0).Err()
}

func del(key string) error {
	return redclient.Del(key).Err()
}

func get(key string) string {
	return redclient.Get(key).Val()
}

func getError(key string) (string, error) {
	return redclient.Get(key).Result()
}*/
