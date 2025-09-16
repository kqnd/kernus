package details

import (
	"fmt"
	"strings"
	"time"

	"github.com/kern/internal/models"
)

type LogsTab struct {
	formatter     *Formatter
	lastUpdate    time.Time
	cachedLogs    []string
	cachedContent string
}

func NewLogsTab() *LogsTab {
	return &LogsTab{
		formatter: NewFormatter(),
	}
}

func (l *LogsTab) Render(container *models.Container) string {
	if container == nil {
		return ""
	}

	if l.shouldUseCache(container.Logs) {
		return l.cachedContent
	}

	content := l.renderLogs(container)
	l.updateCache(container.Logs, content)
	return content
}

func (l *LogsTab) shouldUseCache(logs []string) bool {
	return time.Since(l.lastUpdate) < 5*time.Second &&
		l.cachedContent != "" &&
		l.logsEqual(logs, l.cachedLogs)
}

func (l *LogsTab) updateCache(logs []string, content string) {
	l.cachedLogs = make([]string, len(logs))
	copy(l.cachedLogs, logs)
	l.cachedContent = content
	l.lastUpdate = time.Now()
}

func (l *LogsTab) logsEqual(logs1, logs2 []string) bool {
	if len(logs1) != len(logs2) {
		return false
	}
	for i, log := range logs1 {
		if log != logs2[i] {
			return false
		}
	}
	return true
}

func (l *LogsTab) renderLogs(container *models.Container) string {
	if len(container.Logs) == 0 {
		return l.renderNoLogs(container)
	}

	var result strings.Builder
	result.WriteString("[yellow]Container Logs[white]\n\n")
	result.WriteString(fmt.Sprintf("[gray]Container: %s | Lines: %d | Last Update: %s[white]\n",
		container.ShortName(),
		len(container.Logs),
		l.formatter.FormatTime(time.Now())))
	result.WriteString("[gray]" + strings.Repeat("─", 70) + "[white]\n\n")

	maxLines := 50
	startIndex := 0
	if len(container.Logs) > maxLines {
		startIndex = len(container.Logs) - maxLines
		result.WriteString(fmt.Sprintf("[yellow]... showing last %d lines of %d total ...[white]\n\n",
			maxLines, len(container.Logs)))
	}

	for i := startIndex; i < len(container.Logs); i++ {
		logLine := container.Logs[i]
		formattedLine := l.formatLogLine(logLine, i+1)
		result.WriteString(formattedLine + "\n")
	}

	result.WriteString("\n[gray]" + strings.Repeat("─", 70) + "[white]\n")
	result.WriteString("[darkgray]Press 'r' to refresh logs | Use scroll to navigate[white]")

	return result.String()
}

func (l *LogsTab) renderNoLogs(container *models.Container) string {
	statusMessage := ""
	if container.Status != models.StatusRunning {
		statusMessage = fmt.Sprintf("\n[yellow]Container is %s - logs may be limited[white]", container.Status)
	}

	return fmt.Sprintf(`[yellow]Container Logs[white]

[gray]┌─────────────────────────────────────────────────────────┐[white]
[gray]│[white]  No logs available for container:                    [gray]│[white]
[gray]│[white]  %s[gray]│[white]
[gray]│[white]                                                     [gray]│[white]
[gray]│[white]  Possible reasons:                                  [gray]│[white]
[gray]│[white]  • Container has not produced any output            [gray]│[white]
[gray]│[white]  • Logs are being written to files instead         [gray]│[white]
[gray]│[white]  • Container just started                          [gray]│[white]
[gray]│[white]  • Logging driver is not configured                [gray]│[white]
[gray]└─────────────────────────────────────────────────────────┘[white]%s

[darkgray]Press 'r' to refresh logs[white]`,
		l.formatContainerName(container.ShortName()),
		statusMessage)
}

func (l *LogsTab) formatContainerName(name string) string {
	maxLen := 50
	if len(name) > maxLen {
		return name[:maxLen-3] + "..."
	}
	return fmt.Sprintf("%-50s", name)
}

