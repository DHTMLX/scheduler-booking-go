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
			expectedHours: "00:15",
		},
		{
			minutes:       30,
			expectedHours: "00:30",
		},
		{
			minutes:       60,
			expectedHours: "01:00",
		},
		{
			minutes:       90,
			expectedHours: "01:30",
		},
		{
			minutes:       135,
			expectedHours: "02:15",
		},
		{
			minutes:       545,
			expectedHours: "09:05",
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
			minutes:       1480,
			expectedHours: "24:40",
		},
	}

	for _, c := range cases {
		hours := m2t(c.minutes)
		if hours != c.expectedHours {
			t.Fatalf("expected %s, got %s", c.expectedHours, hours)
		}
	}
}

func TestDaysFromRules(t *testing.T) {
	cases := []struct {
		rrule string
		days  map[int]struct{}
	}{
		{
			rrule: "FREQ=WEEKLY;INTERVAL=1;BYDAY=MO",
			days: map[int]struct{}{
				1: {},
			},
		},
		{
			rrule: "INTERVAL=1;FREQ=WEEKLY;BYDAY=MO,WE,FR",
			days: map[int]struct{}{
				1: {},
				3: {},
				5: {},
			},
		},
		{
			rrule: "BYDAY=SA,SU;FREQ=WEEKLY;INTERVAL=1",
			days: map[int]struct{}{
				0: {},
				6: {},
			},
		},
		{
			rrule: "BYDAY=SU;INTERVAL=2;FREQ=WEEKLY",
			days: map[int]struct{}{
				0: {},
			},
		},
		{
			rrule: "FREQ=WEEKLY;BYDAY=TU,TH;INTERVAL=1",
			days: map[int]struct{}{
				2: {},
				4: {},
			},
		},
		{
			rrule: "INTERVAL=1;BYDAY=TU,TH;FREQ=WEEKLY",
			days: map[int]struct{}{
				2: {},
				4: {},
			},
		},
		{
			rrule: "INTERVAL=1;BYDAY=;FREQ=WEEKLY",
			days:  map[int]struct{}{},
		},
		{
			rrule: "INTERVAL=1;BYDAY=",
			days:  map[int]struct{}{},
		},
		{
			rrule: "BYDAY=;FREQ=WEEKLY",
			days:  map[int]struct{}{},
		},
		{
			rrule: "BYDAY=;",
			days:  map[int]struct{}{},
		},
		{
			rrule: "FREQ=WEEKLY;INTERVAL=1;BYDAY=Mo,wE,fr",
			days: map[int]struct{}{
				1: {},
				3: {},
				5: {},
			},
		},
		{
			rrule: "INTERVAL=1;FREQ=WEEKLY;BYDAY=MO,WE,FE",
			days: map[int]struct{}{
				1: {},
				3: {},
			},
		},
		{
			rrule: "INTERVAL=1;FREQ=WEEKLY;BYDAY=FR,WE,MO",
			days: map[int]struct{}{
				1: {},
				3: {},
				5: {},
			},
		},
	}

	for _, c := range cases {
		days := daysFromRules(c.rrule)
		if !reflect.DeepEqual(c.days, days) {
			t.Fatalf("expected %v, got %v", c.days, days)
		}
	}
}

