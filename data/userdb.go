package data

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/tionis/tsdr-api/glyph" // This provides glyph-specific errors
)

// GetUserDBHandler returns a UserDB Handler object that exposes all function needed to interact woth the userDB
func (d *GlyphData) GetUserDBHandler() *glyph.UserDB {
	return &glyph.UserDB{
		GetUserData:                  d.GetUserData,
		SetUserData:                  d.SetUserData,
		DeleteUserData:               d.DeleteUserData,
		GetUserIDFromValueOfKey:      d.GetUserIDFromValueOfKey,
		DoesMatrixUserIDExist:        d.DoesUserIDExist,
		AddAuthSession:               d.AddAuthSession,
		AddAuthSessionWithAdapterAdd: d.AddAuthSessionWithAdapterAdd,
		AuthenticateSession:          d.AuthenticateSession,
		DeauthenticateSession:        d.DeauthenticateSession,
		DeleteSession:                d.DeleteSession,
		GetAuthSessions:              d.GetAuthSessions,
		RegisterNewUser:              d.UserAdd,
	}
}

// GetUserIDFromValueOfKey returns the userID where key and value are matched,
// this is mostly used to map chat platform ids to the main id
func (d *GlyphData) GetUserIDFromValueOfKey(key, value string) (string, error) {
	stmt, err := d.db.Prepare(`SELECT userID FROM userdata WHERE key = $1 AND value = $2`)
	if err != nil {
		return "", err
	}
	row := stmt.QueryRow(key, value)

	var userID string
	err = row.Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", glyph.ErrNoMappingFound
		}
		return "", err
	}
	return userID, nil
}

// UserAdd adds an user with given userID, email and isAdmin parameters,
// preferredAdapters is a json string array containing the adapter the user wants to be notified on.
func (d *GlyphData) UserAdd(userID, email string, isAdmin bool, preferredAdapters []string) error {
	if !glyph.IsValidMatrixID.MatchString(userID) {
		return glyph.ErrMatrixIDInvalid
	}
	data, err := json.Marshal(preferredAdapters)
	if err != nil {
		return nil
	}
	stmt, err := d.db.Prepare(`INSERT INTO users (userID, email, isAdmin, preferredAdapters) VALUES ($1, $2, $3, $4)`)
	if err != nil {
		return err
	}
	return stmt.QueryRow(userID, email, isAdmin, string(data)).Err()
}

// UserIsAdmin return true if the user with given userID is an tasadar admin
func (d *GlyphData) UserIsAdmin(userID string) (bool, error) {
	stmt, err := d.db.Prepare(`SELECT isAdmin FROM users WHERE userID = $1`)
	if err != nil {
		return false, err
	}

	var isAdmin bool
	err = stmt.QueryRow(userID).Scan(&isAdmin)
	if err != nil {
		return false, err
	}
	return isAdmin, nil
}

// UserSetMail sets the email address of an user
func (d *GlyphData) UserSetMail(userID, email string) error {
	stmt, err := d.db.Prepare(`UPDATE users SET email = $2 WHERE userID = $1`)
	if err != nil {
		return err
	}
	return stmt.QueryRow(userID, email).Err()
}

// UserSetPreferredAdapters sets the preferred adapters array of a user to the given string array
func (d *GlyphData) UserSetPreferredAdapters(userID, preferredAdapters []string) error {
	data, err := json.Marshal(preferredAdapters)
	if err != nil {
		return err
	}
	stmt, err := d.db.Prepare(`UPDATE users SET preferredAdapters = $2 WHERE userID = $1`)
	if err != nil {
		return err
	}
	return stmt.QueryRow(userID, string(data)).Err()
}

