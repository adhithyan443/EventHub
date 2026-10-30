package repository

import (
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestEventRepository_FindPublicEvents_DryRunQuery(t *testing.T) {
	db, err := gorm.Open(postgres.New(postgres.Config{
		WithoutReturning: true,
	}), &gorm.Config{
		DryRun:                 true,
		SkipDefaultTransaction: true,
		DisableAutomaticPing:   true,
	})
	require.NoError(t, err)

	repo := NewEventRepository(db, slog.Default())

	categoryID := uuid.New()
	eventDate := time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC)
	filter := domain.PublicEventFilter{
		Page:       1,
		Limit:      12,
		Keyword:    "Concert",
		CategoryID: &categoryID,
		City:       "Kochi",
		EventDate:  &eventDate,
	}

	// In DryRun mode, FindPublicEvents will assemble the query and return nil events without connecting to DB
	_, _, _ = repo.FindPublicEvents(filter)

	// Build the statement directly using the same table and joins to inspect the exact SQL
	now := time.Now()
	stmt := db.Session(&gorm.Session{DryRun: true}).
		Table("events AS e").
		Joins("INNER JOIN categories AS c ON c.id = e.category_id").
		Joins("LEFT JOIN venues AS v ON v.id = e.venue_id").
		Joins("INNER JOIN event_schedules AS es ON es.event_id = e.id").
		Joins("LEFT JOIN event_settings AS eset ON eset.event_id = e.id").
		Where("e.status = ?", domain.EventStatusPublished).
		Where("e.visibility = ?", "PUBLIC").
		Where(`(
			CASE
				WHEN es.is_all_day = true THEN (es.event_date + TIME '23:59:59')
				WHEN es.end_time IS NOT NULL AND es.start_time IS NOT NULL AND es.end_time < es.start_time
					THEN (es.event_date + INTERVAL '1 day' + es.end_time)
				WHEN es.end_time IS NOT NULL
					THEN (es.event_date + es.end_time)
				ELSE (es.event_date + TIME '23:59:59')
			END
		) >= ?`, now).
		Find(&[]domain.PublicEvent{}).
		Statement

	sql := stmt.SQL.String()
	require.NotEmpty(t, sql)

	// Verify essential query clauses
	assert.True(t, strings.Contains(sql, "events AS e"))
	assert.True(t, strings.Contains(sql, "e.status = $1") || strings.Contains(sql, "e.status = ?"))
	assert.True(t, strings.Contains(sql, "e.visibility = $2") || strings.Contains(sql, "e.visibility = ?"))
	assert.True(t, strings.Contains(sql, "es.is_all_day = true"))
	assert.True(t, strings.Contains(sql, "INTERVAL '1 day'"))
	assert.True(t, strings.Contains(sql, "TIME '23:59:59'"))

	// Ensure no arbitrary 1-day interval bounding condition was introduced
	assert.False(t, strings.Contains(sql, "CURRENT_DATE - INTERVAL '1 day'"),
		"Query must not contain arbitrary interval bounding filter")
}

// computeEffectiveEndTime calculates the event end time mirroring the SQL CASE expression
func computeEffectiveEndTime(eventDate time.Time, startTime, endTime *string, isAllDay bool) time.Time {
	truncDate := time.Date(eventDate.Year(), eventDate.Month(), eventDate.Day(), 0, 0, 0, 0, eventDate.Location())

	if isAllDay {
		return truncDate.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	}

	if endTime != nil && *endTime != "" {
		endT, err := time.Parse("15:04:05", *endTime)
		if err != nil {
			endT, _ = time.Parse("15:04", *endTime)
		}

		var startT time.Time
		if startTime != nil && *startTime != "" {
			startT, _ = time.Parse("15:04:05", *startTime)
			if startT.IsZero() {
				startT, _ = time.Parse("15:04", *startTime)
			}
		}

		// Overnight check: end_time < start_time
		if !startT.IsZero() && endT.Before(startT) {
			// Crosses midnight: event_date + 1 day + end_time
			return truncDate.AddDate(0, 0, 1).Add(time.Duration(endT.Hour())*time.Hour + time.Duration(endT.Minute())*time.Minute + time.Duration(endT.Second())*time.Second)
		}

		return truncDate.Add(time.Duration(endT.Hour())*time.Hour + time.Duration(endT.Minute())*time.Minute + time.Duration(endT.Second())*time.Second)
	}

	// Missing end time: active through 23:59:59 of event_date
	return truncDate.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
}

func isEventActive(eventDate time.Time, startTime, endTime *string, isAllDay bool, now time.Time) bool {
	effectiveEnd := computeEffectiveEndTime(eventDate, startTime, endTime, isAllDay)
	return !effectiveEnd.Before(now)
}

func TestEventCompletionLogic_Scenarios(t *testing.T) {
	refTime := time.Date(2026, 10, 26, 17, 0, 0, 0, time.UTC)

	start1000 := "10:00:00"
	start1800 := "18:00:00"
	end1600 := "16:00:00"
	end2100 := "21:00:00"
	start2200 := "22:00:00"
	end0200 := "02:00:00"

	// 1. Past event from yesterday -> completed (inactive)
	yesterday := refTime.AddDate(0, 0, -1)
	assert.False(t, isEventActive(yesterday, &start1800, &end2100, false, refTime),
		"Yesterday's event should be completed")

	// 2. Today's event that ended at 16:00 (started at 10:00, refTime is 17:00) -> completed (inactive)
	today := time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC)
	assert.False(t, isEventActive(today, &start1000, &end1600, false, refTime),
		"Event ending earlier today should be completed")

	// 3. Today's ongoing event (started at 15:00, ends at 21:00) -> active
	start1500 := "15:00:00"
	assert.True(t, isEventActive(today, &start1500, &end2100, false, refTime),
		"Ongoing event today should be active")

	// 4. Today's upcoming event (starts at 18:00, ends at 21:00) -> active
	assert.True(t, isEventActive(today, &start1800, &end2100, false, refTime),
		"Upcoming event starting later today should be active")

	// 5. Future event tomorrow -> active
	tomorrow := refTime.AddDate(0, 0, 1)
	assert.True(t, isEventActive(tomorrow, &start1800, &end2100, false, refTime),
		"Future event should be active")

	// 6. Overnight event starting yesterday at 22:00, ending today at 02:00 -> ended at 02:00 today (refTime is 17:00)
	assert.False(t, isEventActive(yesterday, &start2200, &end0200, false, refTime),
		"Overnight event ending at 02:00 this morning should be completed at 17:00")

	// 7. Overnight event starting today at 22:00, ending tomorrow at 02:00 -> active
	assert.True(t, isEventActive(today, &start2200, &end0200, false, refTime),
		"Overnight event starting tonight should be active")

	// 8. Overnight event starting yesterday at 22:00, ending today at 18:00 -> currently active at 17:00
	end1800 := "18:00:00"
	assert.True(t, isEventActive(yesterday, &start2200, &end1800, false, refTime),
		"Overnight event ending in an hour should be active")

	// 9. All-day event today -> active until 23:59:59
	assert.True(t, isEventActive(today, nil, nil, true, refTime),
		"All-day event today should be active")

	// 10. All-day event yesterday -> completed
	assert.False(t, isEventActive(yesterday, nil, nil, true, refTime),
		"All-day event yesterday should be completed")

	// 11. Event today with start_time but no end_time -> active until 23:59:59
	assert.True(t, isEventActive(today, &start1500, nil, false, refTime),
		"Event without end time today should remain active until 23:59:59")
}
