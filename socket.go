package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"
)

func listenSocket(socketPath string, events chan<- *SSHEvent) (net.Listener, error) {
	if info, err := os.Lstat(socketPath); err == nil {
		if info.Mode()&os.ModeSocket == 0 {
			return nil, fmt.Errorf("Socket 路径已被非 Socket 文件占用: %s", socketPath)
		}
		if err := os.Remove(socketPath); err != nil {
			return nil, err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

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

			buf, err := bufio.NewReader(io.LimitReader(conn, 4096)).ReadBytes('\n')
			if err != nil {
				conn.Write([]byte("ERROR: read failed\n"))
				conn.Close()
				continue
			}

			var pe pamEvent
			if err := json.Unmarshal(buf, &pe); err != nil {
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
