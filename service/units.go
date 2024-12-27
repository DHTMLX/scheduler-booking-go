package service

import (
	"fmt"
	"log"
	"regexp"
	"scheduler-booking/data"
	"strconv"
	"strings"
	"time"
)

type unitsService struct {
	dao *data.DAO
}

type Unit struct {
	ID       int         `json:"id"`
	Title    string      `json:"title"`
	Category string      `json:"category"`
	Subtitle string      `json:"subtitle"`
	Details  string      `json:"details"`
	Preview  string      `json:"preview"`
	Price    string      `json:"price"`
	Review   data.Review `json:"review"`

	Slots          []Schedule `json:"slots"`
	AvailableSlots []int64    `json:"availableSlots,omitempty"`
	UsedSlots      []int64    `json:"usedSlots,omitempty"`
}

type Schedule struct {
	From  string  `json:"from"`
	To    string  `json:"to"`
	Size  int     `json:"size"`
	Gap   int     `json:"gap"`
	Days  []int   `json:"days,omitempty"`
	Dates []int64 `json:"dates,omitempty"`
}

const (
	allDay      = 24 * 60            // in minutes
	allDayMilli = 60 * 1000 * allDay // in millisecond
	timeLayout  = "15:04:00"
	dateLayout  = "2006-01-02"
)

var week = map[string]int{"SU": 0, "MO": 1, "TU": 2, "WE": 3, "TH": 4, "FR": 5, "SA": 6}

// create a new schedule
func newSchedule(from, to, size, gap int, days []int, dates []int64) *Schedule {
	return &Schedule{
		From:  m2t(from),
		To:    m2t(to),
		Size:  size,
		Gap:   gap,
		Days:  days,
		Dates: dates,
	}
}

func (s *unitsService) GetAll() ([]Unit, error) {
	doctors, err := s.dao.Doctors.GetAll(true)
	if err != nil {
		return nil, err
	}

	units := createUnits(doctors, true)
	return units, nil
}

