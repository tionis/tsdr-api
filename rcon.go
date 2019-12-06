package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"strings"
	"time"
)

// Stolen from https://github.com/seeruk/minecraft-rcon

const (
	packetIDBadAuth       = -1
	payloadMaxSize        = 1460
	serverdataAuth        = 3
	serverdataExeccommand = 2
)

type payload struct {
	packetID   int32  // 4 bytes
	packetType int32  // 4 bytes
	packetBody []byte // Varies
}

func (p *payload) calculatePacketSize() int32 {
	return int32(len(p.packetBody) + 10)
}

func newClient(host string, port int, pass string) (*Client, error) {
	address := fmt.Sprintf("%s:%d", host, port)

	conn, err := net.DialTimeout("tcp", address, 10*time.Second)
	if err != nil {
		return nil, err
	}

	client := new(Client)
	client.connection = conn
	client.password = pass

	err = client.sendAuthentication(pass)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// Client is an RCON client based around the Valve RCON Protocol, see more about the protocol in the
// Valve Wiki: https://developer.valvesoftware.com/wiki/Source_RCON_Protocol
type Client struct {
	connection net.Conn
	password   string
}

func (c *Client) sendAuthentication(pass string) error {
	payload := createPayload(serverdataAuth, pass)

	_, err := c.sendPayload(payload)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) sendCommand(command string) (string, error) {
	payload := createPayload(serverdataExeccommand, command)

	response, err := c.sendPayload(payload)
	if err != nil {
		return "", err
	}

	// Trim null bytes
	response.packetBody = bytes.Trim(response.packetBody, "\x00")

	return strings.TrimSpace(string(response.packetBody)), nil
}

func (c *Client) reconnect() error {
	conn, err := net.DialTimeout("tcp",
		c.connection.RemoteAddr().String(), 10*time.Second)
	if err != nil {
		return err
	}

	c.connection = conn

	err = c.sendAuthentication(c.password)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) sendPayload(request *payload) (*payload, error) {
	packet, err := createPacketFromPayload(request)
	if err != nil {
		return nil, err
	}

	_, err = c.connection.Write(packet)
	if err != nil {
		return nil, err
	}

	response, err := createPayloadFromPacket(c.connection)
	if err != nil {
		return nil, err
	}

	if response.packetID == packetIDBadAuth {
		return nil, errors.New("rcon: authentication unsuccessful")
	}

	return response, nil
}

func createPacketFromPayload(payload *payload) ([]byte, error) {
	var buf bytes.Buffer

	_ = binary.Write(&buf, binary.LittleEndian, payload.calculatePacketSize())
	_ = binary.Write(&buf, binary.LittleEndian, payload.packetID)
	_ = binary.Write(&buf, binary.LittleEndian, payload.packetType)
	_ = binary.Write(&buf, binary.LittleEndian, payload.packetBody)
	_ = binary.Write(&buf, binary.LittleEndian, [2]byte{})

	if buf.Len() >= payloadMaxSize {
		return nil, fmt.Errorf("rcon: payload exceeded maximum allowed size of %d", payloadMaxSize)
	}

	return buf.Bytes(), nil
}

func createPayload(packetType int, body string) *payload {
	return &payload{
		packetID:   rand.Int31(),
		packetType: int32(packetType),
		packetBody: []byte(body),
	}
}

func createPayloadFromPacket(packetReader io.Reader) (*payload, error) {
	var packetSize int32
	var packetID int32
	var packetType int32

	errs := newStack()

	// Read packetSize, packetID, and packetType
	errs.add(binary.Read(packetReader, binary.LittleEndian, &packetSize))
	errs.add(binary.Read(packetReader, binary.LittleEndian, &packetID))
	errs.add(binary.Read(packetReader, binary.LittleEndian, &packetType))

	if !errs.empty() {
		return nil, errors.New("createPayloadFromPacket: Failed reading bytes")
	}

	// Body size length is packet size without the empty string byte at the end
	packetBody := make([]byte, packetSize-8)

	_, err := io.ReadFull(packetReader, packetBody)
	if err != nil {
		return nil, err
	}

	result := new(payload)
	result.packetID = packetID
	result.packetType = packetType
	result.packetBody = packetBody

	return result, nil
}

// Err Handling
type stack struct {
	errors []error
}

func newStack() *stack {
	return new(stack)
}

func (s *stack) add(error error) {
	if error != nil {
		s.errors = append(s.errors, error)
	}
}

func (s *stack) empty() bool {
	return len(s.errors) == 0
}
