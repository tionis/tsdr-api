package data

import (
	"errors"
)

// AdapterMessage represents a message being sent to an adapter
type AdapterMessage struct {
	UserID  string
	Message string
}

// RegisterAdapterChannel registeres the given channel in the channel map with given adapter ID
func (d *GlyphData) RegisterAdapterChannel(adapterID string, channel chan AdapterMessage) error {
	// TODO add channel to map
	return errors.New("not implemented yet")
}

// GetAdapterChannel gets the channel to send an AdapterMessage through
func (d *GlyphData) GetAdapterChannel(adapterID string) (chan AdapterMessage, error) {
	// TODO implement this
	return nil, errors.New("not implemented yet")
}
