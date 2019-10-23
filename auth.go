//Userauth Code
// Sync user.auth.php from s3 and check against it,
// if unavailable check against environment variable
package main

import (
	"log"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

func auth(user string, pass string) (bool, error) {
	val, err := redclient.Get(user).Result()
	if err != nil {
		return false, err
	}
	return val == pass, err
}

func authGin(c *gin.Context) {
	// Returns Result off authentication
	buf := make([]byte, 1024)
	num, _ := c.Request.Body.Read(buf)
	params, err := url.ParseQuery(string(buf[0:num]))
	if err != nil {
		log.Println("[TasadarAuth] ", c.Error(err))
		return
	}
	user := strings.Join(params["User"], " ")
	pass := strings.Join(params["Password"], " ")

	if b, err := auth(user, pass); b && err == nil {
		c.String(200, "Access Granted")
	} else if err == nil {
		c.String(403, "Access Denied")
	} else {
		c.String(500, "Internal Server Error - Check the logs")
		log.Println("[TasadarAuth] Error in authGin: ", err)
	}
}

func updateAuth() {
	// read file
	// update the key value store
	/*err := client.Set("key", "value", 0).Err()
	if err != nil {
		panic(err)
	}

	val, err := client.Get("key").Result()
	if err != nil {
		panic(err)
	}
	fmt.Println("key", val)

	val2, err := client.Get("key2").Result()
	if err == redis.Nil {
		fmt.Println("key2 does not exist")
	} else if err != nil {
		panic(err)
	} else {
		fmt.Println("key2", val2)
	}*/
}
