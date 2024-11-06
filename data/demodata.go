package data

import (
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

	nextWeekDay := func(day int) int64 {
		next := (6+day-int(now.Weekday()))%7 + 1
		return time.Date(y, m, d+next, 0, 0, 0, 0, time.UTC).UnixMilli()
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
					// every week day 9:00-17:00 (sun, sat - holidays)
					RecurringSchedule(9*60, 17*60, "MO,TU,WE,TH,FR"),
				},
				// next tue, wed, thu 2:00-6:00
				genSchedule(2*60, 6*60, nextWeekDay(2), 3)...,
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
					// sat 20:00-4:00 (sun)
					RecurringSchedule(20*60, 4*60, "SA"), // and RecurringSchedule(20*60, 28*60, "SA")
				},
				// next wed 18:00-22:00
				genSchedule(18*60, 22*60, nextWeekDay(3), 1)...,
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
		},
	}

	err := tx.Create(doctors).Error
	if err != nil {
		panic(err)
	}
}
