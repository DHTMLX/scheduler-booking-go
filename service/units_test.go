package service

import (
	"reflect"
	"scheduler-booking/common"
	"testing"
	"time"
)

func TestDaysFromRules(t *testing.T) {
	cases := []struct {
		rrule string
		days  []int
	}{
		{
			rrule: "FREQ=WEEKLY;INTERVAL=1;BYDAY=MO",
			days: []int{
				1,
			},
		},
		{
			rrule: "INTERVAL=1;FREQ=WEEKLY;BYDAY=MO,WE,FR",
			days: []int{
				1,
				3,
				5,
			},
		},
		{
			rrule: "BYDAY=SA,SU;FREQ=WEEKLY;INTERVAL=1",
			days: []int{
				6,
				0,
			},
		},
		{
			rrule: "BYDAY=SU;INTERVAL=2;FREQ=WEEKLY",
			days: []int{
				0,
			},
		},
		{
			rrule: "FREQ=WEEKLY;BYDAY=TU,TH;INTERVAL=1",
			days: []int{
				2,
				4,
			},
		},
		{
			rrule: "INTERVAL=1;BYDAY=TU,TH;FREQ=WEEKLY",
			days: []int{
				2,
				4,
			},
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
			days: []int{
				1,
				3,
				5,
			},
		},
		{
			rrule: "INTERVAL=1;FREQ=WEEKLY;BYDAY=MO,WE,FE", // MO,WE
			days: []int{
				1,
				3,
			},
		},
		{
			rrule: "INTERVAL=1;FREQ=WEEKLY;BYDAY=FR,WE,MO",
			days: []int{
				5,
				3,
				1,
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

func TestCreateSchedules(t *testing.T) {
	cases := []struct {
		From  int
		To    int
		Size  int
		Gap   int
		Days  []int
		Dates []int64

		Schedules []Schedule
	}{
		{
			From: 22*60 + 40,   // 22:40
			To:   24*60 + 1*60, // 01:00
			Size: 20,
			Gap:  40,
			// 22:40, 23:40, 00:40,
			Days:  []int{1, 2, 3},
			Dates: []int64{time.Date(2025, 1, 8, 0, 0, 0, 0, time.UTC).UnixMilli()},

			Schedules: []Schedule{
				{
					From:  common.NewJTime(22*60 + 40), // 22:40
					To:    common.NewJTime(24*60 + 40), // 24:40
					Days:  []int{1, 2, 3},
					Dates: []int64{time.Date(2025, 1, 8, 0, 0, 0, 0, time.UTC).UnixMilli()},
				},
				{
					From:  common.NewJTime(40),     // 00:40
					To:    common.NewJTime(1 * 60), // 01:00
					Days:  []int{2, 3, 4},
					Dates: []int64{time.Date(2025, 1, 9, 0, 0, 0, 0, time.UTC).UnixMilli()},
				},
			},
		},
		{
			From: 21*60 + 35,   // 21:35
			To:   24*60 + 1*60, // 01:00
			Size: 30,
			Gap:  35,
			Days: []int{4, 5, 6},
			Dates: []int64{
				time.Date(2025, 1, 8, 0, 0, 0, 0, time.UTC).UnixMilli(),
				time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC).UnixMilli(),
			},

			Schedules: []Schedule{
				{
					From: common.NewJTime(21*60 + 35), // 21:35
					To:   common.NewJTime(24*60 + 50), // 24:50
					Days: []int{4, 5, 6},
					Dates: []int64{
						time.Date(2025, 1, 8, 0, 0, 0, 0, time.UTC).UnixMilli(),
						time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC).UnixMilli(),
					},
				},
			},
		},
		{
			From: 22*60 + 40,   // 22:40
			To:   24*60 + 1*60, // 01:00
			Size: 40,
			Gap:  40,
			Days: []int{4, 5, 6},
			Dates: []int64{
				time.Date(2025, 1, 8, 0, 0, 0, 0, time.UTC).UnixMilli(),
				time.Date(2025, 1, 9, 0, 0, 0, 0, time.UTC).UnixMilli(),
			},

			Schedules: []Schedule{
				{
					From: common.NewJTime(22*60 + 40), // 22:40
					To:   common.NewJTime(24 * 60),    // 24:00
					Days: []int{4, 5, 6},
					Dates: []int64{
						time.Date(2025, 1, 8, 0, 0, 0, 0, time.UTC).UnixMilli(),
						time.Date(2025, 1, 9, 0, 0, 0, 0, time.UTC).UnixMilli(),
					},
				},
				{
					From: common.NewJTime(0),       // 00:00
					To:   common.NewJTime(01 * 60), // 01:00
					Days: []int{5, 6, 0},
					Dates: []int64{
						time.Date(2025, 1, 9, 0, 0, 0, 0, time.UTC).UnixMilli(),
						time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC).UnixMilli(),
					},
				},
			},
		},
		{
			From: 23*60 + 40, // 23:40
			To:   24*60 + 10, // 00:10
			Size: 40,
			Gap:  40,
			Days: []int{4, 5, 6},
			Dates: []int64{
				time.Date(2025, 1, 8, 0, 0, 0, 0, time.UTC).UnixMilli(),
				time.Date(2025, 1, 9, 0, 0, 0, 0, time.UTC).UnixMilli(),
			},

			Schedules: []Schedule{
				{
					From: common.NewJTime(23*60 + 40), // 23:40
					To:   common.NewJTime(24*60 + 10), // 24:10
					Days: []int{4, 5, 6},
					Dates: []int64{
						time.Date(2025, 1, 8, 0, 0, 0, 0, time.UTC).UnixMilli(),
						time.Date(2025, 1, 9, 0, 0, 0, 0, time.UTC).UnixMilli(),
					},
				},
			},
		},
	}

	for i, c := range cases {
		schedules := createSchedules(c.From, c.To, c.Size, c.Gap, c.Days, c.Dates)
		if len(c.Schedules) != len(schedules) {
			t.Fatal("len()", i, c.Schedules, schedules)
		}

		for i, schedule := range schedules {
			if c.Schedules[i].From.Get() != schedule.From.Get() ||
				c.Schedules[i].To.Get() != schedule.To.Get() ||
				!reflect.DeepEqual(c.Schedules[i].Days, schedule.Days) ||
				!reflect.DeepEqual(c.Schedules[i].Dates, schedule.Dates) {
				t.Fatalf("expected: %+v\ngot: %+v", c.Schedules[i], schedule)
			}
		}
	}
}

func TestBookedSlots(t *testing.T) {
	type TestCase struct {
		slots         []time.Time
		date          int64 // (optional) default - schedule date
		from          int
		to            int
		size          int
		gap           int
		answer        []int64 // for client reservation
		answerReplace []int64 // for booking
	}

	cases := []TestCase{
		{
			slots: []time.Time{time.Date(2024, 12, 16, 12, 00, 0, 0, time.UTC)}, // 2024-12-16 12:00
			from:  12 * 60,
			to:    12 * 60,
			size:  20,
			gap:   20,
			//
			answer:        []int64{},
			answerReplace: []int64{},
		},
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
		stamps := getRoutBookedSlots(
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
