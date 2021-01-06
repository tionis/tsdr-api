package data

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/tionis/tsdr-api/glyph"
)

// SendTokenData represents the data associated with a send-token
type SendTokenData struct {
	token      string
	userID     string
	adapters   []string
	validUntil time.Time
}

// GetSendTokenByID takes a sendTokenID and returns all data regarding the token if found
func (d *GlyphData) GetSendTokenByID(sendTokenID string) (SendTokenData, error) {
	stmt, err := d.db.Prepare(`SELECT userID, adapters, validUntil FROM sendtokens WHERE sendToken = $1`)
	if err != nil {
		return SendTokenData{}, err
	}
	rows, err := stmt.Query(sendTokenID)
	if err != nil {
		return SendTokenData{}, err
	}
	var userID, adapters string
	var validUntil time.Time
	err = rows.Scan(&userID, &adapters, &validUntil)
	if err != nil {
		return SendTokenData{}, err
	}
	var adaptersArray []string
	err = json.Unmarshal([]byte(adapters), &adaptersArray)
	if err != nil {
		return SendTokenData{}, err
	}
	if validUntil.Before(time.Now()) {
		go d.DeleteSendToken(sendTokenID)
		return SendTokenData{}, glyph.ErrSendTokenInvalid
	}
	return SendTokenData{sendTokenID, userID, adaptersArray, validUntil}, nil
}

// GetSendTokensByUserID takes an userID and return all data to all tokens registered to this user if found
func (d *GlyphData) GetSendTokensByUserID(userID string) ([]SendTokenData, error) {
	stmt, err := d.db.Prepare(`SELECT sendToken, adapters, validUntil FROM sendtokens WHERE userID = $1`)
	if err != nil {
		return []SendTokenData{}, err
	}
	rows, err := stmt.Query(userID)
	if err != nil {
		return []SendTokenData{}, err
	}
	var sendToken, adapters string
	var validUntil time.Time
	var sendTokens []SendTokenData
	for rows.Next() {
		err = rows.Scan(&sendToken, &adapters, &validUntil)
		if err != nil {
			return []SendTokenData{}, err
		}
		var adaptersArray []string
		err = json.Unmarshal([]byte(adapters), &adaptersArray)
		if err != nil {
			return []SendTokenData{}, err
		}
		if validUntil.Before(time.Now()) {
			go d.DeleteSendToken(sendToken)
			continue
		}
		sendTokens = append(sendTokens, SendTokenData{sendToken, userID, adaptersArray, validUntil})
	}
	return sendTokens, nil
}

// DeleteSendToken takes the id of a sendtoken and deletes it
func (d *GlyphData) DeleteSendToken(token string) error {
	stmt, err := d.db.Prepare(`DELETE FROM sendtokens WHERE sendToken = $1`)
	if err != nil {
		return err
	}
	_, err = stmt.Exec(token)
	if err != nil {
		if err == sql.ErrNoRows {
			return glyph.ErrNoSuchToken
		}
		return err
	}
	return nil
}

// AddSendToken takes metadata for a send token saves it and returns the token value itself
func (d *GlyphData) AddSendToken(userID, adapters []string, validUntil time.Time) (string, error) {
	if validUntil.Before(time.Now()) {
		return "", glyph.ErrSendTokenInvalid
	}
	stmt, err := d.db.Prepare(`INSERT INTO sendtokens (sendToken, userID, adapters, validUntil) VALUES ($1, $2, $3, $4)`)
	if err != nil {
		return "", err
	}
	adaptersString, err := json.Marshal(adapters)
	token := generateSendToken()
	return token, stmt.QueryRow(token, userID, adaptersString, validUntil).Err()
}

// UpdateSendToken takes SendTokenData and updates the database so it is stored, overwriting existing ones
func (d *GlyphData) UpdateSendToken(token SendTokenData) error {
	return errors.New("not implemented yet") // TODO

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

func generateSendToken() string {
	return strings.ReplaceAll(uuid.New().String(), "-", "")
}
