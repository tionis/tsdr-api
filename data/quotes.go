package data

import (
	"database/sql"
	"errors"
	"strconv"
	"time"

	_ "github.com/heroku/x/hmetrics/onload" // Heroku advanced go metrics
	"github.com/tionis/tsdr-api/glyph"      // This provides glyph specific errors
)

// GetQuoteDBHandler returns a glyph.QuoteDB object that exposes
// all functions neeeded to interact with the Quote database
func (d *GlyphData) GetQuoteDBHandler() *glyph.QuoteDB {
	return &glyph.QuoteDB{
		AddQuote:         d.AddQuote,
		GetRandomQuote:   d.GetRandomQuote,
		GetQuoteOfTheDay: d.GetQuoteOfTheDayOfUser,
		SetQuoteOfTheDay: d.SetQuoteOfTheDayOfUser,
	}
}

// GetRandomQuote gets a random quote with specified parameters. If they are emtpy strings they are ignored
func (d *GlyphData) GetRandomQuote(byAuthor, inLanguage, inUniverse string) (glyph.Quote, error) {
	stmt, err := d.db.Prepare(`SELECT id, quote, author, language, universe FROM quotes WHERE (length($1)=0 OR author=$1) AND (length($2)=0 OR language=$2) AND (length($3)=0 OR universe=$3) ORDER BY RANDOM() LIMIT 1`)
	if err != nil {
		return glyph.Quote{}, err
	}
	row := stmt.QueryRow(byAuthor, inLanguage, inUniverse)

	var quote, author, language, universe string
	var id int
	err = row.Scan(&id, &quote, &author, &language, &universe)
	if err != nil {
		if err == sql.ErrNoRows {
			return glyph.Quote{}, glyph.ErrNoQuotesFound
		}
		return glyph.Quote{}, err
	}
	return glyph.Quote{
		ID:       strconv.Itoa(id),
		Content:  quote,
		Author:   author,
		Language: language,
		Universe: universe}, nil
}

// AddQuote adds specified quote to database
func (d *GlyphData) AddQuote(quote glyph.Quote) error {
	stmt, err := d.db.Prepare(`INSERT INTO quotes (quote, author, language, universe) VALUES ($1, $2, $3, $4)`)
	if err != nil {
		return err
	}
	_, err = stmt.Query(quote.Content, quote.Author, quote.Language, quote.Universe)
	if err != nil {
		return err
	}
	return nil
}

// GetQuoteOfTheDayOfUser gets the quote of the day object of the user with the given userID
func (d *GlyphData) GetQuoteOfTheDayOfUser(userID string) (glyph.QuoteOfTheDay, error) {
	stmt, err := d.db.Prepare(`SELECT id, quote, author, language, universe, validUntil FROM quotes LEFT JOIN qotd ON quotes.ID = qotd.quoteID WHERE qotd.userID = $1;`)
	if err != nil {
		return glyph.QuoteOfTheDay{}, err
	}
	row := stmt.QueryRow(userID)
	var quote, author, language, universe string
	var validUntil time.Time
	var id int
	err = row.Scan(&id, &quote, &author, &language, &universe, &validUntil)
	if err != nil {
		if err == sql.ErrNoRows {
			return glyph.QuoteOfTheDay{}, glyph.ErrNoUserDataFound
		}
		return glyph.QuoteOfTheDay{}, err
	}
	return glyph.QuoteOfTheDay{
		Quote: glyph.Quote{
			ID:       strconv.Itoa(id),
			Content:  quote,
			Author:   author,
			Language: language,
			Universe: universe,
		},
		ValidUntil: validUntil,
	}, nil
}

// SetQuoteOfTheDayOfUser sets the quote of the day object of the user with the given userID
func (d *GlyphData) SetQuoteOfTheDayOfUser(userID string, quoteOfTheDay glyph.QuoteOfTheDay) error {
	stmt, err := d.db.Prepare(`INSERT INTO qotd (userID, quoteID, validUntil) VALUES ($1, $2, $3) ON CONFLICT (userID) DO UPDATE SET quoteID = $2, validUntil = $3;`)
	if err != nil {
		return err
	}
	quoteID, err := strconv.Atoi(quoteOfTheDay.Quote.ID)
	if err != nil {
		return err
	}
	if quoteID == 0 {
		return errors.New("no quote id given")
	}
	row := stmt.QueryRow(userID, quoteID, quoteOfTheDay.ValidUntil)
	err = row.Err()
	if err != nil {
		return err
	}
	return nil
}
