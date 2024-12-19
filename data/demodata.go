package data

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"gorm.io/gorm"
)

func dataDown(tx *gorm.DB) {
	must(tx.Exec("DELETE FROM `doctors`").Error)
	must(tx.Exec("DELETE FROM `doctor_routine`").Error)
	must(tx.Exec("DELETE FROM `doctor_schedule`").Error)
	must(tx.Exec("DELETE FROM `doctor_recurring`").Error)
	must(tx.Exec("DELETE FROM `occupied_slots`").Error)
	must(tx.Exec("DELETE FROM `reviews`").Error)
}

var (
	firstNames = []string{
		"Emma",
		"Olivia",
		"James",
		"Mia",
		"Amelia",
		"Alexander",
		"Harper",
		"William",
		"Abigail",
		"Lily",
	}

	lastNames = []string{
		"Johnson",
		"Smith",
		"Brown",
		"Wilson",
		"Jackson",
		"King",
		"Scott",
		"Green",
		"Adams",
		"Baker",
	}
)

const (
	nameFormat  = "%s %s"
	emailFormat = "%s.%s@scheduler.booking"
)

func dataUp(tx *gorm.DB) {
	now := time.Now().UTC()
	y, m, d := now.Date()

	genSchedule := func(from, to int, date int64, days int) []DoctorSchedule {
		date += int64(from) * 60 * 1000
		routine := make([]DoctorSchedule, days)
		for i := range routine {
			workDay := DoctorSchedule{
				From: from,
				To:   to,
				DoctorRoutine: &DoctorRoutine{
					Date: date + int64(i)*24*60*60*1000,
				},
			}
			routine[i] = workDay
		}
		return routine
	}

	newSlots := func(date int64, times ...int) []OccupiedSlot {
		slots := make([]OccupiedSlot, len(times))
		for i, time := range times {
			first := firstNames[rand.Intn(len(firstNames))]
			last := lastNames[rand.Intn(len(lastNames))]

			slots[i] = OccupiedSlot{
				Date:        date + int64(time)*60*1000,
				ClientName:  fmt.Sprintf(nameFormat, first, last),
				ClientEmail: fmt.Sprintf(emailFormat, first, last),
			}
		}

		return slots
	}

	genSlots := func(slots ...[]OccupiedSlot) []OccupiedSlot {
		allSlots := make([]OccupiedSlot, 0, len(slots)*2)
		for _, slot := range slots {
			allSlots = append(allSlots, slot...)
		}

		return allSlots
	}

	nextWeekDay := func(day int, weeks ...int) int64 {
		days := 0
		if len(weeks) > 0 && weeks[0] > 0 {
			days = 7 * weeks[0]
		}

		next := (7 + day - int(now.Weekday())) % 7
		return time.Date(y, m, d+next+days, 0, 0, 0, 0, time.UTC).UnixMilli()
	}

	RecurringSchedule := func(from, to int, rrule string) DoctorSchedule {
		date := time.Date(y, m, d, 0, from, 0, 0, time.UTC).UnixMilli()

		if to < from {
			to += 24 * 60 // one day
		}

		return DoctorSchedule{
			From: from,
			To:   to,
			DoctorRecurringRoutine: &DoctorRecurringRoutine{
				Date:     date,
				Rrule:    "INTERVAL=1;FREQ=WEEKLY;BYDAY=" + strings.ToUpper(rrule),
				Duration: (to - from) * 60,
			},
		}
	}

	doctors := []Doctor{
		{
			Name:     "Dr. Conrad Hubbard",
			Category: "Psychiatrist",
			Subtitle: "2 years of experience",
			Details:  "Desert Springs Hospital (Schroeders Avenue 90, Fannett, Ethiopia)",
			SlotSize: 20,
			Price:    "$45",
			ImageURL: "https://snippet.dhtmlx.com/codebase/data/booking/01/img/11.jpg",
			Gap:      20,
			Review: Review{
				Count: 1245,
				Stars: 4,
			},
			DoctorSchedule: append(
				[]DoctorSchedule{
					// every week day 9:00-17:00 (except sun, sat - holidays)
					RecurringSchedule(9*60, 17*60, "MO,TU,WE,TH,FR"),
				},
				// next tue, wed, thu 2:00-6:00
				genSchedule(2*60, 6*60, nextWeekDay(2), 3)...,
			),
			OccupiedSlots: genSlots(
				newSlots(nextWeekDay(1), 9*60+40),           // next mon 9:40
				newSlots(nextWeekDay(2), 11*60, 15*60),      // next tue 11:00, 15:00
				newSlots(nextWeekDay(3, 1), 11*60),          // after next wed 11:00
				newSlots(nextWeekDay(4), 3*60+20, 16*60+20), // next thu 3:20, 16:20
				newSlots(nextWeekDay(4, 1), 5*60+20),        // after next thu 5:20
				newSlots(nextWeekDay(5), 13*60+20),          // next fri 13:20
			),
		},
		{
			Name:     "Dr. Debra Weeks",
			Category: "Allergist",
			Subtitle: "7 years of experience",
			Details:  "Silverstone Medical Center (Vanderbilt Avenue 13, Chestnut, New Zealand)",
			SlotSize: 45,
			Price:    "$120",
			ImageURL: "https://snippet.dhtmlx.com/codebase/data/booking/01/img/03.jpg",
			Gap:      5,
			Review: Review{
				Count: 6545,
				Stars: 4,
			},
			DoctorSchedule: append(
				[]DoctorSchedule{
					// mon, wed 7:00-15:00
					RecurringSchedule(7*60, 15*60, "MO,WE"),
					// tue, thu 12:00-20:00
					RecurringSchedule(12*60, 20*60, "TU,TH"),
					// sat-sun 20:00-4:00
					RecurringSchedule(20*60, 4*60, "SA"), // or RecurringSchedule(20*60, 28*60, "SA")
				},
				// next wed 18:00-22:00
				genSchedule(18*60, 22*60, nextWeekDay(3), 1)...,
			),
			OccupiedSlots: genSlots(
				newSlots(nextWeekDay(1), 7*60+50),            // next mon 7:50
				newSlots(nextWeekDay(2), 13*60+40),           // next tue 13:40
				newSlots(nextWeekDay(3), 11*60+10),           // next wed 11:10
				newSlots(nextWeekDay(4), 14*60+30, 17*60+50), // next thu 14:30 17:50
				newSlots(nextWeekDay(4, 1), 17*60+50),        // after next thu 17:50
				newSlots(nextWeekDay(0), 2*60+40),            // next SUN 2:40; or newSlots(nextWeekDay(6), 24*60+2*60+40)
			),
		},
		{
			Name:     "Dr. Barnett Mueller",
			Category: "Ophthalmologist",
			Subtitle: "6 years of experience",
			Details:  "Navy Street 1, Kiskimere, United States",
			SlotSize: 25,
			Price:    "$35",
			ImageURL: "https://snippet.dhtmlx.com/codebase/data/booking/01/img/02.jpg",
			Gap:      0,
			Review: Review{
				Count: 184,
				Stars: 3,
			},
			DoctorSchedule: []DoctorSchedule{
				// mon, wed, fri 9:00-17:00
				RecurringSchedule(9*60, 17*60, "MO,WE,FR"),
				// sat, sun 15:00-19:00
				RecurringSchedule(15*60, 19*60, "SA,SU"),
			},
			OccupiedSlots: genSlots(
				newSlots(nextWeekDay(1), 13*60+10),    // after next mon 13:10
				newSlots(nextWeekDay(1, 1), 12*60+45), // after next mon 12:45
				newSlots(nextWeekDay(3), 9*60+25),     // next wed 9:25
				newSlots(nextWeekDay(5), 11*60+55),    // next fri 11:55
				newSlots(nextWeekDay(5, 1), 11*60+30), // after next fri 11:30
				newSlots(nextWeekDay(6), 16*60+10),    // next sat 16:10
				newSlots(nextWeekDay(0), 17*60),       // next sun 17:00
			),
		},
		{
			Name:     "Dr. Myrtle Wise",
			Category: "Ophthalmologist",
			Subtitle: "4 years of experience",
			Details:  "Prescott Place 5, Freeburn, Bulgaria",
			SlotSize: 25,
			Price:    "$40",
			ImageURL: "https://snippet.dhtmlx.com/codebase/data/booking/01/img/01.jpg",
			Gap:      5,
			Review: Review{
				Count: 829,
				Stars: 5,
			},
			DoctorSchedule: append(
				[]DoctorSchedule{
					// tue, thu 7:00-15:00
					RecurringSchedule(7*60, 15*60, "TU,TH"),
					// sat, sun 11:00-15:00
					RecurringSchedule(11*60, 15*60, "SA,SU"),
				},
				// next fri, sat 4:00-8:00
				genSchedule(4*60, 8*60, nextWeekDay(5), 2)...,
			),
			OccupiedSlots: genSlots(
				newSlots(nextWeekDay(2), 7*60, 10*60),    // next tue 7:00, 10:00
				newSlots(nextWeekDay(4), 9*60+30),        // next thu 9:30
				newSlots(nextWeekDay(5), 7*60+30),        // next fri 7:30
				newSlots(nextWeekDay(6), 11*60+30, 5*60), // next sat 11:30, 5:00
				newSlots(nextWeekDay(0), 12*60),          // next sun 12:00
			),
		},
		{
			Name:     "Dr. Browning Peck",
			Category: "Dentist",
			Subtitle: "11 years of experience",
			SlotSize: 60,
			Details:  "Seacoast Terrace 174, Belvoir, Mauritania",
			Price:    "$175",
			ImageURL: "https://snippet.dhtmlx.com/codebase/data/booking/01/img/12.jpg",
			Gap:      10,
			Review: Review{
				Count: 391,
				Stars: 5,
			},
			DoctorSchedule: []DoctorSchedule{
				// thu, fri, sat, sun 9:00-17:00
				RecurringSchedule(9*60, 17*60, "TH,FR,SA,SU"),
			},
			OccupiedSlots: genSlots(
				newSlots(nextWeekDay(4), 11*60+20),       // next thu 11:20
				newSlots(nextWeekDay(5), 14*60+50),       // next fri 14:50
				newSlots(nextWeekDay(6), 9*60, 13*60+20), // next sat 9:00, 13:20
				newSlots(nextWeekDay(0), 14*60+50),       // next sun 14:50
			),
		},
	}

	err := tx.Create(doctors).Error
	if err != nil {
		panic(err)
	}
}