func createUnits(doctors []data.Doctor, replace bool) []Unit {
	units := make([]Unit, len(doctors))
	for i, doctor := range doctors {
		slotsDays := make(map[int][]time.Time)    // search by days
		slotsDates := make(map[int64][]time.Time) // search by dates

		for _, slot := range doctor.OccupiedSlots {
			slotDate := time.UnixMilli(slot.Date).UTC()

			weekDay := int(slotDate.Weekday())
			slotsDays[weekDay] = append(slotsDays[weekDay], slotDate)

			date := slotDate.Truncate(24 * time.Hour).UnixMilli()
			slotsDates[date] = append(slotsDates[date], slotDate)
		}

		routines := []*data.DoctorSchedule{}
		extensions := make(map[string][]*data.DoctorSchedule) // recID -> []sch
		recurring := make(map[int][]*data.DoctorSchedule)     // weekDay of rec -> []sch
		deleted := make(map[int64]int)                        // timestamp -> from

		blockedSlots := make(map[int64]struct{})
		schedules := make([]Schedule, 0)

		// routine events
		for _, sch := range doctor.DoctorSchedule {
			if rout := sch.DoctorRoutine; rout != nil {
				if rout.RecurringEventID != "" {
					extensions[rout.RecurringEventID] = append(extensions[rout.RecurringEventID], &sch)
					continue
				}

				routines = append(routines, &sch)
			}
		}

		// regurring events
		for _, sch := range doctor.DoctorSchedule {
			if rec := sch.DoctorRecurringRoutine; rec != nil {
				date := time.UnixMilli(rec.Date).UTC()
				recID := strconv.Itoa(rec.ID)

				recDays := daysFromRules(rec.Rrule)

				exts := extensions[recID]
				clearedExtensions := make(map[int64]struct{}, len(exts))

				// check exts
				for _, sch := range exts {
					ext := sch.DoctorRoutine

					orig, err := time.Parse("2006-01-02 15:04", ext.OriginalStart)
					if err != nil {
						continue
					}

					h, m, _ := orig.Clock()
					origFrom := h*60 + m

					origDay := int(orig.Weekday())
					if _, ok := recDays[origDay]; ok && sch.From == origFrom {
						if !ext.Deleted {
							routines = append(routines, sch)
						}

						deleted[orig.UnixMilli()] = origFrom
						clearedExtensions[orig.UnixMilli()] = struct{}{}
					}
				}

				days := make([]int, 0, len(recDays))
				for day := range recDays {
					recurring[day] = append(recurring[day], &sch)

					// slots
					booked := getRecBookedSlots(slotsDays, day, date, sch.From, sch.To, doctor.SlotSize, doctor.Gap, clearedExtensions, replace)
					for _, slot := range booked {
						blockedSlots[slot] = struct{}{}
					}

					days = append(days, day)
				}

				// create empty schedules
				emptySchedules := createEmpty(days, date)
				for _, empty := range emptySchedules {
					deleted[empty] = sch.From
				}

				// create schedules
				newSchedules := createSchedules(sch.From, sch.To, doctor.SlotSize, doctor.Gap, days, nil)
				schedules = append(schedules, newSchedules...)
			}
		}

		checkDay := make(map[int64]struct{})
		for _, sch := range routines {
			rout := sch.DoctorRoutine

			date := time.UnixMilli(rout.Date).UTC()
			weekDay := int(date.Weekday())

			// create for recurring
			for _, recSch := range recurring[weekDay] {
				rec := recSch.DoctorRecurringRoutine
				if _, ok := checkDay[rout.Date]; !ok && rec.Date <= rout.Date {
					newSchedules := createSchedules(recSch.From, recSch.To, doctor.SlotSize, doctor.Gap, nil, []int64{rout.Date})
					schedules = append(schedules, newSchedules...)

					checkDay[rout.Date] = struct{}{}
					delete(deleted, rout.Date)
				}
			}

			if sch.To > allDay {
				// FIXME
			}

			// slots
			booked := getBookedSlots(slotsDates, rout.Date, sch.From, sch.To, doctor.SlotSize, doctor.Gap, replace)
			for _, slot := range booked {
				blockedSlots[slot] = struct{}{}
			}

			// schedules
			newSchedules := createSchedules(sch.From, sch.To, doctor.SlotSize, doctor.Gap, nil, []int64{rout.Date})
			schedules = append(schedules, newSchedules...)
		}

		// deleted
		for date, from := range deleted {
			newSchedules := createSchedules(from, from, doctor.SlotSize, doctor.Gap, nil, []int64{date})
			schedules = append(schedules, newSchedules...)
		}

		slots := make([]int64, 0, len(blockedSlots))
		for slot := range blockedSlots {
			slots = append(slots, slot)
		}

		units[i] = Unit{
			ID:        doctor.ID,
			Title:     doctor.Name,
			Subtitle:  doctor.Details,
			Details:   doctor.Subtitle,
			Category:  doctor.Category,
			Price:     doctor.Price,
			Review:    doctor.Review,
			Preview:   doctor.ImageURL,
			UsedSlots: slots,
			Slots:     schedules,
		}
	}

	return units
}

func daysFromRules(rrule string) map[int]struct{} {
	re := regexp.MustCompile(`BYDAY=([^;]+)`)
	matches := re.FindStringSubmatch(rrule)

	days := make(map[int]struct{})
	if len(matches) > 1 {
		weekDays := strings.Split(matches[1], ",")

		for _, weekDay := range weekDays {
			if num, ok := week[strings.ToUpper(weekDay)]; ok {
				days[num] = struct{}{}
			} else {
				log.Printf("WARN: invalid day abbreviation: %s", weekDay)
			}
		}
	}

	return days
}

func createSchedules(from, to, size, gap int, days []int, dates []int64) []Schedule {
	schedule := []Schedule{}
	if len(dates) == 0 && len(days) == 0 {
		// skip this rule as it is already expired
		return schedule
	}

	medium := to
	if medium > allDay {
		slot := size + gap
		remTo := (to - from) % slot
		rem := (to - remTo - allDay + slot) % slot
		medium = allDay + rem
	}

	sch := newSchedule(from, medium, size, gap, days, dates)
	schedule = append(schedule, *sch)

	if to > allDay && to-medium >= size {
		newDays := make([]int, len(days))
		newDates := make([]int64, len(dates))

		for i, day := range days {
			newDays[i] = (day + 1) % 7
		}

		for i, date := range dates {
			y, m, d := time.UnixMilli(date).UTC().Date()
			newDates[i] = newStamp(y, m, d+1, 0)
		}

		sch := newSchedule(medium-allDay, to-allDay, size, gap, newDays, newDates)
		schedule = append(schedule, *sch)
	}

	return schedule
}

