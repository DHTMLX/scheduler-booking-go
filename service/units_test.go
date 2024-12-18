package service

import (
	"reflect"
	"testing"
	"time"
)

func Test_m2t(t *testing.T) {
	cases := []struct {
		minutes       int
		expectedHours string
	}{
		{
			minutes:       15,
			expectedHours: "0:15",
		},
		{
			minutes:       30,
			expectedHours: "0:30",
		},
		{
			minutes:       60,
			expectedHours: "1:00",
		},
		{
			minutes:       90,
			expectedHours: "1:30",
		},
		{
			minutes:       135,
			expectedHours: "2:15",
		},
		{
			minutes:       545,
			expectedHours: "9:05",
		},
		{
			minutes:       875,
			expectedHours: "14:35",
		},
		{
			minutes:       1020,
			expectedHours: "17:00",
		},
		{
			minutes:       1260,
			expectedHours: "21:00",
		},
		{
			minutes:       1440,
			expectedHours: "24:00",
		},
		{
			minutes:       1600,
			expectedHours: "24:00",
		},
	}

	for _, c := range cases {
		h := m2t(c.minutes)
		if h != c.expectedHours {
			t.Fatalf("expected %s, got %s", c.expectedHours, h)
		}
	}
}

func TestDaysFromRules(t *testing.T) {
	cases := []struct {
		rrule string
		days  []int
	}{
		{
			rrule: "FREQ=WEEKLY;INTERVAL=1;BYDAY=MO",
			days:  []int{1},
		},
		{
			rrule: "INTERVAL=1;FREQ=WEEKLY;BYDAY=MO,WE,FR",
			days:  []int{1, 3, 5},
		},
		{
			rrule: "BYDAY=SA,SU;FREQ=WEEKLY;INTERVAL=1",
			days:  []int{6, 0},
		},
		{
			rrule: "BYDAY=SU;INTERVAL=2;FREQ=WEEKLY",
			days:  []int{0},
		},
		{
			rrule: "FREQ=WEEKLY;BYDAY=TU,TH;INTERVAL=1",
			days:  []int{2, 4},
		},
		{
			rrule: "INTERVAL=1;BYDAY=TU,TH;FREQ=WEEKLY",
			days:  []int{2, 4},
		},
		{
			rrule: "INTERVAL=1;BYDAY=;FREQ=WEEKLY",
			days:  []int{},
		},
		{
			rrule: "INTERVAL=1;BYDAY=",
			days:  []int{},
		},
		{
			rrule: "BYDAY=;FREQ=WEEKLY",
			days:  []int{},
		},
		{
			rrule: "BYDAY=;",
			days:  []int{},
		},
		{
			rrule: "FREQ=WEEKLY;INTERVAL=1;BYDAY=Mo,wE,fr",
			days:  []int{1, 3, 5},
		},
		{
			rrule: "INTERVAL=1;FREQ=WEEKLY;BYDAY=MO,WE,FE",
			days:  []int{1, 3},
		},
		{
			rrule: "INTERVAL=1;FREQ=WEEKLY;BYDAY=FR,WE,MO",
			days:  []int{5, 3, 1},
		},
	}

	for _, c := range cases {
		days := daysFromRules(c.rrule)
		if !reflect.DeepEqual(c.days, days) {
			t.Fatalf("expected %v, got %v", c.days, days)
		}
	}
}

