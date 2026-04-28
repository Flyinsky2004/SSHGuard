package main

import (
	"encoding/json"
	"log"
	"net"
	"os"
	"time"
)

func listenSocket(socketPath string, events chan<- *SSHEvent) (net.Listener, error) {
	os.Remove(socketPath)

	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: socketPath, Net: "unix"})
	if err != nil {
		return nil, err
	}

	if err := os.Chmod(socketPath, 0600); err != nil {
		listener.Close()
		return nil, err
	}

	go func() {
		for {
			conn, err := listener.AcceptUnix()
			if err != nil {
				close(events)
				return
			}

			conn.SetReadDeadline(time.Now().Add(5 * time.Second))

			buf := make([]byte, 4096)
			n, err := conn.Read(buf)
			if err != nil {
				conn.Write([]byte("ERROR: read failed\n"))
				conn.Close()
				continue
			}

			var pe pamEvent
			if err := json.Unmarshal(buf[:n], &pe); err != nil {
				conn.Write([]byte("ERROR: invalid JSON\n"))
				conn.Close()
				continue
			}

			if pe.Type != "ssh_login" || pe.User == "" {
				conn.Write([]byte("ERROR: invalid event\n"))
				conn.Close()
				continue
			}

			ts, err := time.Parse(time.RFC3339, pe.Timestamp)
			if err != nil {
				ts = time.Now()
			}

			hostname := pe.Hostname
			if hostname == "" {
				hostname, _ = os.Hostname()
			}

			events <- &SSHEvent{
				Timestamp:  ts,
				Hostname:   hostname,
				User:       pe.User,
				SourceIP:   pe.SourceIP,
				SourcePort: pe.SourcePort,
				AuthMethod: pe.AuthMethod,
			}

			conn.Write([]byte("OK\n"))
			conn.Close()
		}
	}()

	log.Printf("正在监听 Unix Socket: %s", socketPath)
	return listener, nil
}
