package progress

import (
	"regexp"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// GitProgressWriter wraps git progress output and sends bubbletea messages
type GitProgressWriter struct {
	program *tea.Program
}

// NewGitProgressWriter creates a writer that parses git output and sends progress messages
func NewGitProgressWriter(p *tea.Program) *GitProgressWriter {
	return &GitProgressWriter{program: p}
}

// Write implements io.Writer, parsing git progress output
func (w *GitProgressWriter) Write(p []byte) (n int, err error) {
	line := string(p)

	// Parse git progress patterns
	percent, detail := parseGitProgress(line)
	if percent >= 0 {
		w.program.Send(SubProgressMsg{
			Percent: percent,
			Detail:  detail,
		})
	}

	return len(p), nil
}

// parseGitProgress parses git clone/fetch progress output
// Returns percent (0-100) and detail string, or -1 if not a progress line
func parseGitProgress(line string) (float64, string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return -1, ""
	}

	// Patterns like:
	// "Receiving objects:  67% (156/233)"
	// "Resolving deltas:  100% (45/45)"
	// "Compressing objects:  50% (5/10)"
	// "Counting objects: 100% (233/233), done."
	patterns := []struct {
		re     *regexp.Regexp
		prefix string
	}{
		{regexp.MustCompile(`Receiving objects:\s+(\d+)%\s+\((\d+)/(\d+)\)`), "Receiving objects"},
		{regexp.MustCompile(`Resolving deltas:\s+(\d+)%\s+\((\d+)/(\d+)\)`), "Resolving deltas"},
		{regexp.MustCompile(`Compressing objects:\s+(\d+)%\s+\((\d+)/(\d+)\)`), "Compressing objects"},
		{regexp.MustCompile(`Counting objects:\s+(\d+)%\s+\((\d+)/(\d+)\)`), "Counting objects"},
		{regexp.MustCompile(`Enumerating objects:\s+(\d+)`), "Enumerating objects"},
	}

	for _, p := range patterns {
		if matches := p.re.FindStringSubmatch(line); matches != nil {
			if len(matches) >= 2 {
				percent, _ := strconv.ParseFloat(matches[1], 64)
				detail := p.prefix
				if len(matches) >= 4 {
					detail = p.prefix + ": " + matches[2] + "/" + matches[3]
				}
				return percent, detail
			}
		}
	}

	// Handle "Enumerating objects: 233" (no percentage)
	if strings.HasPrefix(line, "Enumerating objects:") {
		re := regexp.MustCompile(`Enumerating objects:\s+(\d+)`)
		if matches := re.FindStringSubmatch(line); matches != nil {
			return 0, "Enumerating objects: " + matches[1]
		}
	}

	return -1, ""
}

// ByteProgressWriter tracks bytes written and sends progress messages
type ByteProgressWriter struct {
	program    *tea.Program
	total      int64
	written    int64
	lastUpdate float64
}

// NewByteProgressWriter creates a writer that tracks byte progress
func NewByteProgressWriter(p *tea.Program, total int64) *ByteProgressWriter {
	return &ByteProgressWriter{
		program: p,
		total:   total,
	}
}

// Write implements io.Writer, tracking bytes and sending progress
func (w *ByteProgressWriter) Write(p []byte) (n int, err error) {
	n = len(p)
	w.written += int64(n)

	if w.total > 0 {
		percent := float64(w.written) / float64(w.total) * 100

		// Only send updates every 1% to avoid flooding
		if percent-w.lastUpdate >= 1 || percent >= 100 {
			w.lastUpdate = percent
			w.program.Send(SubProgressMsg{
				Percent: percent,
				Detail:  formatBytes(w.written) + " / " + formatBytes(w.total),
			})
		}
	}

	return n, nil
}

// formatBytes formats bytes into human-readable string
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return strconv.FormatInt(bytes, 10) + " B"
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return strconv.FormatFloat(float64(bytes)/float64(div), 'f', 1, 64) + " " + string("KMGTPE"[exp]) + "B"
}