func (l *LogsTab) formatLogLine(logLine string, lineNumber int) string {
	cleanLine := l.stripAnsiCodes(logLine)

	timestamp, message := l.extractTimestampAndMessage(cleanLine)

	logLevel := l.detectLogLevel(message)
	color := l.getLogLevelColor(logLevel)

	maxMessageLength := 80
	if len(message) > maxMessageLength {
		message = message[:maxMessageLength-3] + "..."
	}

	if timestamp != "" {
		return fmt.Sprintf("[gray]%3d[white] [darkgray]%s[white] [%s]%s[white]",
			lineNumber, timestamp, color, message)
	}

	return fmt.Sprintf("[gray]%3d[white] [%s]%s[white]", lineNumber, color, message)
}

func (l *LogsTab) stripAnsiCodes(input string) string {
	result := input

	result = strings.ReplaceAll(result, "\x1b[0m", "")
	result = strings.ReplaceAll(result, "[0m", "")
	result = strings.ReplaceAll(result, "\x1b[90m", "")
	result = strings.ReplaceAll(result, "[90m", "")
	result = strings.ReplaceAll(result, "]0m", "")

	result = strings.ReplaceAll(result, "║│", "")
	result = strings.ReplaceAll(result, "│║", "")
	result = strings.ReplaceAll(result, "║", "")
	result = strings.ReplaceAll(result, "│", "")

	result = strings.ReplaceAll(result, "\x1b[", "")
	result = strings.ReplaceAll(result, "\033[", "")

	result = strings.ReplaceAll(result, "\x00", "")
	result = strings.ReplaceAll(result, "\x07", "")
	result = strings.ReplaceAll(result, "\x08", "")

	return strings.TrimSpace(result)
}

func (l *LogsTab) extractTimestampAndMessage(logLine string) (timestamp, message string) {
	cleanLine := logLine

	cleanLine = strings.ReplaceAll(cleanLine, "                                      ", " ")

	timestamps := l.findAllTimestamps(cleanLine)
	if len(timestamps) > 0 {
		timestamp = timestamps[0]

		firstTimestampIndex := strings.Index(cleanLine, timestamp)
		if firstTimestampIndex >= 0 {
			afterFirstTimestamp := cleanLine[firstTimestampIndex+len(timestamp):]

			for _, ts := range timestamps[1:] {
				afterFirstTimestamp = strings.ReplaceAll(afterFirstTimestamp, ts, "")
			}

			message = l.extractCleanMessage(afterFirstTimestamp)

			if len(strings.TrimSpace(message)) < 5 {
				if bracketStart := strings.Index(afterFirstTimestamp, "["); bracketStart >= 0 {
					if bracketEnd := strings.Index(afterFirstTimestamp[bracketStart:], "]"); bracketEnd >= 0 {
						possibleMessage := afterFirstTimestamp[bracketStart+bracketEnd+1:]
						possibleMessage = l.extractCleanMessage(possibleMessage)
						if len(strings.TrimSpace(possibleMessage)) > len(strings.TrimSpace(message)) {
							message = possibleMessage
						}
					}
				}
			}

			return l.formatTimestamp(timestamp), message
		}
	}

	message = l.extractCleanMessage(cleanLine)
	return "", message
}

func (l *LogsTab) findAllTimestamps(text string) []string {
	var timestamps []string

	for i := 0; i < len(text)-19; i++ {
		if i+30 < len(text) {
			for j := i + 19; j <= i+30 && j < len(text); j++ {
				substr := text[i:j]
				if l.isISO8601Timestamp(substr[:19]) && (j == len(text) || !l.isTimestampChar(text[j])) {
					timestamps = append(timestamps, substr)
					i = j - 1
					break
				}
			}
		}
	}

	return timestamps
}

func (l *LogsTab) isTimestampChar(c byte) bool {
	return (c >= '0' && c <= '9') || c == '.' || c == 'Z'
}

func (l *LogsTab) formatTimestamp(timestamp string) string {
	if strings.Contains(timestamp, ".") && len(timestamp) > 25 {
		parts := strings.Split(timestamp, ".")
		if len(parts) == 2 {
			microseconds := parts[1]
			if len(microseconds) > 6 {
				microseconds = microseconds[:3]
			}
			timestamp = parts[0] + "." + microseconds
		}
	}

	if len(timestamp) >= 19 {
		timePart := timestamp[11:19]
		if strings.Contains(timestamp, ".") {
			dotIndex := strings.Index(timestamp, ".")
			if dotIndex > 11 && dotIndex < len(timestamp)-1 {
				microseconds := timestamp[dotIndex+1:]
				if strings.HasSuffix(microseconds, "Z") {
					microseconds = microseconds[:len(microseconds)-1]
				}
				if len(microseconds) > 3 {
					microseconds = microseconds[:3]
				}
				timePart = timestamp[11:dotIndex] + "." + microseconds
			}
		}
		return timePart
	}

	return timestamp
}

