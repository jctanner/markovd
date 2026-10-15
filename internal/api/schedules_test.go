package api

import (
	"testing"
	"time"
)

func TestScheduleTimes(t *testing.T) {
	now := time.Date(2026, 10, 15, 3, 0, 0, 0, time.UTC)
	at := time.Date(2026, 10, 15, 0, 30, 0, 0, time.FixedZone("EDT", -4*3600))
	got, err := nextScheduleTime("once", "", "America/New_York", &at, now)
	if err != nil || got.Format(time.RFC3339) != "2026-10-15T04:30:00Z" {
		t.Fatalf("midnight padding: %v %v", got, err)
	}
	for _, tc := range []struct {
		kind, cron, zone string
		at               *time.Time
	}{
		{"once", "", "UTC", nil}, {"once", "", "UTC", &now}, {"once", "* * * * *", "UTC", &at},
		{"cron", "* * * * *", "UTC", &at}, {"cron", "* * * * *", "Bogus/Zone", nil},
		{"cron", "0 0 0 * * *", "UTC", nil}, {"cron", "@every 1s", "UTC", nil},
		{"cron", "0 0 31 2 *", "UTC", nil}, {"bad", "", "UTC", nil},
	} {
		if _, err := nextScheduleTime(tc.kind, tc.cron, tc.zone, tc.at, now); err == nil {
			t.Errorf("accepted invalid schedule %+v", tc)
		}
	}
}

func TestCronDSTAndCoalescing(t *testing.T) {
	for _, tc := range []struct{ now, want string }{
		{"2026-10-15T03:00:00Z", "2026-10-15T04:30:00Z"},
		{"2026-11-02T03:00:00Z", "2026-11-02T05:30:00Z"},
		{"2026-10-18T13:45:00Z", "2026-10-19T04:30:00Z"},
	} {
		now, _ := time.Parse(time.RFC3339, tc.now)
		got, err := nextScheduleTime("cron", "30 0 * * *", "America/New_York", nil, now)
		if err != nil || got.Format(time.RFC3339) != tc.want {
			t.Errorf("%s: %v %v want %s", tc.now, got, err, tc.want)
		}
	}
	// The spring-forward 02:30 wall time does not exist; cron skips that day.
	now, _ := time.Parse(time.RFC3339, "2027-03-14T06:00:00Z")
	got, err := nextScheduleTime("cron", "30 2 * * *", "America/New_York", nil, now)
	if err != nil || got.Format(time.RFC3339) != "2027-03-15T06:30:00Z" {
		t.Fatalf("DST skip: %v %v", got, err)
	}
}
