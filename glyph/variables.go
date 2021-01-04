package glyph

import (
	"regexp"
	"time"
)

// standardContextDelay is the standard ttl of chat contexts
var standardContextDelay = time.Minute * 5

// IsValidUserName checks if the string is a valid username (after matrix ID and thus tasadar.net specification)
var IsValidUserName = regexp.MustCompile(`(?m)^[a-z\-_]+$`)

// IsValidMatrixID checks if the string is a valid matrix id (but ignores the case in which the domain starts or ends with an dash)
var IsValidMatrixID = regexp.MustCompile(`(?m)^@[a-z\-_]+:([A-Za-z0-9-]{1,63}\.)+[A-Za-z]{2,6}$`)
