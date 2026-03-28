package repository

import (
	"context"
	"regexp"
	"strconv"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestListAlertEvents_DoesNotBindLimitAsArg(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer func() { _ = db.Close() }()

	repo := &opsRepository{db: db}
	start := time.Date(2026, 3, 28, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)

	filter := &service.OpsAlertEventFilter{
		Limit:     10,
		StartTime: &start,
		EndTime:   &end,
	}

	rows := sqlmock.NewRows([]string{
		"id",
		"rule_id",
		"severity",
		"status",
		"title",
		"description",
		"metric_value",
		"threshold_value",
		"dimensions",
		"fired_at",
		"resolved_at",
		"email_sent",
		"created_at",
	}).AddRow(
		int64(1),
		int64(2),
		"high",
		"firing",
		"High error rate",
		"Error rate exceeded threshold",
		1.25,
		1.00,
		[]byte(`{"platform":"openai"}`),
		start,
		nil,
		false,
		start,
	)

	queryPattern := `(?s)FROM ops_alert_events.*fired_at >= \$1.*fired_at < \$2.*ORDER BY fired_at DESC, id DESC.*LIMIT 10`
	mock.ExpectQuery(queryPattern).
		WithArgs(start, end).
		WillReturnRows(rows)

	out, err := repo.ListAlertEvents(context.Background(), filter)
	if err != nil {
		t.Fatalf("ListAlertEvents error: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("len(out) = %d, want 1", len(out))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}

func TestBuildOpsAlertEventsWhere_MaxPlaceholderMatchesArgsLen(t *testing.T) {
	start := time.Date(2026, 3, 27, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	before := end.Add(-time.Hour)
	beforeID := int64(99)
	groupID := int64(7)
	emailSent := true

	filter := &service.OpsAlertEventFilter{
		Status:        "firing",
		Severity:      "high",
		EmailSent:     &emailSent,
		StartTime:     &start,
		EndTime:       &end,
		BeforeFiredAt: &before,
		BeforeID:      &beforeID,
		Platform:      "openai",
		GroupID:       &groupID,
	}

	where, args := buildOpsAlertEventsWhere(filter)
	if len(args) == 0 {
		t.Fatalf("args should not be empty")
	}

	re := regexp.MustCompile(`\$(\d+)`)
	matches := re.FindAllStringSubmatch(where, -1)
	if len(matches) == 0 {
		t.Fatalf("expected placeholders in where clause: %s", where)
	}

	maxPos := 0
	for _, m := range matches {
		v, err := strconv.Atoi(m[1])
		if err != nil {
			t.Fatalf("invalid placeholder index %q: %v", m[1], err)
		}
		if v > maxPos {
			maxPos = v
		}
	}

	if maxPos != len(args) {
		t.Fatalf("max placeholder = %d, args len = %d, where = %s", maxPos, len(args), where)
	}
}
