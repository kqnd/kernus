package details

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/kiev/kernus/internal/models"
)

var (
	logsCache     []string
	logsCacheID   string
	logsCacheTime time.Time
	logsMu        sync.Mutex
	ansiRegex     = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
)

const maxLogLines = 50

func RenderLogs(b *strings.Builder, c *models.Container) {
	if len(c.Logs) == 0 {
		b.WriteString("\n  [gray]No logs available[white]\n")
		b.WriteString("\n  Press 'r' to refresh\n")
		return
	}

	logsMu.Lock()
	cacheValid := logsCacheID == c.ID &&
		time.Since(logsCacheTime) < 5*time.Second &&
		len(logsCache) == len(c.Logs)
	if !cacheValid {
		logsCache = c.Logs
		logsCacheID = c.ID
		logsCacheTime = time.Now()
	}
	logsMu.Unlock()

	lines := c.GetRecentLogs(maxLogLines)
	totalLines := len(c.Logs)

	b.WriteString("\n")
	if totalLines > maxLogLines {
		fmt.Fprintf(b, "  [gray]... showing last %d of %d lines ...[white]\n\n", maxLogLines, totalLines)
	}

	startNum := totalLines - len(lines) + 1
	for i, line := range lines {
		lineNum := startNum + i
		cleaned := stripAnsiCodes(line)
		ts := extractTimestamp(cleaned)
		level := detectLogLevel(cleaned)
		color := levelColor(level)

		msg := cleaned
		if ts != "" {
			idx := strings.Index(cleaned, ts)
			if idx >= 0 {
				after := idx + len(ts)
				if after < len(cleaned) {
					msg = strings.TrimSpace(cleaned[after:])
				}
			}
			fmt.Fprintf(b, "  [gray]%4d[white]  [gray]%s[white]  [%s]%s[white]\n", lineNum, ts, color, msg)
		} else {
			fmt.Fprintf(b, "  [gray]%4d[white]  [%s]%s[white]\n", lineNum, color, msg)
		}
	}

	b.WriteString("\n  [gray]Press 'r' to refresh | Scroll to navigate[white]\n")
}

func stripAnsiCodes(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

func extractTimestamp(s string) string {
	if len(s) < 20 {
		return ""
	}
	prefix := s
	if len(prefix) > 30 {
		prefix = prefix[:30]
	}
	t, err := time.Parse("2006-01-02T15:04:05", prefix[:19])
	if err != nil {
		return ""
	}
	return t.Format("15:04:05.000")
}

func detectLogLevel(s string) string {
	lower := strings.ToLower(s)
	errorWords := []string{"error", "err", "fatal", "panic", "failed", "exception"}
	for _, w := range errorWords {
		if strings.Contains(lower, w) {
			return "ERROR"
		}
	}
	warnWords := []string{"warn", "warning", "deprecated"}
	for _, w := range warnWords {
		if strings.Contains(lower, w) {
			return "WARN"
		}
	}
	debugWords := []string{"debug", "dbg", "trace", "verbose"}
	for _, w := range debugWords {
		if strings.Contains(lower, w) {
			return "DEBUG"
		}
	}
	return "INFO"
}

func levelColor(level string) string {
	switch level {
	case "ERROR":
		return "red"
	case "WARN":
		return "yellow"
	case "DEBUG":
		return "gray"
	default:
		return "aqua"
	}
}
