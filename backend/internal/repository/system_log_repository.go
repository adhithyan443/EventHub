package repository

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
)

var (
	bearerTokenRegex      = regexp.MustCompile(`(?i)Bearer\s+[A-Za-z0-9\-_=]+\.[A-Za-z0-9\-_=]+\.?[A-Za-z0-9\-_=]*`)
	standaloneJWTRegex    = regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]+\b`)
	passwordQuotedRegex   = regexp.MustCompile(`(?i)("?password"?\s*[:=]\s*")[^"]+(")`)
	passwordKeyValueRegex = regexp.MustCompile(`(?i)(password\s*[:=]\s*)([^\s,"'}{]+)`)
)

type systemLogRepository struct {
	logFilePath string
	logger      *slog.Logger
}

// NewSystemMonitoringRepository creates a new repository for reading application logs safely.
func NewSystemMonitoringRepository(logFilePath string, logger *slog.Logger) domain.SystemMonitoringRepository {
	if logger == nil {
		logger = slog.Default()
	}
	if strings.TrimSpace(logFilePath) == "" {
		logFilePath = "app.log"
	}
	return &systemLogRepository{
		logFilePath: logFilePath,
		logger:      logger,
	}
}

// resolveLogFilePath attempts to locate the log file safely without directory traversal.
func (r *systemLogRepository) resolveLogFilePath() (string, error) {
	cleanPath := filepath.Clean(r.logFilePath)

	// If absolute or exists in current dir
	if _, err := os.Stat(cleanPath); err == nil {
		return cleanPath, nil
	}

	// Try in backend/ subdirectory (if running from root)
	backendPath := filepath.Clean(filepath.Join("backend", r.logFilePath))
	if _, err := os.Stat(backendPath); err == nil {
		return backendPath, nil
	}

	return cleanPath, os.ErrNotExist
}

// readNextLine reads a single line from a bufio.Reader up to maxLineLen bytes.
// If a line exceeds maxLineLen, it consumes and discards the remainder of that line
// until the newline delimiter, allowing subsequent lines to be read cleanly.
func readNextLine(reader *bufio.Reader, maxLineLen int) ([]byte, error) {
	var lineBuf []byte
	isFirstChunk := true

	for {
		chunk, isPrefix, err := reader.ReadLine()
		if err != nil {
			if len(lineBuf) > 0 && errors.Is(err, io.EOF) {
				return lineBuf, nil
			}
			return nil, err
		}

		if isFirstChunk {
			if len(chunk) <= maxLineLen {
				lineBuf = append(lineBuf, chunk...)
			} else {
				lineBuf = append(lineBuf, chunk[:maxLineLen]...)
			}
		} else if len(lineBuf) < maxLineLen {
			spaceLeft := maxLineLen - len(lineBuf)
			if len(chunk) <= spaceLeft {
				lineBuf = append(lineBuf, chunk...)
			} else {
				lineBuf = append(lineBuf, chunk[:spaceLeft]...)
			}
		}

		isFirstChunk = false

		if !isPrefix {
			break
		}
	}

	return lineBuf, nil
}