// UserGetPreferredAdapters gets the preferred adapters array of a user and returns it as a string array
func (d *GlyphData) UserGetPreferredAdapters(userID string) ([]string, error) {
	var out []string
	var preferredAdaptersJSON string
	stmt, err := d.db.Prepare(`SELECT preferredAdapters FROM users WHERE userID = $1`)
	if err != nil {
		return nil, err
	}
	err = stmt.QueryRow(userID).Scan(preferredAdaptersJSON)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal([]byte(preferredAdaptersJSON), &out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// UserSetAdminStatus makes an User tasdar admin if true and takes away that privilege if false
func (d *GlyphData) UserSetAdminStatus(userID string, isAdmin bool) error {
	stmt, err := d.db.Prepare(`UPDATE users SET isAdmin = $2 WHERE userID = $1`)
	if err != nil {
		return err
	}
	return stmt.QueryRow(userID, isAdmin).Err()
}

// UserDelete deletes the user and all associated data with it (except associated quotes)
func (d *GlyphData) UserDelete(userID string) error {
	// Delete user from users, the sql server will take care of deleting data from the other tables referencing the user
	stmt, err := d.db.Prepare(`DELETE FROM users WHERE userID = $1`)
	if err != nil {
		return err
	}
	err = stmt.QueryRow(userID).Err()
	if err != nil {
		return err
	}
	return nil
}

// SetUserData sets the key in the bucket in the data of a user to the data from value
func (d *GlyphData) SetUserData(userID, key, value string) error {
	stmt, err := d.db.Prepare(`INSERT INTO userdata (userID, key, value) VALUES ($1, $2, $3) ON CONFLICT (userID) DO UPDATE SET value = $3;`)
	if err != nil {
		return err
	}
	row := stmt.QueryRow(userID, key, value)
	err = row.Err()
	if err != nil {
		return err
	}
	return nil
}

// GetUserData gets the key in the bucket in the data of a user
func (d *GlyphData) GetUserData(userID, key string) (string, error) {
	stmt, err := d.db.Prepare(`SELECT value FROM userdata WHERE userID = $1 AND key = $2`)
	if err != nil {
		return "", err
	}
	row := stmt.QueryRow(userID, key)

	var value string
	err = row.Scan(&value)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", glyph.ErrNoUserDataFound
		}
		return "", err
	}

	return value, nil
}

// DeleteUserData deletes user data for a given key
func (d *GlyphData) DeleteUserData(userID, key string) error {
	stmt, err := d.db.Prepare(`DELETE FROM userdata WHERE userID = $1 AND key = $2`)
	if err != nil {
		return err
	}
	err = stmt.QueryRow(userID, key).Err()
	if err != nil {
		return err
	}
	return nil
}

