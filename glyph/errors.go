package glyph

import "errors"

// ErrNoCommandMatched represents the state in which no command could be matched
var ErrNoCommandMatched = errors.New("no command was matched")

// ErrNoUserDataFound is thrown if now data for the user with the specified key could be found
var ErrNoUserDataFound = errors.New("no userdata found")

// ErrUserNotFound is thrown when the searched user could not be found
var ErrUserNotFound = errors.New("user not found")

// ErrNoMappingFound is thrown if no valid mapping from a 3PID to an userID could be found
var ErrNoMappingFound = errors.New("no mapping between 3PID and userID found")

// ErrNoSuchSession is thrown if no auth session with the given ID could be founc
var ErrNoSuchSession = errors.New("no session with given ID could be found")

// ErrMatrixIDInvalid is thrown if the given matrix ID does not follow the rules of the matrix convention
var ErrMatrixIDInvalid = errors.New("matrix id not valid")

// ErrAdapterNotRegistered is thrown if the given adapter is not currently registered in registry
var ErrAdapterNotRegistered = errors.New("adapter not registered")