// GetLogs scans the application log file line by line, applying level and search filters,
// and returns the newest matching entries up to 20,000 using a bounded ring buffer.
func (r *systemLogRepository) GetLogs(
	ctx context.Context,
	filter domain.SystemLogFilter,
) (*domain.SystemLogListResult, error) {
	resolvedPath, err := r.resolveLogFilePath()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			r.logger.WarnContext(ctx, "log file not found, returning empty list", slog.String("path", r.logFilePath))
			return &domain.SystemLogListResult{
				Logs:         make([]domain.SystemLogEntry, 0),
				TotalRecords: 0,
				Page:         filter.Page,
				Limit:        filter.Limit,
				TotalPages:   0,
			}, nil
		}
		r.logger.ErrorContext(ctx, "failed to check log file path", slog.Any("error", err))
		return nil, err
	}

	file, err := os.Open(resolvedPath)
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to open log file", slog.String("path", resolvedPath), slog.Any("error", err))
		return nil, err
	}
	defer file.Close()

	// Normalize filter parameters
	targetLevel := strings.ToUpper(strings.TrimSpace(filter.Level))
	searchPattern := strings.ToLower(strings.TrimSpace(filter.Search))

	page := filter.Page
	if page < 1 {
		page = 1
	}
	limit := filter.Limit
	if limit < 1 {
		limit = 50
	} else if limit > 200 {
		limit = 200
	}

	// Use bounded ring buffer to retain the newest 20,000 matching entries without memory growth
	const maxCap = 20000
	ring := make([]domain.SystemLogEntry, maxCap)
	var totalMatched int64

	reader := bufio.NewReaderSize(file, 64*1024)
	const maxLineLen = 1024 * 1024 // 1MB line limit

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		line, readErr := readNextLine(reader, maxLineLen)
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				break
			}
			r.logger.WarnContext(ctx, "error reading line from log file", slog.Any("error", readErr))
			break
		}

		if len(line) == 0 {
			continue
		}

		entry, ok := parseAndFilterLogLine(line, targetLevel, searchPattern)
		if !ok {
			continue
		}

		ring[totalMatched%maxCap] = entry
		totalMatched++
	}

	retainedCount := totalMatched
	if retainedCount > maxCap {
		retainedCount = maxCap
	}

	totalPages := 0
	if retainedCount > 0 {
		totalPages = int(math.Ceil(float64(retainedCount) / float64(limit)))
	}

	if retainedCount == 0 {
		return &domain.SystemLogListResult{
			Logs:         make([]domain.SystemLogEntry, 0),
			TotalRecords: 0,
			Page:         page,
			Limit:        limit,
			TotalPages:   0,
		}, nil
	}

	offset := (page - 1) * limit
	if offset >= int(retainedCount) {
		return &domain.SystemLogListResult{
			Logs:         make([]domain.SystemLogEntry, 0),
			TotalRecords: retainedCount,
			Page:         page,
			Limit:        limit,
			TotalPages:   totalPages,
		}, nil
	}

	end := offset + limit
	if end > int(retainedCount) {
		end = int(retainedCount)
	}

	// Extract newest-first slice directly from ring buffer without extra 20,000-element allocations
	resultCount := end - offset
	paginatedLogs := make([]domain.SystemLogEntry, resultCount)
	lastIdx := (totalMatched - 1) % maxCap

	for i := 0; i < resultCount; i++ {
		globalRank := int64(offset + i)
		ringIdx := (lastIdx - (globalRank % maxCap) + maxCap) % maxCap
		paginatedLogs[i] = ring[ringIdx]
	}

	return &domain.SystemLogListResult{
		Logs:         paginatedLogs,
		TotalRecords: retainedCount,
		Page:         page,
		Limit:        limit,
		TotalPages:   totalPages,
	}, nil
}

