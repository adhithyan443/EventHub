package repository

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
)

func TestSystemLogRepository_BasicOperations(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test_app.log")

	sampleLogs := `{"time":"2026-11-26T10:00:00Z","level":"INFO","msg":"server_started","port":8080}
{"time":"2026-11-26T10:05:00Z","level":"WARN","msg":"cache_miss","key":"user_123"}
{"time":"2026-11-26T10:10:00Z","level":"ERROR","msg":"payment_failed","request_id":"req-999","token":"secret-token-123","error":"gateway timeout"}
{"time":"2026-11-26T10:15:00Z","level":"INFO","msg":"booking_confirmed","request_id":"req-1000","amount":500}
`
	if err := os.WriteFile(logFile, []byte(sampleLogs), 0644); err != nil {
		t.Fatalf("failed to write test log file: %v", err)
	}

	repo := NewSystemMonitoringRepository(logFile, nil)
	ctx := context.Background()

	// 1. Fetch all (Page 1, Limit 10) - newest first
	res, err := repo.GetLogs(ctx, domain.SystemLogFilter{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("GetLogs failed: %v", err)
	}
	if res.TotalRecords != 4 {
		t.Errorf("expected 4 total records, got %d", res.TotalRecords)
	}
	if len(res.Logs) != 4 {
		t.Errorf("expected 4 logs, got %d", len(res.Logs))
	}
	// Newest should be first
	if res.Logs[0].Message != "booking_confirmed" {
		t.Errorf("expected first log to be 'booking_confirmed', got '%s'", res.Logs[0].Message)
	}
	if res.Logs[3].Message != "server_started" {
		t.Errorf("expected last log to be 'server_started', got '%s'", res.Logs[3].Message)
	}

	// 2. Filter by Level = ERROR
	errRes, err := repo.GetLogs(ctx, domain.SystemLogFilter{Level: "ERROR", Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("GetLogs with ERROR filter failed: %v", err)
	}
	if errRes.TotalRecords != 1 {
		t.Errorf("expected 1 ERROR record, got %d", errRes.TotalRecords)
	}
	if errRes.Logs[0].Message != "payment_failed" {
		t.Errorf("expected 'payment_failed', got '%s'", errRes.Logs[0].Message)
	}
	if errRes.Logs[0].Fields["token"] != "[REDACTED]" {
		t.Errorf("expected token to be [REDACTED], got %v", errRes.Logs[0].Fields["token"])
	}

	// 3. Search filter by Request ID
	searchRes, err := repo.GetLogs(ctx, domain.SystemLogFilter{Search: "req-1000", Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("GetLogs with search failed: %v", err)
	}
	if searchRes.TotalRecords != 1 {
		t.Errorf("expected 1 search match, got %d", searchRes.TotalRecords)
	}
	if searchRes.Logs[0].RequestID != "req-1000" {
		t.Errorf("expected request_id req-1000, got %s", searchRes.Logs[0].RequestID)
	}

	// 4. Nonexistent file returns empty result cleanly
	nonExistentRepo := NewSystemMonitoringRepository("non_existent_path.log", nil)
	emptyRes, err := nonExistentRepo.GetLogs(ctx, domain.SystemLogFilter{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("expected no error for missing log file, got: %v", err)
	}
	if emptyRes.TotalRecords != 0 || len(emptyRes.Logs) != 0 {
		t.Errorf("expected empty result, got %d records", emptyRes.TotalRecords)
	}
}

// Regression Test: File with more than 20,000 log entries retains the newest entries
func TestSystemLogRepository_ExceedsCapacity_RetainsNewest(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "large_app.log")

	file, err := os.Create(logFile)
	if err != nil {
		t.Fatalf("failed to create large log file: %v", err)
	}

	totalToWrite := 21000
	for i := 1; i <= totalToWrite; i++ {
		line := fmt.Sprintf(`{"time":"2026-11-26T10:00:00Z","level":"INFO","msg":"entry_%d","id":%d}`+"\n", i, i)
		if _, err := file.WriteString(line); err != nil {
			file.Close()
			t.Fatalf("failed to write line %d: %v", i, err)
		}
	}
	file.Close()

	repo := NewSystemMonitoringRepository(logFile, nil)
	ctx := context.Background()

	res, err := repo.GetLogs(ctx, domain.SystemLogFilter{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("GetLogs on large file failed: %v", err)
	}

	// Capacity is capped at 20,000
	if res.TotalRecords != 20000 {
		t.Errorf("expected TotalRecords to be capped at 20000, got %d", res.TotalRecords)
	}

	if len(res.Logs) != 10 {
		t.Fatalf("expected 10 logs on page 1, got %d", len(res.Logs))
	}

	// Newest entry must be entry_21000 (not entry_20000 from old truncation!)
	expectedNewest := fmt.Sprintf("entry_%d", totalToWrite)
	if res.Logs[0].Message != expectedNewest {
		t.Errorf("expected newest entry to be '%s', got '%s'", expectedNewest, res.Logs[0].Message)
	}

	// Second newest must be entry_20999
	expectedSecond := fmt.Sprintf("entry_%d", totalToWrite-1)
	if res.Logs[1].Message != expectedSecond {
		t.Errorf("expected second newest entry to be '%s', got '%s'", expectedSecond, res.Logs[1].Message)
	}

	// Check pagination near the end of the retained buffer (e.g. page 2000 with limit 10)
	lastPageRes, err := repo.GetLogs(ctx, domain.SystemLogFilter{Page: 2000, Limit: 10})
	if err != nil {
		t.Fatalf("failed to fetch last page: %v", err)
	}
	if len(lastPageRes.Logs) != 10 {
		t.Fatalf("expected 10 logs on page 2000, got %d", len(lastPageRes.Logs))
	}
	// The oldest retained log should be entry_1001 (since 21000 - 20000 + 1 = 1001)
	expectedOldest := "entry_1001"
	if lastPageRes.Logs[9].Message != expectedOldest {
		t.Errorf("expected oldest retained log on page 2000 to be '%s', got '%s'", expectedOldest, lastPageRes.Logs[9].Message)
	}
}

// Regression Test: Filtering while log file exceeds capacity
func TestSystemLogRepository_ExceedsCapacity_WithFilter(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "mixed_app.log")

	file, err := os.Create(logFile)
	if err != nil {
		t.Fatalf("failed to create mixed log file: %v", err)
	}

	// Write 25,000 lines total: every 10th line is ERROR, rest are INFO
	for i := 1; i <= 25000; i++ {
		lvl := "INFO"
		if i%10 == 0 {
			lvl = "ERROR"
		}
		line := fmt.Sprintf(`{"time":"2026-11-26T10:00:00Z","level":"%s","msg":"log_%d"}`+"\n", lvl, i)
		if _, err := file.WriteString(line); err != nil {
			file.Close()
			t.Fatalf("failed to write line %d: %v", i, err)
		}
	}
	file.Close()

	repo := NewSystemMonitoringRepository(logFile, nil)
	ctx := context.Background()

	// Filter by ERROR (there are 2500 ERROR lines total)
	res, err := repo.GetLogs(ctx, domain.SystemLogFilter{Level: "ERROR", Page: 1, Limit: 5})
	if err != nil {
		t.Fatalf("GetLogs with ERROR filter failed: %v", err)
	}

	if res.TotalRecords != 2500 {
		t.Errorf("expected 2500 matching ERROR records, got %d", res.TotalRecords)
	}

	// The newest ERROR is log_25000
	if res.Logs[0].Message != "log_25000" {
		t.Errorf("expected newest ERROR log to be 'log_25000', got '%s'", res.Logs[0].Message)
	}
	if res.Logs[1].Message != "log_24990" {
		t.Errorf("expected second newest ERROR log to be 'log_24990', got '%s'", res.Logs[1].Message)
	}
}

// Regression Test: Recursive secret sanitization for nested objects and arrays
func TestSystemLogRepository_RecursiveSanitization(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "secrets_app.log")

	nestedJSON := `{"time":"2026-11-26T12:00:00Z","level":"INFO","msg":"user_action","user":{"name":"Alice","password":"plainPassword123","profile":{"credit_card":"4111-2222-3333-4444","age":30,"active":true}},"headers":{"Authorization":"Bearer secretTokenXYZ","cookie":"session=abc"},"items":[{"id":1,"secret_key":"sk_live_9999"},{"id":2,"name":"Book"}]}` + "\n"

	if err := os.WriteFile(logFile, []byte(nestedJSON), 0644); err != nil {
		t.Fatalf("failed to write secrets log file: %v", err)
	}

	repo := NewSystemMonitoringRepository(logFile, nil)
	ctx := context.Background()

	res, err := repo.GetLogs(ctx, domain.SystemLogFilter{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("GetLogs failed: %v", err)
	}
	if len(res.Logs) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(res.Logs))
	}

	log := res.Logs[0]

	// 1. Nested user.password must be [REDACTED]
	userMap, ok := log.Fields["user"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected user to be map[string]interface{}, got %T", log.Fields["user"])
	}
	if userMap["password"] != "[REDACTED]" {
		t.Errorf("expected user.password to be [REDACTED], got: %v", userMap["password"])
	}

	// 2. Non-sensitive user.name must be preserved
	if userMap["name"] != "Alice" {
		t.Errorf("expected user.name to be 'Alice', got: %v", userMap["name"])
	}

	// 3. Deeply nested profile.credit_card must be [REDACTED]
	profileMap, ok := userMap["profile"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected profile to be map[string]interface{}, got %T", userMap["profile"])
	}
	if profileMap["credit_card"] != "[REDACTED]" {
		t.Errorf("expected profile.credit_card to be [REDACTED], got: %v", profileMap["credit_card"])
	}

	// 4. Non-sensitive nested types (numbers, booleans) must be preserved
	if profileMap["age"] != float64(30) {
		t.Errorf("expected profile.age to be 30, got: %v (%T)", profileMap["age"], profileMap["age"])
	}
	if profileMap["active"] != true {
		t.Errorf("expected profile.active to be true, got: %v", profileMap["active"])
	}

	// 5. Nested headers.Authorization and headers.cookie must be [REDACTED]
	headersMap, ok := log.Fields["headers"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected headers to be map[string]interface{}, got %T", log.Fields["headers"])
	}
	if headersMap["Authorization"] != "[REDACTED]" {
		t.Errorf("expected headers.Authorization to be [REDACTED], got: %v", headersMap["Authorization"])
	}
	if headersMap["cookie"] != "[REDACTED]" {
		t.Errorf("expected headers.cookie to be [REDACTED], got: %v", headersMap["cookie"])
	}

	// 6. Slices with objects: items[0].secret_key must be [REDACTED], items[1].name preserved
	itemsSlice, ok := log.Fields["items"].([]interface{})
	if !ok {
		t.Fatalf("expected items to be []interface{}, got %T", log.Fields["items"])
	}
	item0 := itemsSlice[0].(map[string]interface{})
	if item0["secret_key"] != "[REDACTED]" {
		t.Errorf("expected item 0 secret_key to be [REDACTED], got: %v", item0["secret_key"])
	}
	item1 := itemsSlice[1].(map[string]interface{})
	if item1["name"] != "Book" {
		t.Errorf("expected item 1 name to be 'Book', got: %v", item1["name"])
	}
}

// Regression Test: JWT sanitization for both Bearer-prefixed and standalone JWTs
func TestSystemLogRepository_JWTSanitization(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "jwt_app.log")

	standaloneJWT := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
	bearerJWT := "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyIjoiMTIzIn0.xyzSignaturePart123"

	logs := fmt.Sprintf(
		`{"time":"2026-11-26T12:00:00Z","level":"INFO","msg":"User token is %s and bearer is %s"}`+"\n"+
			`{"time":"2026-11-26T12:01:00Z","level":"INFO","msg":"Normal message with word eye or key.word.here"}`+"\n",
		standaloneJWT, bearerJWT,
	)

	if err := os.WriteFile(logFile, []byte(logs), 0644); err != nil {
		t.Fatalf("failed to write jwt log file: %v", err)
	}

	repo := NewSystemMonitoringRepository(logFile, nil)
	ctx := context.Background()

	res, err := repo.GetLogs(ctx, domain.SystemLogFilter{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("GetLogs failed: %v", err)
	}

	jwtLog := res.Logs[1] // Order is newest first, so index 1 is first log
	if strings.Contains(jwtLog.Message, standaloneJWT) {
		t.Errorf("standalone JWT was not redacted: %s", jwtLog.Message)
	}
	if strings.Contains(jwtLog.Message, "xyzSignaturePart123") {
		t.Errorf("bearer JWT was not redacted: %s", jwtLog.Message)
	}
	if !strings.Contains(jwtLog.Message, "[REDACTED]") {
		t.Errorf("expected [REDACTED] in message, got: %s", jwtLog.Message)
	}

	// Normal non-JWT text must be preserved
	normalLog := res.Logs[0]
	if normalLog.Message != "Normal message with word eye or key.word.here" {
		t.Errorf("normal message was unexpectedly altered: %s", normalLog.Message)
	}
}

// Regression Test: Oversized log line does not prevent subsequent lines from being read
func TestSystemLogRepository_OversizedLogLineHandling(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "oversized_app.log")

	file, err := os.Create(logFile)
	if err != nil {
		t.Fatalf("failed to create oversized log file: %v", err)
	}

	// Line 1: Normal line
	file.WriteString(`{"time":"2026-11-26T10:00:00Z","level":"INFO","msg":"before_oversized"}` + "\n")

	// Line 2: Giant 2MB line
	giantPayload := strings.Repeat("A", 2*1024*1024)
	file.WriteString(fmt.Sprintf(`{"time":"2026-11-26T10:01:00Z","level":"INFO","msg":"oversized","blob":"%s"}`+"\n", giantPayload))

	// Line 3: Normal line AFTER the giant line
	file.WriteString(`{"time":"2026-11-26T10:02:00Z","level":"INFO","msg":"after_oversized"}` + "\n")
	file.Close()

	repo := NewSystemMonitoringRepository(logFile, nil)
	ctx := context.Background()

	res, err := repo.GetLogs(ctx, domain.SystemLogFilter{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("GetLogs failed on file with oversized line: %v", err)
	}

	// Verify that the line AFTER the oversized line was successfully read!
	// Newest first order: line 3 ("after_oversized"), line 2 (truncated oversized), line 1 ("before_oversized")
	foundAfter := false
	foundBefore := false
	for _, l := range res.Logs {
		if l.Message == "after_oversized" {
			foundAfter = true
		}
		if l.Message == "before_oversized" {
			foundBefore = true
		}
	}

	if !foundAfter {
		t.Errorf("line after oversized line was not read; reader aborted prematurely")
	}
	if !foundBefore {
		t.Errorf("line before oversized line was not read")
	}
}
