// Package minequery is a simple library to query Minequery enabled Minecraft servers.
package main

import (
	"bufio"
	"encoding/json"
	"net"
	"strconv"
	"time"
)

const (
	// DefaultPort is the default Minequery network port.
	// More information: https://github.com/vexsoftware/minequery/blob/v1.5/java/net/minestatus/minequery/Minequery.java#L65
	DefaultPort = 25566
)

// Response is a representation of the Minequery server QUERY_JSON response.
// More information: https://github.com/vexsoftware/minequery/blob/v1.5/java/net/minestatus/minequery/Request.java#L108
type Response struct {
	ServerPort  int      `json:"serverPort"`
	PlayerCount int      `json:"playerCount"`
	MaxPlayers  int      `json:"maxPlayers"`
	PlayerList  []string `json:"playerList"`
}

// Query connects and requests a response from the Minequery server at the specified address.
// Follows Minequery server's Java implementation: https://github.com/vexsoftware/minequery/blob/v1.5/java/net/minestatus/minequery/Request.java#L21
func Query(address string, port uint16, timeout time.Duration) (Response, error) {
	var resp Response

	deadline := time.Now().Add(timeout)

	conn, err := net.DialTimeout("tcp", address+":"+strconv.Itoa(int(port)), timeout)
	if err != nil {
		return resp, err
	}
	defer conn.Close()

	if err = conn.SetDeadline(deadline); err != nil {
		return resp, err
	}

	_, err = conn.Write([]byte("QUERY_JSON\n"))
	if err != nil {
		return resp, err
	}

	payload, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return resp, err
	}

	if err = json.Unmarshal([]byte(payload), &resp); err != nil {
		return resp, err
	}

	return resp, nil
}