// parseAndFilterLogLine parses a raw JSON or text line, sanitizes secrets,
// and applies level and search filters.
func parseAndFilterLogLine(line []byte, targetLevel, searchPattern string) (domain.SystemLogEntry, bool) {
	var rawMap map[string]interface{}
	if err := json.Unmarshal(line, &rawMap); err != nil {
		// Non-JSON fallback line
		rawText := string(line)
		if targetLevel != "" && targetLevel != "ALL" {
			if !strings.Contains(strings.ToUpper(rawText), targetLevel) {
				return domain.SystemLogEntry{}, false
			}
		}
		if searchPattern != "" && !strings.Contains(strings.ToLower(rawText), searchPattern) {
			return domain.SystemLogEntry{}, false
		}

		sanitizedMsg := sanitizeString(rawText)
		return domain.SystemLogEntry{
			Timestamp: time.Now(),
			Level:     "INFO",
			Message:   sanitizedMsg,
		}, true
	}

	// Extract standard slog attributes
	var timestamp time.Time
	if timeVal, ok := rawMap["time"].(string); ok {
		if t, err := time.Parse(time.RFC3339Nano, timeVal); err == nil {
			timestamp = t
		} else if t, err := time.Parse(time.RFC3339, timeVal); err == nil {
			timestamp = t
		}
	}
	if timestamp.IsZero() {
		timestamp = time.Now()
	}

	level := "INFO"
	if lvlVal, ok := rawMap["level"].(string); ok && lvlVal != "" {
		level = strings.ToUpper(lvlVal)
	}

	msg := ""
	if msgVal, ok := rawMap["msg"].(string); ok {
		msg = msgVal
	} else if msgVal, ok := rawMap["message"].(string); ok {
		msg = msgVal
	}

	// Level filter check
	if targetLevel != "" && targetLevel != "ALL" {
		if level != targetLevel {
			return domain.SystemLogEntry{}, false
		}
	}

	requestID := ""
	if reqVal, ok := rawMap["request_id"].(string); ok {
		requestID = reqVal
	}

	// Extract extra attributes into Fields map, filtering out standard slog keys
	fields := make(map[string]interface{})
	for k, v := range rawMap {
		switch k {
		case "time", "level", "msg", "message":
			continue
		case "request_id":
			continue
		default:
			fields[k] = v
		}
	}

	// Recursively sanitize sensitive fields (tokens, passwords, secrets, authorization)
	sanitizedFields := sanitizeLogFields(fields)
	sanitizedMsg := sanitizeString(msg)

	// Search filter check
	if searchPattern != "" {
		match := false
		if strings.Contains(strings.ToLower(sanitizedMsg), searchPattern) {
			match = true
		} else if strings.Contains(strings.ToLower(level), searchPattern) {
			match = true
		} else if strings.Contains(strings.ToLower(requestID), searchPattern) {
			match = true
		} else {
			// Check fields values
			for _, v := range sanitizedFields {
				valStr, _ := json.Marshal(v)
				if strings.Contains(strings.ToLower(string(valStr)), searchPattern) {
					match = true
					break
				}
			}
		}
		if !match {
			return domain.SystemLogEntry{}, false
		}
	}

	return domain.SystemLogEntry{
		Timestamp: timestamp,
		Level:     level,
		Message:   sanitizedMsg,
		RequestID: requestID,
		Fields:    sanitizedFields,
	}, true
}

var sensitiveKeySubstrings = []string{
	"password",
	"token",
	"secret",
	"authorization",
	"cookie",
	"credit_card",
	"card_number",
	"cvv",
}

// isSensitiveKey checks whether a field key represents a sensitive credential.
func isSensitiveKey(key string) bool {
	lowerKey := strings.ToLower(key)
	for _, sens := range sensitiveKeySubstrings {
		if strings.Contains(lowerKey, sens) {
			return true
		}
	}
	return false
}

// sanitizeValue recursively inspects and redacts sensitive keys and values across
// nested maps, slices, and strings, while preserving non-sensitive values and original types.
func sanitizeValue(key string, v interface{}) interface{} {
	if isSensitiveKey(key) {
		return "[REDACTED]"
	}

	switch val := v.(type) {
	case map[string]interface{}:
		sanitizedMap := make(map[string]interface{}, len(val))
		for childK, childV := range val {
			sanitizedMap[childK] = sanitizeValue(childK, childV)
		}
		return sanitizedMap
	case []interface{}:
		sanitizedSlice := make([]interface{}, len(val))
		for i, elem := range val {
			sanitizedSlice[i] = sanitizeValue(key, elem)
		}
		return sanitizedSlice
	case string:
		return sanitizeString(val)
	default:
		return v
	}
}

// sanitizeLogFields recursively sanitizes all keys and values in the fields map.
func sanitizeLogFields(fields map[string]interface{}) map[string]interface{} {
	if len(fields) == 0 {
		return fields
	}

	sanitized := make(map[string]interface{}, len(fields))
	for k, v := range fields {
		sanitized[k] = sanitizeValue(k, v)
	}

	return sanitized
}

// sanitizeString masks Bearer tokens, standalone JWTs, and passwords in text strings.
func sanitizeString(s string) string {
	// Redact Bearer-prefixed tokens
	res := bearerTokenRegex.ReplaceAllString(s, "Bearer [REDACTED]")

	// Redact standalone JWT tokens
	res = standaloneJWTRegex.ReplaceAllString(res, "[REDACTED]")

	// Redact password assignments
	res = passwordQuotedRegex.ReplaceAllString(res, `${1}[REDACTED]${2}`)
	res = passwordKeyValueRegex.ReplaceAllString(res, `${1}[REDACTED]`)

	return res
}
