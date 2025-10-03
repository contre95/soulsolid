package jobs

import (
	"fmt"
	"strings"
)

// ParseAndColorLogContent parses log lines and adds HTML color spans based on log level and content
func ParseAndColorLogContent(content string) string {
	lines := strings.Split(content, "\n")
	coloredLines := make([]string, 0, len(lines))

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			coloredLines = append(coloredLines, line)
			continue
		}

		coloredLine := line

		// Extract log level from the line (format: level=LEVEL)
		level := extractLogLevel(line)

		// Apply coloring rules
		switch level {
		case "ERROR":
			// All ERROR level logs -> red
			coloredLine = fmt.Sprintf(`<span class="log-error">%s</span>`, line)
		case "WARN", "WARNING":
			// All WARN level logs -> yellow
			coloredLine = fmt.Sprintf(`<span class="log-warning">%s</span>`, line)
		case "INFO":
			// Check for successful import messages first
			if strings.Contains(line, `color=green`) {
				coloredLine = fmt.Sprintf(`<span class="log-green">%s</span>`, line)
			} else if strings.Contains(line, `color=blue`) {
				coloredLine = fmt.Sprintf(`<span class="log-blue">%s</span>`, line)
			} else if strings.Contains(line, `color=orange`) {
				coloredLine = fmt.Sprintf(`<span class="log-orange">%s</span>`, line)
			} else if strings.Contains(line, `color=violet`) {
				coloredLine = fmt.Sprintf(`<span class="log-violet">%s</span>`, line)
			}
			// Other INFO logs remain white (no span)
		default:
			// All other logs remain white (no span)
		}
		coloredLines = append(coloredLines, coloredLine)
	}

	return strings.Join(coloredLines, "\n")
}

// extractLogLevel extracts the log level from a log line
func extractLogLevel(line string) string {
	// Look for "level=LEVEL" pattern
	if idx := strings.Index(line, "level="); idx != -1 {
		start := idx + 6 // length of "level="
		// Find the next space or end of line
		end := strings.Index(line[start:], " ")
		if end == -1 {
			end = len(line[start:])
		}
		level := line[start : start+end]
		return strings.ToUpper(level)
	}
	return ""
}
