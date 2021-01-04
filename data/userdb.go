package data

import (
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/tionis/tsdr-api/glyph" // This provides glyph-specific errors
)

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
func (d *GlyphData) UserAdd(userID, email string, isAdmin bool, preferredAdaptersJSON string) error {
	if !glyph.IsValidMatrixID.MatchString(userID) {
		return glyph.ErrMatrixIDInvalid
	}
	stmt, err := d.db.Prepare(`INSERT INTO users (userID, email, isAdmin, preferredAdapters) VALUES ($1, $2, $3, $4)`)
	if err != nil {
		return err
	}
	return stmt.QueryRow(userID, email, isAdmin, preferredAdaptersJSON).Err()
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
	stmt, err := d.db.Prepare(`SELECT userID FROM quotes WHERE userID = $1`)
	if err != nil {
		return false, err
	}
	var userID string
	err = stmt.QueryRow(matrixUserID).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, glyph.ErrUserNotFound
		}
		return false, err
	}
	return true, nil
}

// AddAuthSession adds an auth session with an authWorker that is executed when the session is authenticated.
// The functions returns an error and the ID of the auth session
func (d *GlyphData) AddAuthSession(authWorker func() error, userID string) (string, error) {
	// TODO this will need a new in memory store
	// this will also add it to the list of sessions of userID
	return "", errors.New("not implemented yet")
}

// GetAuthSessionStatus is used to get the status of an auth session with the ID
func (d *GlyphData) GetAuthSessionStatus(authSessionID string) (string, error) {
	// TODO this will interface with the in memory store
	// if no session with ID found glyph.ErrNoSuchSession
	return "", errors.New("not implemented yet")
}

// AuthenticateSession sets the session with given ID as authenticated
func (d *GlyphData) AuthenticateSession(matrixUserID, authSessionID string) error {
	// TODO this will delete the session and execute the function attached to it
	// if no session with ID found glyph.ErrNoSuchSession
	// if session does not belong to user glyph.ErrSessionNotOfUser
	return errors.New("not implemented yet")
}

// DeleteSession deletes the session with given ID
func (d *GlyphData) DeleteSession(authSessionID string) error {
	// TODO this will delete the session from the in memory store
	// if no session with ID found glyph.ErrNoSuchSession
	// this will also delete it from the list of sessions of userID
	return errors.New("not implemented yet")
}

// GetAuthSessions return the state of all sessions registered to the user
func (d *GlyphData) GetAuthSessions(matrixID string) ([]string, error) {
	// TODO this will get all session IDs from the session list by userID and
	// then aggregates all states of the found sessions
	return []string{}, errors.New("not implemented yet")
}
