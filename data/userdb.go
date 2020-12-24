package data

import (
	"database/sql"
	"errors"

	"github.com/tionis/tsdr-api/glyph"
)

// GetUserIDFromValueOfKey returns the userID where key and value are matched,
// this is mostly used to map chat platform ids to the main id
func GetUserIDFromValueOfKey(key, value string) (string, error) {
	// TODO
	// if no mapping found return glyph.ErrNoMappingFound
	return "@tionis:tasadar.net", nil
}

// SetUserData sets the key in the bucket in the data of a user to the data from value
func SetUserData(userID, bucket, key string, value string) error {
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
func GetUserData(userID, bucket, key string) (string, error) {
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
	// TODO
	return errors.New("not implemented yet")
}

// DoesUserIDExist checks if an user with the given (matrix) user id exists
func DoesUserIDExist(matrixUserID string) (bool, error) {
	// TODO
	return false, errors.New("not implemented yet")
}

// MigrateUserToNewID migrates all data of an user to a new ID (while checking if the new one is a valid matrix address)
func MigrateUserToNewID(oldMatrixUserID, newMatrixUserID string) error {
	// TODO
	return errors.New("not implemented yet")
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
