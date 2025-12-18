package adventofcode

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCalculateNextReminder(t *testing.T) {
	// December 1, 2024 at 00:00:00 EST (midnight when day 1 starts)
	day1Start := time.Date(2024, time.December, 1, 0, 0, 0, 0, tzEastern)
	day1Timestamp := day1Start.Unix()

	tests := []struct {
		name              string
		leaderboard       *Leaderboard
		now               time.Time
		expectedDay       int
		expectedReminder  time.Time
	}{
		{
			name: "before event starts",
			leaderboard: &Leaderboard{
				Event:         "2024",
				NumDays:       25,
				Day1Timestamp: day1Timestamp,
			},
			now:              time.Date(2024, time.November, 30, 12, 0, 0, 0, tzEastern),
			expectedDay:      1,
			expectedReminder: time.Date(2024, time.November, 30, 23, 45, 0, 0, tzEastern),
		},
		{
			name: "day 1, before reminder time",
			leaderboard: &Leaderboard{
				Event:         "2024",
				NumDays:       25,
				Day1Timestamp: day1Timestamp,
			},
			now:              time.Date(2024, time.November, 30, 23, 0, 0, 0, tzEastern),
			expectedDay:      1,
			expectedReminder: time.Date(2024, time.November, 30, 23, 45, 0, 0, tzEastern),
		},
		{
			name: "day 1, after reminder time but before day starts",
			leaderboard: &Leaderboard{
				Event:         "2024",
				NumDays:       25,
				Day1Timestamp: day1Timestamp,
			},
			now:              time.Date(2024, time.November, 30, 23, 59, 0, 0, tzEastern),
			expectedDay:      2,
			expectedReminder: time.Date(2024, time.December, 1, 23, 45, 0, 0, tzEastern),
		},
		{
			name: "day 1, after reminder time but before day 2",
			leaderboard: &Leaderboard{
				Event:         "2024",
				NumDays:       25,
				Day1Timestamp: day1Timestamp,
			},
			now:              time.Date(2024, time.December, 1, 12, 0, 0, 0, tzEastern),
			expectedDay:      2,
			expectedReminder: time.Date(2024, time.December, 1, 23, 45, 0, 0, tzEastern),
		},
		{
			name: "day 5, before reminder time",
			leaderboard: &Leaderboard{
				Event:         "2024",
				NumDays:       25,
				Day1Timestamp: day1Timestamp,
			},
			now:              time.Date(2024, time.December, 4, 23, 0, 0, 0, tzEastern),
			expectedDay:      5,
			expectedReminder: time.Date(2024, time.December, 4, 23, 45, 0, 0, tzEastern),
		},
		{
			name: "day 5, after reminder time",
			leaderboard: &Leaderboard{
				Event:         "2024",
				NumDays:       25,
				Day1Timestamp: day1Timestamp,
			},
			now:              time.Date(2024, time.December, 5, 12, 0, 0, 0, tzEastern),
			expectedDay:      6,
			expectedReminder: time.Date(2024, time.December, 5, 23, 45, 0, 0, tzEastern),
		},
		{
			name: "last day (day 25), before reminder",
			leaderboard: &Leaderboard{
				Event:         "2024",
				NumDays:       25,
				Day1Timestamp: day1Timestamp,
			},
			now:              time.Date(2024, time.December, 24, 23, 0, 0, 0, tzEastern),
			expectedDay:      25,
			expectedReminder: time.Date(2024, time.December, 24, 23, 45, 0, 0, tzEastern),
		},
		{
			name: "last day (day 25), after reminder",
			leaderboard: &Leaderboard{
				Event:         "2024",
				NumDays:       25,
				Day1Timestamp: day1Timestamp,
			},
			now:              time.Date(2024, time.December, 25, 12, 0, 0, 0, tzEastern),
			expectedDay:      1,
			expectedReminder: time.Date(2025, time.November, 30, 23, 45, 0, 0, tzEastern),
		},
		{
			name: "after event ends",
			leaderboard: &Leaderboard{
				Event:         "2024",
				NumDays:       25,
				Day1Timestamp: day1Timestamp,
			},
			now:              time.Date(2024, time.December, 26, 12, 0, 0, 0, tzEastern),
			expectedDay:      1,
			expectedReminder: time.Date(2025, time.November, 30, 23, 45, 0, 0, tzEastern),
		},
		{
			name: "custom event with 10 days",
			leaderboard: &Leaderboard{
				Event:         "2024",
				NumDays:       10,
				Day1Timestamp: day1Timestamp,
			},
			now:              time.Date(2024, time.December, 5, 12, 0, 0, 0, tzEastern),
			expectedDay:      6,
			expectedReminder: time.Date(2024, time.December, 5, 23, 45, 0, 0, tzEastern),
		},
		{
			name: "custom event with 10 days, on last day after reminder",
			leaderboard: &Leaderboard{
				Event:         "2024",
				NumDays:       10,
				Day1Timestamp: day1Timestamp,
			},
			now:              time.Date(2024, time.December, 10, 12, 0, 0, 0, tzEastern),
			expectedDay:      1,
			expectedReminder: time.Date(2025, time.November, 30, 23, 45, 0, 0, tzEastern),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextReminder, day := calculateNextReminder(tt.leaderboard, tt.now)

			assert.Equal(t, tt.expectedDay, day, "day number should match")
			assert.True(t, nextReminder.Equal(tt.expectedReminder),
				"reminder time should match: expected %s, got %s",
				tt.expectedReminder.Format(time.RFC3339),
				nextReminder.Format(time.RFC3339))
		})
	}
}
