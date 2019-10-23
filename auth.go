//Userauth Code
// Sync user.auth.php from s3 and check against it,
// if unavailable check against environment variable
package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
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
	// Download newest File
	client := &http.Client{}
	req, _ := http.NewRequest("GET", "https://dev.tasadar.net/api/users.auth.php", nil)
	req.SetBasicAuth("glyph-copy-user", "QFT7PEYDX4M76EqmbGRFMnU6sWtbCLbd")
	resp, _ := client.Do(req)
	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		bodyString := string(bodyBytes)
		wholearray := strings.Split(bodyString, "\n")
		var accounts []string
		for _, txt := range wholearray {
			if !strings.Contains(txt, "#") && txt != "" {
				accounts = append(accounts, txt)
			}
		}
		var userlist []string
		for _, txt := range accounts {
			user := strings.Split(txt, ":")
			err := redclient.Set("auth|"+user[0]+"|hash", user[1], 0).Err()
			if err != nil {
				log.Println("[TasadarAuth] Error updating database: ", err)
			}
			err = redclient.Set("auth|"+user[0]+"|name", user[2], 0).Err()
			if err != nil {
				log.Println("[TasadarAuth] Error updating database: ", err)
			}
			err = redclient.Set("auth|"+user[0]+"|email", user[3], 0).Err()
			if err != nil {
				log.Println("[TasadarAuth] Error updating database: ", err)
			}
			err = redclient.Set("auth|"+user[0]+"|groups", user[4], 0).Err()
			if err != nil {
				log.Println("[TasadarAuth] Error updating database: ", err)
			}
			userlist = append(userlist, user[0])
		}
		userlistJSON, _ := json.Marshal(userlist)
		err = redclient.Set("userlist", string(userlistJSON), 0).Err()
		if err != nil {
			log.Println("[TasadarAuth] Error updating database: ", err)
		}
	}
	log.Println("[TasadarAuth] Finished Updating the database")
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
