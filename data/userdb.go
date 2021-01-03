package data

import (
	"database/sql"
	"errors"
	"regexp"

	"github.com/tionis/tsdr-api/glyph"
)

var isValidMatrixID = regexp.MustCompile(`(?m)^@[a-z\-_]+:([A-Za-z0-9-]{1,63}\.)+[A-Za-z]{2,6}$`)

// GetUserIDFromValueOfKey returns the userID where key and value are matched,
// this is mostly used to map chat platform ids to the main id
func GetUserIDFromValueOfKey(key, value string) (string, error) {
	stmt, err := db.Prepare(`SELECT userID FROM userdata WHERE key = $1 AND value = $2`)
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

// UserAdd adds an user with given userID, email and isAdmin parameters
func UserAdd(userID, email string, isAdmin bool) error {
	if !isValidMatrixID.MatchString(userID) {
		return glyph.ErrMatrixIDInvalid
	}
	stmt, err := db.Prepare(`INSERT INTO users (userID, email, isAdmin) VALUES ($1, $2, $3)`)
	if err != nil {
		return err
	}
	row := stmt.QueryRow(userID, email, isAdmin)
	if err := row.Err(); err != nil {
		return err
	}
	return nil
}

// UserIsAdmin return true if the user with given userID is an tasadar admin
func UserIsAdmin(userID string) (bool, error) {
	stmt, err := db.Prepare(`SELECT isAdmin FROM users WHERE userID = $1`)
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
func UserSetMail(userID, email string) error {
	stmt, err := db.Prepare(`UPDATE users SET email = $2 WHERE userID = $1`)
	if err != nil {
		return err
	}
	row := stmt.QueryRow(userID, email)
	if err := row.Err(); err != nil {
		return err
	}
	return nil
}

// UserSetAdminStatus makes an User tasdar admin if true and takes away that privilege if false
func UserSetAdminStatus(userID string, isAdmin bool) error {
	stmt, err := db.Prepare(`UPDATE users SET isAdmin = $2 WHERE userID = $1`)
	if err != nil {
		return err
	}
	row := stmt.QueryRow(userID, isAdmin)
	if err := row.Err(); err != nil {
		return err
	}
	return nil
}

// UserDelete deletes the user and all associated data with it (except associated quotes)
func UserDelete(userID string) error {
	// Delete user from users, the sql server will take care of deleting data from the other tables referencing the user
	stmt, err := db.Prepare(`DELETE FROM users WHERE userID = $1`)
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
func SetUserData(userID, key string, value string) error {
	stmt, err := db.Prepare(`INSERT INTO userdata (userID, key, value) VALUES ($1, $2, $3) ON CONFLICT (userID) DO UPDATE SET value = $3;`)
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
func GetUserData(userID, key string) (string, error) {
	stmt, err := db.Prepare(`SELECT value FROM userdata WHERE userID = $1 AND key = $2`)
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
func DeleteUserData(userID, key string) error {
	stmt, err := db.Prepare(`DELETE FROM userdata WHERE userID = $1 AND key = $2`)
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
func DoesUserIDExist(matrixUserID string) (bool, error) {
	stmt, err := db.Prepare(`SELECT userID FROM quotes WHERE userID = $1`)
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
func AddAuthSession(authWorker func() error, userID string) (string, error) {
	// TODO
	return "", errors.New("not implemented yet")
}

// GetAuthSessionStatus is used to get the status of an auth session with the ID
func GetAuthSessionStatus(authSessionID string) (string, error) {
	// TODO
	// if no session with ID found glyph.ErrNoSuchSession
	return "", errors.New("not implemented yet")
}

// AuthenticateSession sets the session with given ID as authenticated
func AuthenticateSession(matrixUserID, authSessionID string) error {
	// TODO
	// if no session with ID found glyph.ErrNoSuchSession
	// if session does not belong to user glyph.ErrSessionNotOfUser
	return errors.New("not implemented yet")
}

// DeleteSession deletes the session with given ID
func DeleteSession(authSessionID string) error {
	// TODO
	// if no session with ID found glyph.ErrNoSuchSession
	return errors.New("not implemented yet")
}

// GetAuthSessions return the state of all sessions registered to the user
func GetAuthSessions(matrixID string) ([]string, error) {
	// TODO
	return []string{}, errors.New("not implemented yet")
}
