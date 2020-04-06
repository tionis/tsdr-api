package main

import (
	"log"
	"os"
	"strings"

	"github.com/go-redis/redis/v7"
)

var redclient *redis.Client

func dbInit() {
	// Check if REDIS_URL is defined
	if os.Getenv("REDIS_URL") == "" {
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
}

// Direct Database Interaction Funtions
func sadd(key, value string) error {
	return redclient.SAdd(key, value).Err()
}

func sismember(key, value string) (bool, error) {
	return redclient.SIsMember(key, value).Result()
}

func srem(key, value string) error {
	return redclient.SRem(key, value).Err()
}

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
