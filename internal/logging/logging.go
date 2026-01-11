package logging

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

// LogLevel represents the severity of a log entry
type LogLevel string

const (
	LevelDebug LogLevel = "DEBUG"
	LevelInfo  LogLevel = "INFO"
	LevelWarn  LogLevel = "WARN"
	LevelError LogLevel = "ERROR"
)

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     LogLevel  `json:"level"`
	Source    string    `json:"source"`
	Message   string    `json:"message"`
}

// RingBuffer is a thread-safe circular buffer for log entries
type RingBuffer struct {
	entries []LogEntry
	size    int
	head    int
	count   int
	mu      sync.RWMutex
}

// NewRingBuffer creates a new ring buffer with the specified capacity
func NewRingBuffer(size int) *RingBuffer {
	return &RingBuffer{
		entries: make([]LogEntry, size),
		size:    size,
	}
}

// Add adds a new entry to the ring buffer
func (rb *RingBuffer) Add(entry LogEntry) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.entries[rb.head] = entry
	rb.head = (rb.head + 1) % rb.size
	if rb.count < rb.size {
		rb.count++
	}
}

// GetAll returns all entries in chronological order
func (rb *RingBuffer) GetAll() []LogEntry {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	result := make([]LogEntry, rb.count)
	start := (rb.head - rb.count + rb.size) % rb.size

	for i := 0; i < rb.count; i++ {
		result[i] = rb.entries[(start+i)%rb.size]
	}

	return result
}

// Count returns the number of entries in the buffer
func (rb *RingBuffer) Count() int {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.count
}

// LogWriter is a custom io.Writer that captures logs to a ring buffer
type LogWriter struct {
	buffer    *RingBuffer
	output    io.Writer
	mu        sync.Mutex
	lineBuffer strings.Builder
}

// Global logger instance
var (
	globalBuffer *RingBuffer
	globalWriter *LogWriter
	initOnce     sync.Once
)

// Initialize sets up the global log buffer and returns a writer to use with log.SetOutput
func Initialize(size int) io.Writer {
	initOnce.Do(func() {
		globalBuffer = NewRingBuffer(size)
		globalWriter = &LogWriter{
			buffer: globalBuffer,
			output: os.Stderr,
		}
	})
	return globalWriter
}

// GetBuffer returns the global ring buffer
func GetBuffer() *RingBuffer {
	return globalBuffer
}

// Log level patterns to detect from log output
var levelPatterns = map[LogLevel]*regexp.Regexp{
	LevelError: regexp.MustCompile(`(?i)\b(error|fatal|panic|failed|failure)\b`),
	LevelWarn:  regexp.MustCompile(`(?i)\b(warn|warning)\b`),
	LevelDebug: regexp.MustCompile(`(?i)\b(debug)\b`),
}

// Source patterns to detect from log output
var sourcePatterns = map[string]*regexp.Regexp{
	"scheduler":  regexp.MustCompile(`(?i)\b(scheduler|task|job|cron)\b`),
	"indexer":    regexp.MustCompile(`(?i)\b(indexer|torznab|newznab|prowlarr|search)\b`),
	"importer":   regexp.MustCompile(`(?i)\b(import|upgrade)\b`),
	"download":   regexp.MustCompile(`(?i)\b(download|torrent|nzb|qbittorrent|transmission|sabnzbd|nzbget|grab)\b`),
	"scanner":    regexp.MustCompile(`(?i)\b(scan|scanner|library)\b`),
	"metadata":   regexp.MustCompile(`(?i)\b(metadata|tmdb|tvdb|imdb)\b`),
	"auth":       regexp.MustCompile(`(?i)\b(auth|login|logout|token|session|user)\b`),
	"api":        regexp.MustCompile(`(?i)\b(api|request|response|handler|endpoint)\b`),
}

// Write implements io.Writer
func (lw *LogWriter) Write(p []byte) (n int, err error) {
	lw.mu.Lock()
	defer lw.mu.Unlock()

	// Write to original output
	n, err = lw.output.Write(p)

	// Accumulate bytes into line buffer
	lw.lineBuffer.Write(p)

	// Process complete lines
	content := lw.lineBuffer.String()
	lines := strings.Split(content, "\n")

	// Process all complete lines (all but the last if it doesn't end with newline)
	for i := 0; i < len(lines)-1; i++ {
		line := strings.TrimSpace(lines[i])
		if line != "" {
			lw.processLine(line)
		}
	}

	// Keep incomplete line in buffer
	if len(lines) > 0 {
		lw.lineBuffer.Reset()
		lw.lineBuffer.WriteString(lines[len(lines)-1])
	}

	return n, err
}

// processLine parses a log line and adds it to the buffer
func (lw *LogWriter) processLine(line string) {
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     LevelInfo,
		Source:    "app",
		Message:   line,
	}

	// Try to parse Go's default log format: 2006/01/02 15:04:05 message
	if len(line) > 20 && (line[4] == '/' || line[2] == ':') {
		// Try parsing timestamp
		if t, rest, ok := parseTimestamp(line); ok {
			entry.Timestamp = t
			entry.Message = rest
		}
	}

	// Detect log level
	for level, pattern := range levelPatterns {
		if pattern.MatchString(entry.Message) {
			entry.Level = level
			break
		}
	}

	// Detect source
	for source, pattern := range sourcePatterns {
		if pattern.MatchString(entry.Message) {
			entry.Source = source
			break
		}
	}

	lw.buffer.Add(entry)
}