func TestSlots(t *testing.T) {
	type TestCase struct {
		date          time.Time
		from          int
		to            int
		size          int
		gap           int
		answer        []int64 // for client reservation
		answerReplace []int64 // for booking
	}

	cases := []TestCase{
		{
			date: time.Date(2024, 12, 16, 14, 20, 0, 0, time.UTC), // 2024-12-16 14:20
			from: 12 * 60,
			to:   17 * 60,
			size: 20,
			gap:  20,
			// 12:00, 12:40, 13:20, 14:00, 14:40, 15:20, 16:00, 16:40
			answer:        []int64{1734358800000},                // 2024-12-16 14:20
			answerReplace: []int64{1734357600000, 1734360000000}, // 2024-12-16 14:00; 2024-12-16 14:40
		},
		{
			date: time.Date(2024, 12, 16, 14, 0, 0, 0, time.UTC), // 2024-12-16 14:00
			from: 12 * 60,
			to:   17 * 60,
			size: 40,
			gap:  10,
			// 12:00, 12:50, 13:40, 14:30, 15:20, 16:10
			answer:        []int64{1734357600000},                // 2024-12-16 14:00
			answerReplace: []int64{1734356400000, 1734359400000}, // 2024-12-16 13:40; 2024-12-16 14:30
		},
		{
			date: time.Date(2024, 12, 16, 14, 10, 0, 0, time.UTC), // 2024-12-16 14:10
			from: 12 * 60,
			to:   17 * 60,
			size: 10,
			gap:  40,
			// 12:00, 12:50, 13:40, 14:30, 15:20, 16:10
			answer:        []int64{1734358200000},                // 2024-12-16 14:10
			answerReplace: []int64{1734356400000, 1734359400000}, // 2024-12-16 13:40; 2024-12-16 14:30
		},
		// without moving schedule
		{
			date: time.Date(2024, 12, 16, 12, 0, 0, 0, time.UTC), // 2024-12-16 12:00
			from: 8 * 60,
			to:   17 * 60,
			size: 20,
			gap:  20,
			// 8:00, 8:40, 9:20, 10:00, 10:40, 11:20, 12:00, 12:40, 13:20, 14:00, 14:40, 15:20, 16:00, 16:40
			answer:        []int64{1734350400000}, // 2024-12-16 12:00
			answerReplace: []int64{1734350400000}, // 2024-12-16 12:00
		},
		{
			date: time.Date(2024, 12, 16, 13, 0, 0, 0, time.UTC), // 2024-12-16 13:00
			from: 12 * 60,
			to:   14 * 60,
			size: 40,
			gap:  20,
			// 12:00, 13:00
			answer:        []int64{1734354000000}, // 2024-12-16 13:00
			answerReplace: []int64{1734354000000}, // 2024-12-16 13:00
		},
		// schedule start
		{
			date: time.Date(2024, 12, 16, 8, 0, 0, 0, time.UTC), // 2024-12-16 8:00
			from: 9 * 60,
			to:   11 * 60,
			size: 20,
			gap:  20,
			// 9:00, 9:40, 10:20, 11:00
			answer:        []int64{},
			answerReplace: []int64{},
		},
		{
			date: time.Date(2024, 12, 16, 11, 15, 0, 0, time.UTC), // 2024-12-16 11:15
			from: 12 * 60,
			to:   14 * 60,
			size: 40,
			gap:  10,
			// 12:00, 12:50
			answer:        []int64{},
			answerReplace: []int64{1734350400000}, // 2024-12-16 12:00
		},
		{
			date: time.Date(2024, 12, 16, 11, 10, 0, 0, time.UTC), // 2024-12-16 11:10
			from: 12 * 60,
			to:   14 * 60,
			size: 40,
			gap:  10,
			// 12:00, 12:50
			answer:        []int64{},
			answerReplace: []int64{},
		},
		// schedule end
		{
			date: time.Date(2024, 12, 16, 13, 45, 0, 0, time.UTC), // 2024-12-16 13:45
			from: 12 * 60,
			to:   14 * 60,
			size: 40,
			gap:  20,
			// 12:00, 13:00
			answer:        []int64{},
			answerReplace: []int64{1734354000000}, // 2024-12-16 13:00
		},
		{
			date: time.Date(2024, 12, 16, 14, 0, 0, 0, time.UTC), // 2024-12-16 14:00
			from: 12 * 60,
			to:   14 * 60,
			size: 40,
			gap:  20,
			// 12:00, 13:00
			answer:        []int64{},
			answerReplace: []int64{},
		},
		{
			date: time.Date(2024, 12, 16, 13, 30, 0, 0, time.UTC), // 2024-12-16 13:30
			from: 12 * 60,
			to:   14*60 + 40,
			size: 40,
			gap:  20,
			// 12:00, 13:00, 14:00
			answer:        []int64{1734355800000},                // 2024-12-16 13:30
			answerReplace: []int64{1734354000000, 1734357600000}, // 2024-12-16 13:00; 2024-12-16 14:00
		},
		// other
		{
			date: time.Date(2024, 12, 16, 14, 0, 0, 0, time.UTC), // 2024-12-16 14:00
			from: 14 * 60,
			to:   14*60 + 30,
			size: 40,
			gap:  20,
			// []
			answer:        []int64{},
			answerReplace: []int64{},
		},
	}

	checkTestCase := func(c *TestCase, replace bool) {
		stamps := getTimestamps(
			&c.date,
			c.from,
			c.to,
			c.size,
			c.gap,
			replace,
		)

		answer := c.answer
		if replace {
			answer = c.answerReplace
		}

		if !reflect.DeepEqual(stamps, answer) {
			t.Fatalf("expected %v, got %v", answer, stamps)
		}
	}

	for _, c := range cases {
		checkTestCase(&c, false)
		checkTestCase(&c, true)
	}
}
