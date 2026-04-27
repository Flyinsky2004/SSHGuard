package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"time"

	"github.com/nxadm/tail"
)

var sshAcceptedRe = regexp.MustCompile(
	`^(\w{3}\s+\d+\s+\d{2}:\d{2}:\d{2})\s+(\S+)\s+sshd\[\d+\]:\s+Accepted\s+(\S+)\s+for\s+(\S+)\s+from\s+(\S+)\s+port\s+(\d+)`,
)

func monitorLog(logPath string, events chan<- *SSHEvent) error {
	if _, err := os.Stat(logPath); err != nil {
		return fmt.Errorf("log file not found %s: %w", logPath, err)
	}

	t, err := tail.TailFile(logPath, tail.Config{
		Follow:    true,
		ReOpen:    true,
		MustExist: true,
		Location:  &tail.SeekInfo{Offset: 0, Whence: os.SEEK_END},
	})
	if err != nil {
		return fmt.Errorf("tail file %s: %w", logPath, err)
	}

	go func() {
		for line := range t.Lines {
			if line.Err != nil {
				log.Printf("tail error: %v", line.Err)
				continue
			}
			ev := parseLine(line.Text)
			if ev != nil {
				events <- ev
			}
		}
		close(events)
	}()

	log.Printf("monitoring SSH log: %s", logPath)
	return nil
}

func parseLine(line string) *SSHEvent {
	matches := sshAcceptedRe.FindStringSubmatch(line)
	if matches == nil {
		return nil
	}
	ts, err := parseTimestamp(matches[1])
	if err != nil {
		log.Printf("parse timestamp: %v (line: %s)", err, line)
		return nil
	}
	return &SSHEvent{
		Timestamp:  ts,
		Hostname:   matches[2],
		AuthMethod: matches[3],
		User:       matches[4],
		SourceIP:   matches[5],
		SourcePort: matches[6],
	}
}

// parseTimestamp parses syslog-style "Mon DD HH:MM:SS" timestamps.
// Since syslog doesn't include the year, we infer it from the current year,
// handling year boundary near Dec/Jan (close enough for our use case).
func parseTimestamp(s string) (time.Time, error) {
	t, err := time.Parse("Jan 2 15:04:05", s)
	if err != nil {
		return time.Time{}, err
	}
	now := time.Now()
	t = t.AddDate(now.Year(), 0, 0)
	return t, nil
}
