package data

import "github.com/tionis/tsdr-api/glyph" // This provides errors for glyph specific data operation

// AdapterMessage represents a message being sent to an adapter
type AdapterMessage struct {
	UserID  string
	Message string
}

// RegisterAdapterChannel registeres the given channel in the channel map with given adapter ID
func (d *GlyphData) RegisterAdapterChannel(adapterID string, channel chan AdapterMessage) error {
	d.adapterMessageChannelsLock.Lock()
	defer d.adapterMessageChannelsLock.Unlock()

	d.adapterMessageChannels[adapterID] = channel
	return nil
}

// GetAdapterChannel gets the channel to send an AdapterMessage through
func (d *GlyphData) GetAdapterChannel(adapterID string) (chan AdapterMessage, error) {
	d.adapterMessageChannelsLock.RLock()
	val, ok := d.adapterMessageChannels[adapterID]
	d.adapterMessageChannelsLock.RUnlock()
	if !ok {
		return nil, glyph.ErrAdapterNotRegistered
	}
	return val, nil
}