func m2t(m int) string {
	// if m >= allDay {
	// 	return "24:00"
	// }

	hours := m / 60
	minutes := m % 60
	return fmt.Sprintf("%d:%02d", hours, minutes)
}

// year, month, day, min
func newStamp(y int, m time.Month, d, min int) int64 {
	return time.Date(y, m, d, 0, min, 0, 0, time.UTC).UnixMilli()
}

func getBookedSlots(slots map[int64][]time.Time, date int64, from, to, size, gap int, replace bool) []int64 {
	current := time.UnixMilli(date).UTC()
	y, m, d := current.Date()

	slotsDate := slots[date-allDayMilli]                      // prev day
	slotsDate = append(slotsDate, slots[date]...)             // today
	slotsDate = append(slotsDate, slots[date+allDayMilli]...) // next day

	segment := size + gap
	newTo := to - (to-from)%segment
	if newTo+size <= to {
		newTo += segment
	}

	bookedSlots := make([]int64, 0, len(slotsDate)*2)
	for _, slot := range slotsDate {
		ts := int(slot.Sub(current).Minutes())
		if !replace {
			if from <= ts && ts+size <= to {
				bookedSlots = append(bookedSlots, newStamp(y, m, d, ts))
			}
			continue
		}

		rem := (segment + (ts-from)%segment) % segment

		before := ts - rem
		if from < before+segment && before < newTo {
			bookedSlots = append(bookedSlots, newStamp(y, m, d, before))
		}

		if rem != 0 {
			after := ts + segment - rem
			if from < after+segment && after < newTo {
				bookedSlots = append(bookedSlots, newStamp(y, m, d, after))
			}
		}
	}

	return bookedSlots
}

func getRecBookedSlots(slots map[int][]time.Time, day int, date time.Time, from, to, size, gap int, exts map[int64]struct{}, replace bool) []int64 {
	// slotsDate := slots[day]

	slotsDate := slots[(day+6)%7]                      // prev day
	slotsDate = append(slotsDate, slots[day]...)       // current
	slotsDate = append(slotsDate, slots[(day+1)%7]...) // next day

	segment := size + gap
	newTo := to - (to-from)%segment
	if newTo+size <= to {
		newTo += segment
	}

	newDate := date.Add(-time.Duration(segment) * time.Minute)
	bookedSlots := make([]int64, 0, len(slotsDate))
	for _, slot := range slotsDate {
		if newDate.Before(slot) {
			y, m, d := slot.Date()

			weekDay := int(slot.Weekday())
			diff := day - weekDay
			current := time.Date(y, m, d+diff, 0, from, 0, 0, time.UTC)
			if _, ok := exts[current.UnixMilli()]; ok {
				continue
			}

			ts := int(slot.Sub(current).Minutes())
			if !replace {
				if from <= ts && ts+size <= to {
					bookedSlots = append(bookedSlots, newStamp(y, m, d, ts))
				}
				continue
			}

			rem := (segment + (ts-from)%segment) % segment

			before := ts - rem
			if from < before+segment && before < newTo {
				bookedSlots = append(bookedSlots, newStamp(y, m, d, before))
			}

			if rem != 0 {
				after := ts + segment - rem
				if from < after+segment && after < newTo {
					bookedSlots = append(bookedSlots, newStamp(y, m, d, after))
				}
			}
		}
	}

	return bookedSlots
}

func createEmpty(days []int, date time.Time) []int64 {
	today := time.Now().UTC()
	today = today.Add(-12 * time.Hour) // for demo

	if !date.After(today) {
		return nil
	}

	rY, rM, rD := date.Date()
	weekDay := int(today.Weekday())

	emties := make([]int64, 0)
	for _, day := range days {
		next := (7 + day - weekDay) % 7
		for nextDate := time.Date(rY, rM, rD+next, 0, 0, 0, 0, time.UTC); nextDate.Before(date); nextDate = nextDate.Add(7 * 24 * time.Hour) {
			emties = append(emties, nextDate.UnixMilli())
		}
	}

	return emties
}
