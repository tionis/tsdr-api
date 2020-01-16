package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

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

// Gets the users.auth file from the dev.tasadar.net server and imports it to redis
func updateAuth() {
	// Download newest File
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://dev.tasadar.net/api/users.auth.php", nil)
	if err != nil {
		log.Println("[Fatal] - Error constructiong updateAuth request: ", err)
		return
	}
	req.SetBasicAuth("glyph-copy-user", "QFT7PEYDX4M76EqmbGRFMnU6sWtbCLbd")
	resp, err := client.Do(req)
	if err != nil {
		log.Println("[Fatal] - Error getting user.auth.php file: ", err)
		return
	}
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
			err := redclient.Set("auth|"+user[0]+"|hash", user[1], time.Hour*10).Err()
			if err != nil {
				log.Println("[TasadarAuth] Error updating database: ", err)
			}
			err = redclient.Set("auth|"+user[0]+"|name", user[2], time.Hour*10).Err()
			if err != nil {
				log.Println("[TasadarAuth] Error updating database: ", err)
			}
			err = redclient.Set("auth|"+user[0]+"|email", user[3], time.Hour*10).Err()
			if err != nil {
				log.Println("[TasadarAuth] Error updating database: ", err)
			}
			err = redclient.Set("auth|"+user[0]+"|groups", user[4], time.Hour*10).Err()
			if err != nil {
				log.Println("[TasadarAuth] Error updating database: ", err)
			}
			userlist = append(userlist, user[0])
		}
		userlistJSON, err := json.Marshal(userlist)
		if err != nil {
			log.Println("Error marshaling userlist to json: ", err)
			return
		}
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
				// Remove Minecraft User from whitelist
				mcName, err := redclient.Get("auth|" + currentUser + "|mc").Result()
				if err != nil {
					log.Println("[TasadarAuth] - Deleted User " + currentUser + " but found no minecraft User --> Ignoring")
				} else {
					if mcRunning {
						client, err := newClient(os.Getenv("RCON_ADDRESS"), 25575, os.Getenv("RCON_PASS"))
						if err == nil {
							response, err := client.sendCommand("whitelist remove " + mcName)
							if err == nil {
								if !strings.Contains(response, "Removed") {
									log.Println("[TasadarAuth] Removed mcUser " + mcName + "from whitelist, because " + currentUser + "was removed!")
								} else {
									log.Println("[TasadarAuth] Error while executing command on Minecraft Server to delete user "+mcName+" - error: ", err)
									// Example for mc blacklist queue management
									// mcAddToBlackListQueue(currentUser);
									msgAlpha <- "Please blacklist " + currentUser + " manually as there was an error:\n\n" + response
								}
							} else {
								log.Println("[TasadarAuth] Error connecting to Minecraft Server to delete user "+mcName+" - error: ", err)
								msgAlpha <- "Please blacklist " + currentUser + " manually - server was offline"
							}
						} else {
							log.Println("[TasadarAuth] Error connecting to Minecraft Server to delete user "+mcName+" - error: ", err)
							msgAlpha <- "Please blacklist " + currentUser + " manually - server was offline"
						}
					} else {
						log.Println("[TasadarAuth] Error while deleting mcUser " + mcName + ": Server variable offline")
						msgAlpha <- "Please blacklist " + currentUser + " manually - server was offline"
					}
				}
			}
		}

		// Put new list into database
		err = redclient.Set("userlist", string(userlistJSON), time.Hour*10).Err()
		if err != nil {
			log.Println("[TasadarAuth] Error updating database: ", err)
			msgAlpha <- "Error in TasadarAuth!"
		}
	}
	// Commented out for preventing Log Spamming
	//log.Println("[TasadarAuth] Finished Updating the database")
}

func authUser(username, password string) bool {
	val, err := redclient.Get("auth|" + username + "|hash").Result()
	if err != nil {
		return false
	}
	return checkPasswordHash(password, val)
}

func authGetGroupsString(username string) (string, error) {
	val, err := redclient.Get("auth|" + username + "|groups").Result()
	if err != nil {
		return "", err
	}
	if val == "" {
		return val, errors.New("empty groups string")
	}
	return val, nil
}

func authSetPassword(username, newpassword string) error {
	return errors.New("not implemented")
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