// DoesUserIDExist checks if an user with the given (matrix) user id exists
func (d *GlyphData) DoesUserIDExist(matrixUserID string) (bool, error) {
	stmt, err := d.db.Prepare(`SELECT userID FROM users WHERE userID = $1`)
	if err != nil {
		return false, err
	}
	var userID string
	err = stmt.QueryRow(matrixUserID).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// AddAuthSession adds an auth session with an authWorker that is executed when the session is authenticated.
// The functions returns an error and the ID of the auth session
func (d *GlyphData) AddAuthSession(key, value, matrixUserID string) (string, error) {
	return d.authSessionAddGeneral(key, value, matrixUserID, 0)
}

// AddAuthSessionWithAdapterAdd dds an auth session and adapterID, adapter-specific userID and
// a general matrixUserID. When the auth succeeds the adapter-specific userID will be written
// to the adapterID+"ID" userdata field and the adapter is added to the adapters userdata field
// as part of the json array
func (d *GlyphData) AddAuthSessionWithAdapterAdd(adapter, adapterID, matrixUserID string) (string, error) {
	return d.authSessionAddGeneral(adapter, adapterID, matrixUserID, 1)
}

// authSessionAddGeneral is a general abstraction above the insert into authSession statement
func (d *GlyphData) authSessionAddGeneral(key, value, userID string, mode int) (string, error) {
	authToken := ""
	validUntil := time.Now().Add(glyph.AuthSessionDelay)
	stmt, err := d.db.Prepare(`INSERT INTO authsessions (authToken, userID, key, value, validUntil, mode) VALUES ($1, $2, $3, $4, $5, $6)`)
	if err != nil {
		return authToken, err
	}
	i := 0
	for err != nil && i < 5 {
		//var res sql.Result
		// Try again 5 times on error then return it
		authToken := glyph.GenerateAuthSessionID()
		_, err = stmt.Exec(authToken, userID, key, value, validUntil, mode)
		i++
	}
	return authToken, err
}

// AuthenticateSession sets the session with given ID as authenticated
func (d *GlyphData) AuthenticateSession(matrixUserID, authToken string) error {
	// First get the values to set
	stmt, err := d.db.Prepare(`SELECT key, value, validUntil, mode FROM authsessions WHERE userID = $1 AND authToken = $2`)
	if err != nil {
		return nil
	}
	rows, err := stmt.Query(matrixUserID, authToken)
	if err != nil {
		if err == sql.ErrNoRows {
			return glyph.ErrNoSuchSession
		}
		return nil
	}
	var key, value string
	var mode int
	var validUntil time.Time
	rows.Scan(&key, &value, &validUntil, &mode)
	if validUntil.Before(time.Now()) {
		go d.DeleteSession(authToken)
		return glyph.ErrNoSuchSession
	}
	switch mode {
	case 0:
		err = d.SetUserData(matrixUserID, key, value)
		if err != nil {
			return err
		}
		go d.DeleteSession(authToken)
	case 1:
		// Add adapter specific user ID to User Data
		err = d.SetUserData(matrixUserID, key+"ID", value)
		if err != nil {
			return err
		}
		go d.DeleteSession(authToken)
		// Add adapter to list of available adapters by reading, converting, appending, converting and writing
		value, err := d.GetUserData(matrixUserID, "adapters")
		if err != nil {
			return err
		}
		var adapters []string
		err = json.Unmarshal([]byte(value), &adapters)
		if err != nil {
			return err
		}
		adapters = append(adapters, key)
		adapterJSON, err := json.Marshal(adapters)
		if err != nil {
			return err
		}
		return d.SetUserData(matrixUserID, "adapters", string(adapterJSON))
	}
	return nil
}

// DeauthenticateSession removes the connection between a chat platform account and the matrixAccount
// Notice: this does not work on the matrixAccount as they are linked by default
func (d *GlyphData) DeauthenticateSession(userID, adapterID string) error {
	if adapterID == "matrix" {
		return glyph.ErrImmutableMatrixID
	}
	// Remove adapter specific user ID to User Data
	err := d.DeleteUserData(userID, adapterID+"ID")
	if err != nil {
		return err
	}

	// Remove adapter from adapters list
	value, err := d.GetUserData(userID, "adapters")
	if err != nil {
		return err
	}
	var adapters []string
	err = json.Unmarshal([]byte(value), &adapters)
	if err != nil {
		return err
	}
	for i, item := range adapters {
		if item == adapterID {
			adapters = removeFromStringSliceIgnoringOrder(adapters, i)
			break
		}
	}
	if len(adapters) == 0 {
		return d.DeleteUserData(userID, "adapters")
	}
	adapterJSON, err := json.Marshal(adapters)
	if err != nil {
		return err
	}
	return d.SetUserData(userID, "adapters", string(adapterJSON))
}

func removeFromStringSliceIgnoringOrder(s []string, i int) []string {
	s[i] = s[len(s)-1]
	// We do not need to put s[i] at the end, as it will be discarded anyway
	return s[:len(s)-1]
}

// DeleteSession deletes the session with given ID
func (d *GlyphData) DeleteSession(authToken string) error {
	stmt, err := d.db.Prepare(`DELETE FROM authsessions WHERE authToken = $1`)
	if err != nil {
		return err
	}
	_, err = stmt.Exec(authToken)
	if err != nil {
		if err == sql.ErrNoRows {
			return glyph.ErrNoSuchSession
		}
		return err
	}
	return nil
}

// GetAuthSessions return the state of all sessions registered to the user
func (d *GlyphData) GetAuthSessions(matrixID string) ([]string, error) {
	stmt, err := d.db.Prepare(`DELETE FROM userdata WHERE userID = $1`)
	if err != nil {
		return nil, err
	}
	rows, err := stmt.Query(matrixID)
	if err != nil {
		return nil, err
	}
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	// Result is your slice string.
	rawResult := make([][]byte, len(cols))
	result := make([]string, len(cols))

	dest := make([]interface{}, len(cols)) // A temporary interface{} slice
	for i := range rawResult {
		dest[i] = &rawResult[i] // Put pointers to each string in the interface slice
	}

	for rows.Next() {
		err = rows.Scan(dest...)
		if err != nil {
			return nil, err
		}

		for i, raw := range rawResult {
			if raw == nil {
				result[i] = "\\N"
			} else {
				result[i] = string(raw)
			}
		}
	}
	return result, nil
}
