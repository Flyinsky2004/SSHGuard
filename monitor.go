package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"time"

	"github.com/nxadm/tail"
)

// sshAcceptedRe matches "Accepted" lines from both syslog and systemd-journald formats.
// Syslog:    Jan 22 10:15:30 myhost sshd[12345]: Accepted password for root from 1.2.3.4 port 12345
// Journald:  2026-04-27T14:26:38.670099+08:00 localhost sshd[277030]: Accepted publickey for root from 1.2.3.4 port 34102 ssh2: ED25519 SHA256:...
var sshAcceptedRe = regexp.MustCompile(
	`^(\w{3}\s+\d{1,2}\s+\d{2}:\d{2}:\d{2}|\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\S*)\s+(\S+)\s+sshd\[\d+\]:\s+Accepted\s+(\S+)\s+for\s+(\S+)\s+from\s+(\S+)\s+port\s+(\d+)`,
)

var syslogTsRe = regexp.MustCompile(`^[A-Z][a-z]{2}\s`)
var iso8601TsRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T`)

func monitorLog(logPath string, events chan<- *SSHEvent) error {
	if _, err := os.Stat(logPath); err != nil {
		return fmt.Errorf("日志文件未找到 %s: %w", logPath, err)
	}

	t, err := tail.TailFile(logPath, tail.Config{
		Follow:    true,
		ReOpen:    true,
		MustExist: true,
		Location:  &tail.SeekInfo{Offset: 0, Whence: os.SEEK_END},
	})
	if err != nil {
		return fmt.Errorf("监听日志文件失败 %s: %w", logPath, err)
	}

	go func() {
		for line := range t.Lines {
			if line.Err != nil {
				log.Printf("日志读取错误: %v", line.Err)
				continue
			}
			ev := parseLine(line.Text)
			if ev != nil {
				events <- ev
			}
		}
		close(events)
	}()

	log.Printf("正在监控 SSH 日志: %s", logPath)
	return nil
}

func parseLine(line string) *SSHEvent {
	matches := sshAcceptedRe.FindStringSubmatch(line)
	if matches == nil {
		return nil
	}
	ts, err := parseTimestamp(matches[1])
	if err != nil {
		log.Printf("解析时间戳失败: %v (日志行: %s)", err, line)
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

// parseTimestamp parses both syslog and ISO 8601 timestamps.
func parseTimestamp(s string) (time.Time, error) {
	// syslog: "Jan 2 15:04:05"
	if syslogTsRe.MatchString(s) {
		t, err := time.Parse("Jan 2 15:04:05", s)
		if err != nil {
			return time.Time{}, err
		}
		t = t.AddDate(time.Now().Year(), 0, 0)
		return t, nil
	}
	// ISO 8601: "2026-04-27T14:26:38.670099+08:00"
	if iso8601TsRe.MatchString(s) {
		for _, layout := range []string{
			time.RFC3339Nano,
			"2006-01-02T15:04:05.999999-07:00",
			"2006-01-02T15:04:05-07:00",
			"2006-01-02T15:04:05.999999Z",
			"2006-01-02T15:04:05Z",
		} {
			if t, err := time.Parse(layout, s); err == nil {
				return t, nil
			}
		}
	}
	return time.Time{}, fmt.Errorf("未知的时间戳格式: %s", s)
}
