package data

import (
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/tionis/tsdr-api/glyph"
)

// GetSendTokenHandler returns a SendTokenHandlerFuncs object that implements
// the function the glyph bot needs to interact with the SendToken DB
func (d *GlyphData) GetSendTokenHandler() glyph.SendTokenHandlerFuncs {
	return glyph.SendTokenHandlerFuncs{
		GetSendTokenByID:      d.GetSendTokenByID,
		GetSendTokensByUserID: d.GetSendTokensByUserID,
		DeleteSendToken:       d.DeleteSendToken,
		AddSendToken:          d.AddSendToken,
		UpdateSendToken:       d.UpdateSendToken,
	}
}

// GetSendTokenByID takes a sendTokenID and returns all data regarding the token if found
func (d *GlyphData) GetSendTokenByID(sendTokenID string) (glyph.SendTokenData, error) {
	stmt, err := d.db.Prepare(`SELECT userID, adapters, validUntil FROM sendtokens WHERE sendToken = $1`)
	if err != nil {
		return glyph.SendTokenData{}, err
	}
	rows, err := stmt.Query(sendTokenID)
	if err != nil {
		return glyph.SendTokenData{}, err
	}
	var userID, adapters string
	var validUntil time.Time
	err = rows.Scan(&userID, &adapters, &validUntil)
	if err != nil {
		return glyph.SendTokenData{}, err
	}
	var adaptersArray []string
	err = json.Unmarshal([]byte(adapters), &adaptersArray)
	if err != nil {
		return glyph.SendTokenData{}, err
	}
	if validUntil.Before(time.Now()) {
		go d.DeleteSendToken(sendTokenID)
		return glyph.SendTokenData{}, glyph.ErrSendTokenInvalid
	}
	return glyph.SendTokenData{Token: sendTokenID, UserID: userID, Adapters: adaptersArray, ValidUntil: validUntil}, nil
}

// GetSendTokensByUserID takes an userID and return all data to all tokens registered to this user if found
func (d *GlyphData) GetSendTokensByUserID(userID string) ([]glyph.SendTokenData, error) {
	stmt, err := d.db.Prepare(`SELECT sendToken, adapters, validUntil FROM sendtokens WHERE userID = $1`)
	if err != nil {
		return []glyph.SendTokenData{}, err
	}
	rows, err := stmt.Query(userID)
	if err != nil {
		return []glyph.SendTokenData{}, err
	}
	var sendToken, adapters string
	var validUntil time.Time
	var sendTokens []glyph.SendTokenData
	for rows.Next() {
		err = rows.Scan(&sendToken, &adapters, &validUntil)
		if err != nil {
			return []glyph.SendTokenData{}, err
		}
		var adaptersArray []string
		err = json.Unmarshal([]byte(adapters), &adaptersArray)
		if err != nil {
			return []glyph.SendTokenData{}, err
		}
		if validUntil.Before(time.Now()) {
			go d.DeleteSendToken(sendToken)
			continue
		}
		sendTokens = append(sendTokens, glyph.SendTokenData{Token: sendToken, UserID: userID, Adapters: adaptersArray, ValidUntil: validUntil})
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

// UpdateSendToken takes glyph.SendTokenData and updates the database so it is stored, overwriting existing ones
func (d *GlyphData) UpdateSendToken(token glyph.SendTokenData) error {
	stmt, err := d.db.Prepare(`INSERT INTO sendtokens (sendToken, userID, adapters, validUntil) VALUES ($1, $2, $3) ON CONFLICT (sendToken) DO UPDATE SET userID = $2, adapters = $3, validUntil = $4 ;`)
	if err != nil {
		return err
	}
	adapterString, err := json.Marshal(token.Adapters)
	if err != nil {
		return err
	}
	row := stmt.QueryRow(token.Token, token.UserID, adapterString, token.ValidUntil)
	err = row.Err()
	if err != nil {
		return err
	}
	return nil
}

// genereateSendToken returns a generated randomly unique token to use for send-tokens
func generateSendToken() string {
	return strings.ReplaceAll(uuid.New().String(), "-", "")
}
