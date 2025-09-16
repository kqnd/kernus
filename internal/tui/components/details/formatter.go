package details

import (
	"fmt"
	"time"
)

type Formatter struct{}

func NewFormatter() *Formatter {
	return &Formatter{}
}

func (f *Formatter) FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%dB", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func (f *Formatter) FormatNumber(num int64) string {
	if num < 1000 {
		return fmt.Sprintf("%d", num)
	} else if num < 1000000 {
		return fmt.Sprintf("%.1fK", float64(num)/1000)
	} else if num < 1000000000 {
		return fmt.Sprintf("%.1fM", float64(num)/1000000)
	}
	return fmt.Sprintf("%.1fB", float64(num)/1000000000)
}

func (f *Formatter) FormatTime(t time.Time) string {
	if t.IsZero() {
		return "[gray]Never[white]"
	}
	return t.Format("2006-01-02 15:04:05")
}

func (f *Formatter) FormatExitCode(code int) string {
	if code == 0 {
		return "[green]0 (success)[white]"
	} else if code > 0 {
		return fmt.Sprintf("[red]%d (error)[white]", code)
	}
	return "[gray]N/A[white]"
}

func (f *Formatter) TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func (f *Formatter) GetUsageColor(percentage float64) string {
	if percentage < 50 {
		return "green"
	} else if percentage < 80 {
		return "yellow"
	}
	return "red"
}