// parseTimestamp attempts to parse a timestamp from the beginning of a log line
func parseTimestamp(line string) (time.Time, string, bool) {
	// Try Go's default log format: 2006/01/02 15:04:05
	if len(line) >= 19 {
		formats := []string{
			"2006/01/02 15:04:05",
			"2006-01-02 15:04:05",
			"15:04:05",
		}

		for _, format := range formats {
			if t, err := time.Parse(format, line[:len(format)]); err == nil {
				rest := strings.TrimSpace(line[len(format):])
				// If we only got time, use today's date
				if format == "15:04:05" {
					now := time.Now()
					t = time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), t.Second(), 0, time.Local)
				}
				return t, rest, true
			}
		}
	}

	return time.Time{}, "", false
}

// LogQuery represents query parameters for filtering logs
type LogQuery struct {
	Level  string // Minimum level (DEBUG shows all)
	Source string // Filter by source
	Search string // Text search
	Limit  int    // Max entries to return
}

// LogsResponse represents the API response for logs
type LogsResponse struct {
	Entries []LogEntry `json:"entries"`
	Total   int        `json:"total"`
	HasMore bool       `json:"hasMore"`
}

// Query filters and returns log entries based on the query parameters
func Query(q LogQuery) LogsResponse {
	if globalBuffer == nil {
		return LogsResponse{Entries: []LogEntry{}, Total: 0, HasMore: false}
	}

	entries := globalBuffer.GetAll()
	filtered := make([]LogEntry, 0, len(entries))

	minLevel := getLevelPriority(LogLevel(strings.ToUpper(q.Level)))
	searchLower := strings.ToLower(q.Search)

	for _, entry := range entries {
		// Filter by level
		if getLevelPriority(entry.Level) < minLevel {
			continue
		}

		// Filter by source
		if q.Source != "" && q.Source != "all" && !strings.EqualFold(entry.Source, q.Source) {
			continue
		}

		// Filter by search text
		if q.Search != "" && !strings.Contains(strings.ToLower(entry.Message), searchLower) {
			continue
		}

		filtered = append(filtered, entry)
	}

	total := len(filtered)
	hasMore := false

	if q.Limit > 0 && len(filtered) > q.Limit {
		// Return newest entries (last N)
		filtered = filtered[len(filtered)-q.Limit:]
		hasMore = true
	}

	return LogsResponse{
		Entries: filtered,
		Total:   total,
		HasMore: hasMore,
	}
}

// getLevelPriority returns a numeric priority for log levels
func getLevelPriority(level LogLevel) int {
	switch level {
	case LevelDebug:
		return 0
	case LevelInfo:
		return 1
	case LevelWarn:
		return 2
	case LevelError:
		return 3
	default:
		return 1
	}
}

// ExportAll returns all logs as a formatted string for download
func ExportAll() string {
	if globalBuffer == nil {
		return ""
	}

	entries := globalBuffer.GetAll()
	var sb strings.Builder

	sb.WriteString("Outpost Logs Export\n")
	sb.WriteString(fmt.Sprintf("Generated: %s\n", time.Now().Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("Total entries: %d\n", len(entries)))
	sb.WriteString(strings.Repeat("=", 80) + "\n\n")

	for _, entry := range entries {
		sb.WriteString(fmt.Sprintf("[%s] [%s] [%s] %s\n",
			entry.Timestamp.Format("2006-01-02 15:04:05"),
			entry.Level,
			entry.Source,
			entry.Message,
		))
	}

	return sb.String()
}

// LogWithSource creates a log message with an explicit source tag
func LogWithSource(source, format string, args ...interface{}) string {
	msg := fmt.Sprintf(format, args...)
	return fmt.Sprintf("[%s] %s", source, msg)
}

// Helper functions for tagged logging
func Scheduler(format string, args ...interface{}) string {
	return LogWithSource("scheduler", format, args...)
}

func Indexer(format string, args ...interface{}) string {
	return LogWithSource("indexer", format, args...)
}

func Importer(format string, args ...interface{}) string {
	return LogWithSource("importer", format, args...)
}

func Download(format string, args ...interface{}) string {
	return LogWithSource("download", format, args...)
}

func Scanner(format string, args ...interface{}) string {
	return LogWithSource("scanner", format, args...)
}

func Metadata(format string, args ...interface{}) string {
	return LogWithSource("metadata", format, args...)
}

func Auth(format string, args ...interface{}) string {
	return LogWithSource("auth", format, args...)
}

func API(format string, args ...interface{}) string {
	return LogWithSource("api", format, args...)
}

// MarshalJSON for LogEntry to format timestamp as ISO string
func (e LogEntry) MarshalJSON() ([]byte, error) {
	type Alias LogEntry
	return json.Marshal(&struct {
		Timestamp string `json:"timestamp"`
		*Alias
	}{
		Timestamp: e.Timestamp.Format(time.RFC3339),
		Alias:     (*Alias)(&e),
	})
}

// ParseLogFile reads and parses a log file into the buffer
func ParseLogFile(filename string) error {
	if globalBuffer == nil {
		return fmt.Errorf("logging not initialized")
	}

	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			entry := LogEntry{
				Timestamp: time.Now(),
				Level:     LevelInfo,
				Source:    "app",
				Message:   line,
			}

			// Try to parse timestamp and detect level/source
			if t, rest, ok := parseTimestamp(line); ok {
				entry.Timestamp = t
				entry.Message = rest
			}

			for level, pattern := range levelPatterns {
				if pattern.MatchString(entry.Message) {
					entry.Level = level
					break
				}
			}

			for source, pattern := range sourcePatterns {
				if pattern.MatchString(entry.Message) {
					entry.Source = source
					break
				}
			}

			globalBuffer.Add(entry)
		}
	}

	return scanner.Err()
}
