package data

import (
	"database/sql"

	_ "github.com/heroku/x/hmetrics/onload" // Heroku advanced go metrics
	"github.com/tionis/tsdr-api/glyph"
)

// GetRandomQuote gets a random quote with specified parameters. If they are emtpy strings they are ignored
func GetRandomQuote(byAuthor, inLanguage, inUniverse string) (glyph.Quote, error) {
	stmt, err := db.Prepare(`SELECT quote, author, language, universe FROM quotes WHERE (length($1)=0 OR author=$1) AND (length($2)=0 OR language=$2) AND (length($3)=0 OR universe=$3) ORDER BY RANDOM() LIMIT 1`)
	if err != nil {
		return glyph.Quote{}, err
	}
	row := stmt.QueryRow(byAuthor, inLanguage, inUniverse)

	var quote, author, language, universe string
	err = row.Scan(&quote, &author, &language, &universe)
	if err != nil {
		if err == sql.ErrNoRows {
			return glyph.Quote{}, glyph.ErrNoQuotesFound
		}
		return glyph.Quote{}, err
	}
	return glyph.Quote{
		Content:  quote,
		Author:   author,
		Language: language,
		Universe: universe}, nil
}

// AddQuote adds specified quote to database
func AddQuote(quote glyph.Quote) error {
	stmt, err := db.Prepare(`INSERT INTO quotes (quote, author, language, universe) VALUES ($1, $2, $3, $4)`)
	if err != nil {
		return err
	}
	_, err = stmt.Query(quote.Content, quote.Author, quote.Language, quote.Universe)
	if err != nil {
		return err
	}
	return nil
}
