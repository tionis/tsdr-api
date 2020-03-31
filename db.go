package main

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/go-redis/redis/v7"
)

var redclient *redis.Client

func dbInit() {
	// Init postgres
	/*if os.Getenv("DATABASE_URL") == "" || os.Getenv("REDIS_URL") == "" {
		log.Fatal("[Fatal] Error getting Database Information!")
	}
	postgresString1 := strings.Split(strings.TrimPrefix(os.Getenv("DATABASE_URL"), "postgres://"), "@")
	postgresString2 := strings.Split(postgresString1[0], ":")
	postgresString3 := strings.Split(postgresString1[1], ":")
	postgresString4 := strings.Split(postgresString3[1], "/")
	host := postgresString3[0]
	port, err := strconv.Atoi(postgresString4[0])
	if err != nil {
		log.Fatal("Could not read Postgres Port")
	}
	user := postgresString2[0]
	password := postgresString2[1]
	dbname := postgresString4[1]
	psqlInfo = fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	*/
	redisS1 := strings.Split(strings.TrimPrefix(os.Getenv("REDIS_URL"), "redis://"), "@")
	redisS2 := strings.Split(redisS1[0], ":")
	redclient = redis.NewClient(&redis.Options{
		Addr:     redisS1[1],
		Password: redisS2[1],
		DB:       0, // use default DB
	})
	if _, err := redclient.Ping().Result(); err != nil {
		log.Println("[FATAL] - Error connecting to redis database! err: ", err)
	}
	// Check the database
	// If empty build it to spec and fill it with relevant data
	// (Maybe Check Data for Integrity?)
}

func set(key string, value string) error {
	return redclient.Set(key, value, 0).Err()
}

func delete(key string) error {
	return redclient.Set(key, "", 0).Err()
}

func setTimer(key string, value string, duration time.Duration) error {
	return redclient.Set(key, value, duration).Err()
}

func get(key string) string {
	return redclient.Get(key).Val()
}

func getResult(key string) (string, error) {
	return redclient.Get(key).Result()
}
