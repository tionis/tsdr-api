//Userauth Code
// Sync user.auth.php from s3 and check against it,
// if unavailable check against environment variable
package main

import (
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
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
	header := c.Request.Header
	basicString := strings.Join(header["Authorization"], "")
	if basicString == "" {
		c.Header("WWW-Authenticate", "Basic")
		c.String(401, "Not authorized")
		return
	}
	basicString = strings.TrimPrefix(basicString, "Basic ")
	basic, err := base64.StdEncoding.DecodeString(basicString)
	if err != nil {
		log.Println("[TasadarAuth] Error in basic Auth: ", err)
	}
	pair := strings.Split(string(basic), ":")
	if authUser(pair[0], pair[1]) {
		c.String(200, "OK")
	} else {
		c.String(401, "Not authorized")
	}
}

func authGinGroup(c *gin.Context) {
	header := c.Request.Header
	basicString := strings.Join(header["Authorization"], "")
	if basicString == "" {
		c.Header("WWW-Authenticate", "Basic")
		c.String(401, "Not authorized")
		return
	}
	basicString = strings.TrimPrefix(basicString, "Basic ")
	basic, err := base64.StdEncoding.DecodeString(basicString)
	if err != nil {
		log.Println("[TasadarAuth] Error in basic Auth: ", err)
	}
	pair := strings.Split(string(basic), ":")
	if authUser(pair[0], pair[1]) {
		val, err := redclient.Get("auth|" + pair[0] + "|groups").Result()
		if err != nil {
			log.Println("[TasadarAuth] Error looking up group of user: ", err)
		}
		if strings.Contains(val, c.Param("group")) {
			c.String(200, "OK")
		} else {
			c.String(401, "Not authorized")
		}
	} else {
		c.String(401, "Not authorized")
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
		var oldlist []string
		oldval, err := redclient.Get("userlist").Result()
		if err != nil {
			log.Println("[TasadarAuth] An error occurred! ", err)
			msgAlpha <- "Error in TasadarAuth!"
		}
		err = json.Unmarshal([]byte(oldval), &oldlist)
		if err != nil {
			log.Println("[TasadarAuth] An error occurred! ", err)
			msgAlpha <- "Error in TasadarAuth!"
		}
		// Check for user deletions
		for _, txt := range oldlist {
			found := false
			currentUser := txt
			for _, txt2 := range userlist {
				if txt == txt2 {
					found = true
					break
				}
			}
			if !found {
				err = redclient.Set("auth|"+currentUser+"|hash", "", 0).Err()
				if err != nil {
					log.Println("[TasadarAuth] Error updating database: ", err)
				}
				err = redclient.Set("auth|"+currentUser+"|groups", "", 0).Err()
				if err != nil {
					log.Println("[TasadarAuth] Error updating database: ", err)
				}
				//msgAlpha <- "New Account Deletion: " + currentUser
			}
		}

		// Put new list into database
		err = redclient.Set("userlist", string(userlistJSON), 0).Err()
		if err != nil {
			log.Println("[TasadarAuth] Error updating database: ", err)
			msgAlpha <- "Error in TasadarAuth!"
		}
	}
	log.Println("[TasadarAuth] Finished Updating the database")
}

func authUser(username, password string) bool {
	val, err := redclient.Get("auth|" + username + "|hash").Result()
	if err != nil {
		return false
	}
	return checkPasswordHash(password, val)
}

func respondWithError(code int, message string, c *gin.Context) {
	resp := map[string]string{"error": message}

	c.JSON(code, resp)
	c.Abort()
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func checkPasswordHash(password, hash string) bool {
	if password == "" || hash == "" {
		return false
	}
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
