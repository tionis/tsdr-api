package main

import "time"

func dbInit() {
	// Check the database
	// If empty build it to spec and fill it with relevant data
	// (Maybe Check Data for Integrity?)
}

//ToDo
// Methods to Interface with Database here
// Goal: No Database Specific code outside this file!

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