func TestBookedSlots(t *testing.T) {
	type TestCase struct {
		slots         []time.Time
		date          int64 // (optional) default is the date of the first slot
		from          int
		to            int
		size          int
		gap           int
		answer        []int64 // for client reservation
		answerReplace []int64 // for booking
	}

	cases := []TestCase{
		{
			slots: []time.Time{time.Date(2024, 12, 16, 14, 20, 0, 0, time.UTC)}, // 2024-12-16 14:20
			from:  12 * 60,
			to:    17 * 60,
			size:  20,
			gap:   20,
			// 12:00, 12:40, 13:20, 14:00, 14:40, 15:20, 16:00, 16:40
			answer:        []int64{1734358800000},                // 2024-12-16 14:20
			answerReplace: []int64{1734357600000, 1734360000000}, // 2024-12-16 14:00; 2024-12-16 14:40
		},
		{
			slots: []time.Time{time.Date(2024, 12, 16, 14, 0, 0, 0, time.UTC)}, // 2024-12-16 14:00
			from:  12 * 60,
			to:    17 * 60,
			size:  40,
			gap:   10,
			// 12:00, 12:50, 13:40, 14:30, 15:20, 16:10
			answer:        []int64{1734357600000},                // 2024-12-16 14:00
			answerReplace: []int64{1734356400000, 1734359400000}, // 2024-12-16 13:40; 2024-12-16 14:30
		},
		{
			slots: []time.Time{time.Date(2024, 12, 16, 14, 10, 0, 0, time.UTC)}, // 2024-12-16 14:10
			from:  12 * 60,
			to:    17 * 60,
			size:  10,
			gap:   40,
			// 12:00, 12:50, 13:40, 14:30, 15:20, 16:10
			answer:        []int64{1734358200000},                // 2024-12-16 14:10
			answerReplace: []int64{1734356400000, 1734359400000}, // 2024-12-16 13:40; 2024-12-16 14:30
		},
		// without moving schedule
		{
			slots: []time.Time{time.Date(2024, 12, 16, 12, 0, 0, 0, time.UTC)}, // 2024-12-16 12:00
			from:  8 * 60,
			to:    17 * 60,
			size:  20,
			gap:   20,
			// 8:00, 8:40, 9:20, 10:00, 10:40, 11:20, 12:00, 12:40, 13:20, 14:00, 14:40, 15:20, 16:00, 16:40
			answer:        []int64{1734350400000}, // 2024-12-16 12:00
			answerReplace: []int64{1734350400000}, // 2024-12-16 12:00
		},
		{
			slots: []time.Time{time.Date(2024, 12, 16, 13, 0, 0, 0, time.UTC)}, // 2024-12-16 13:00
			from:  12 * 60,
			to:    14 * 60,
			size:  40,
			gap:   20,
			// 12:00, 13:00
			answer:        []int64{1734354000000}, // 2024-12-16 13:00
			answerReplace: []int64{1734354000000}, // 2024-12-16 13:00
		},
		// schedule start
		{
			slots: []time.Time{time.Date(2024, 12, 16, 8, 0, 0, 0, time.UTC)}, // 2024-12-16 8:00
			from:  9 * 60,
			to:    11 * 60,
			size:  20,
			gap:   20,
			// 9:00, 9:40, 10:20, 11:00
			answer:        []int64{},
			answerReplace: []int64{},
		},
		{
			slots: []time.Time{time.Date(2024, 12, 16, 11, 15, 0, 0, time.UTC)}, // 2024-12-16 11:15
			from:  12 * 60,
			to:    14 * 60,
			size:  40,
			gap:   10,
			// 12:00, 12:50
			answer:        []int64{},
			answerReplace: []int64{1734350400000}, // 2024-12-16 12:00
		},
		{
			slots: []time.Time{time.Date(2024, 12, 16, 11, 10, 0, 0, time.UTC)}, // 2024-12-16 11:10
			from:  12 * 60,
			to:    14 * 60,
			size:  40,
			gap:   10,
			// 12:00, 12:50
			answer:        []int64{},
			answerReplace: []int64{},
		},
		// schedule end
		{
			slots: []time.Time{time.Date(2024, 12, 16, 13, 45, 0, 0, time.UTC)}, // 2024-12-16 13:45
			from:  12 * 60,
			to:    14 * 60,
			size:  40,
			gap:   20,
			// 12:00, 13:00
			answer:        []int64{},
			answerReplace: []int64{1734354000000}, // 2024-12-16 13:00
		},
		{
			slots: []time.Time{time.Date(2024, 12, 16, 14, 0, 0, 0, time.UTC)}, // 2024-12-16 14:00
			from:  12 * 60,
			to:    14 * 60,
			size:  40,
			gap:   20,
			// 12:00, 13:00
			answer:        []int64{},
			answerReplace: []int64{},
		},
		{
			slots: []time.Time{time.Date(2024, 12, 16, 13, 30, 0, 0, time.UTC)}, // 2024-12-16 13:30
			from:  12 * 60,
			to:    14*60 + 40,
			size:  40,
			gap:   20,
			// 12:00, 13:00, 14:00
			answer:        []int64{1734355800000},                // 2024-12-16 13:30
			answerReplace: []int64{1734354000000, 1734357600000}, // 2024-12-16 13:00; 2024-12-16 14:00
		},
		// two-day schedule
		{
			slots: []time.Time{time.Date(2024, 12, 16, 23, 45, 0, 0, time.UTC)}, // 2024-12-16 23:45
			from:  22 * 60,
			to:    26 * 60,
			size:  40,
			gap:   20,
			// 22:00, 23:00, 24:00, 01:00
			answer:        []int64{1734392700000},                // 2024-12-16 23:45
			answerReplace: []int64{1734390000000, 1734393600000}, // 2024-12-16 23:00,  2024-12-17 00:00
		},
		{
			slots: []time.Time{time.Date(2024, 12, 17, 0, 15, 0, 0, time.UTC)}, // 2024-12-17 0:15
			date:  time.Date(2024, 12, 16, 0, 0, 0, 0, time.UTC).UnixMilli(),
			from:  22 * 60,
			to:    24 * 60,
			size:  40,
			gap:   40,
			// 22:00, 23:20
			answer:        []int64{},
			answerReplace: []int64{1734391200000}, // 2024-12-16 23:20
		},
		{
			slots: []time.Time{time.Date(2024, 12, 16, 23, 45, 0, 0, time.UTC)}, // 2024-12-16 23:45
			date:  time.Date(2024, 12, 17, 0, 0, 0, 0, time.UTC).UnixMilli(),
			from:  0 * 60,
			to:    2 * 60,
			size:  30,
			gap:   10,
			// 00:00, 00:40, 01:20
			answer:        []int64{},
			answerReplace: []int64{1734393600000}, // 2024-12-17 00:00
		},
		// other
		{
			slots: []time.Time{time.Date(2024, 12, 16, 14, 0, 0, 0, time.UTC)}, // 2024-12-16 14:00
			from:  14 * 60,
			to:    14*60 + 30,
			size:  40,
			gap:   20,
			// []
			answer:        []int64{},
			answerReplace: []int64{},
		},
		{
			slots: []time.Time{
				time.Date(2024, 12, 15, 23, 30, 0, 0, time.UTC), // 2024-12-16 23:30
				time.Date(2024, 12, 17, 0, 5, 0, 0, time.UTC),   // 2024-12-17 00:05
			},
			date: time.Date(2024, 12, 16, 0, 0, 0, 0, time.UTC).UnixMilli(), // 2024-12-16 00:00
			from: 0 * 60,
			to:   24 * 60,
			size: 4 * 60,
			gap:  5,
			// 00:00, 04:05, 08:10, 12:15, 16:20
			answer:        []int64{},
			answerReplace: []int64{1734307200000}, // 2024-12-16 00:00
		},
		{
			slots: []time.Time{
				time.Date(2024, 12, 15, 23, 30, 0, 0, time.UTC), // 2024-12-16 23:30
				time.Date(2024, 12, 17, 0, 5, 0, 0, time.UTC),   // 2024-12-17 00:05
			},
			date: time.Date(2024, 12, 16, 0, 0, 0, 0, time.UTC).UnixMilli(), // 2024-12-16 00:00
			from: 0 * 60,
			to:   24 * 60,
			size: 3 * 60,
			gap:  1*60 + 5,
			// 00:00, 04:05, 08:10, 12:15, 16:20, 20:25
			answer:        []int64{},
			answerReplace: []int64{1734307200000, 1734380700000}, // 2024-12-16 00:00; 2024-12-16 20:25
		},
	}

	checkTestCase := func(slots map[int64][]time.Time, c *TestCase, replace bool) {
		stamps := getBookedSlots(
			slots,
			c.date,
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
		slots := make(map[int64][]time.Time)
		for _, slot := range c.slots {
			date := slot.Truncate(24 * time.Hour).UnixMilli()
			slots[date] = append(slots[date], slot)

			if c.date == 0 {
				c.date = date
			}
		}

		checkTestCase(slots, &c, false)
		checkTestCase(slots, &c, true)
	}
}