func (l *LogsTab) isISO8601Timestamp(s string) bool {
	if len(s) < 19 {
		return false
	}

	return s[4] == '-' && s[7] == '-' && s[10] == 'T' &&
		s[13] == ':' && s[16] == ':'
}

func (l *LogsTab) extractCleanMessage(text string) string {
	text = strings.TrimSpace(text)

	for len(text) > 25 && l.isISO8601Timestamp(text[:19]) {
		endIdx := 19
		for endIdx < len(text) && l.isTimestampChar(text[endIdx]) {
			endIdx++
		}
		text = text[endIdx:]
		text = strings.TrimSpace(text)
	}

	prefixesToRemove := []string{
		"nundb::process_request",
		"ws::io",
		"[INFO]", "[ERROR]", "[WARN]", "[DEBUG]",
		"INFO:", "ERROR:", "WARN:", "DEBUG:",
	}

	for _, prefix := range prefixesToRemove {
		if strings.HasPrefix(text, prefix) {
			text = strings.TrimSpace(text[len(prefix):])
			break
		}
	}

	if strings.Contains(text, "[") && strings.Contains(text, "]") {
		parts := strings.Split(text, "]")
		if len(parts) > 1 {
			possibleMessage := parts[len(parts)-1]
			possibleMessage = strings.TrimSpace(possibleMessage)
			if len(possibleMessage) > 0 {
				text = possibleMessage
			}
		}
	}

	words := strings.Fields(text)
	cleanWords := make([]string, 0, len(words))

	for _, word := range words {
		if strings.Contains(word, "::") && len(word) < 30 {
			continue
		}

		if len(word) <= 4 && (strings.HasSuffix(word, "m") ||
			strings.HasPrefix(word, "[") ||
			strings.ContainsAny(word, "[]{}()")) {
			continue
		}

		if len(word) > 10 && l.isAllDigits(word) {
			continue
		}

		cleanWords = append(cleanWords, word)
	}

	result := strings.Join(cleanWords, " ")

	if len(strings.TrimSpace(result)) < 5 {
		result = text
		result = strings.ReplaceAll(result, "90m", "")
		result = strings.ReplaceAll(result, "[12...", "")
		result = strings.ReplaceAll(result, "...", "")

		if strings.Contains(result, "Server processed message") {
			result = "Server processed message"
		}

		result = strings.TrimSpace(result)
	}

	if len(strings.TrimSpace(result)) == 0 {
		result = "[Log entry]"
	}

	return result
}

func (l *LogsTab) isAllDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return len(s) > 0
}

func (l *LogsTab) detectLogLevel(message string) string {
	upperLine := strings.ToUpper(message)

	if strings.Contains(upperLine, "ERROR") || strings.Contains(upperLine, "ERR") ||
		strings.Contains(upperLine, "FATAL") || strings.Contains(upperLine, "PANIC") ||
		strings.Contains(upperLine, "FAILED") || strings.Contains(upperLine, "EXCEPTION") {
		return "ERROR"
	}

	if strings.Contains(upperLine, "WARN") || strings.Contains(upperLine, "WARNING") ||
		strings.Contains(upperLine, "DEPRECATED") {
		return "WARN"
	}

	if strings.Contains(upperLine, "DEBUG") || strings.Contains(upperLine, "DBG") ||
		strings.Contains(upperLine, "TRACE") || strings.Contains(upperLine, "VERBOSE") {
		return "DEBUG"
	}

	if strings.Contains(upperLine, "ACCEPTED") || strings.Contains(upperLine, "STARTED") ||
		strings.Contains(upperLine, "CONNECTED") || strings.Contains(upperLine, "LISTENING") ||
		strings.Contains(upperLine, "REQUEST") || strings.Contains(upperLine, "RESPONSE") {
		return "INFO"
	}

	if len(strings.TrimSpace(message)) > 10 {
		return "INFO"
	}

	return "DEFAULT"
}

func (l *LogsTab) getLogLevelColor(level string) string {
	switch level {
	case "ERROR":
		return "red"
	case "WARN":
		return "yellow"
	case "INFO":
		return "cyan"
	case "DEBUG":
		return "gray"
	default:
		return "white"
	}
}
