package main

import (
	"database/sql"
	"log"
	"os"
	"strings"
	"time"

	"github.com/go-redis/redis/v7"
	_ "github.com/lib/pq"
)

var redclient *redis.Client
var db *sql.DB

func dbInit() {
	// Init postgres
	if os.Getenv("DATABASE_URL") == "" || os.Getenv("REDIS_URL") == "" {
		log.Println("Database: " + os.Getenv("DATABASE_URL") + "  |Redis:  " + os.Getenv("REDIS_URL"))
		log.Fatal("[Tasadar] Fatal Error getting Database Information!")
	}
	redisS1 := strings.Split(strings.TrimPrefix(os.Getenv("REDIS_URL"), "redis://"), "@")
	redisS2 := strings.Split(redisS1[0], ":")
	redclient = redis.NewClient(&redis.Options{
		Addr:     redisS1[1],
		Password: redisS2[1],
		DB:       0, // use default DB
	})
	if _, err := redclient.Ping().Result(); err != nil {
		log.Fatal("[Tasadar] Fatal Error connecting to redis database! err: ", err)
	}
	var err error
	db, err = sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Println("[PostgreSQL] Server Connection failed: ", err)
	}
	db.SetMaxOpenConns(19) // Heroku free plan limit - 1 debug connection
	_ = db.Ping()
	if err != nil {
		log.Println("[PostgreSQL] Server Ping failed: ", err)
		err = db.Close()
		if err != nil {
			log.Println("[PostgreSQL] Error closing Postgres Session")
		}
		return
	}

	// Init the Database
	// Quotator Database
	_, err = db.Query(`CREATE TABLE IF NOT EXISTS quotes(id SERIAL PRIMARY KEY, quote text, author text, language text, universe text)`)
	if err != nil {
		log.Fatal("[Tasadar] Error creating table quotes: ", err)
	}
	//err = db.Close()
	//if err != nil {
	//	log.Println("[Tasadar] Error closing connection to database")
	//}
}

// Direct Database Interaction Functions
func setWithTimer(key, value string, time time.Duration) error {
	return redclient.Set(key, value, time).Err()
}

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

func set(key string, value string) error {
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
}
