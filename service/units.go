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
	allDay      = 24 * 60              // in minutes
	minuteMilli = 60 * 1000            // in millisecond
	allDayMilli = minuteMilli * allDay // in millisecond
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

		routines := make(map[int][]data.DoctorSchedule)      // week day -> []sch
		extensions := make(map[string][]data.DoctorSchedule) // recID    -> []sch
		empty := make(map[int]map[int64]struct{})            // week day -> map timestamp

		bookedSlots := make(map[int64]struct{})
		schedules := make([]Schedule, 0)

		// separation of routine events
		for _, sch := range doctor.DoctorSchedule {
			if rout := sch.DoctorRoutine; rout != nil {
				if rout.RecurringEventID != "" {
					extensions[rout.RecurringEventID] = append(extensions[rout.RecurringEventID], sch)
					continue
				}

				day := int(time.UnixMilli(rout.Date).UTC().Weekday())
				routines[day] = append(routines[day], sch)
			}
		}

		// recurring events
		for _, recSch := range doctor.DoctorSchedule {
			if rec := recSch.DoctorRecurringRoutine; rec != nil {
				recID := strconv.Itoa(recSch.ID)
				recDays := daysFromRules(rec.Rrule)
				exts := extensions[recID]

				empty[recSch.ID] = make(map[int64]struct{})

				// check exts
				for _, extSch := range exts {
					ext := extSch.DoctorRoutine

					original, err := time.Parse("2006-01-02 15:04", ext.OriginalStart)
					if err != nil {
						continue
					}

					h, m, _ := original.Clock()
					origFrom := h*60 + m

					origDay := int(original.Weekday())
					for _, recDay := range recDays {
						if recDay == origDay && recSch.From == origFrom {
							if !ext.Deleted {
								day := int(time.UnixMilli(ext.Date).UTC().Weekday())
								routines[day] = append(routines[day], extSch)
							}

							// deleted
							origDate := original.Truncate(24 * time.Hour).UnixMilli()
							emptySch := data.DoctorSchedule{
								From: recSch.From,
								To:   recSch.To,
								DoctorRoutine: &data.DoctorRoutine{
									Date:             origDate,
									Deleted:          true,
									RecurringEventID: recID,
								},
							}

							empty[recSch.ID][original.UnixMilli()] = struct{}{}

							routines[origDay] = append(routines[origDay], emptySch)
						}
					}
				}

				date := time.UnixMilli(rec.Date).UTC()
				for day := range recDays {
					// slots
					booked := getRecBookedSlots(slotsDays, day, date, recSch.From, recSch.To, doctor.SlotSize, doctor.Gap, empty[recSch.ID], replace)
					for _, slot := range booked {
						bookedSlots[slot] = struct{}{}
					}
				}

				// create empty schedules
				emptySchedules := createEmpty(recDays, date)
				for i, emptyDate := range emptySchedules {
					sch := data.DoctorSchedule{
						From: recSch.From,
						To:   recSch.To,
						DoctorRoutine: &data.DoctorRoutine{
							Date:             emptyDate,
							Deleted:          true,
							RecurringEventID: recID,
						},
					}

					empty[recSch.ID][timeStamp(emptyDate, recSch.From)] = struct{}{}

					day := recDays[i%len(recDays)]
					routines[day] = append(routines[day], sch)
				}
			}
		}

		taken := make(map[int][]int64)
		for _, routSchs := range routines {
			for _, routSch := range routSchs {
				if rout := routSch.DoctorRoutine; rout != nil {
					// slots
					booked := getBookedSlots(slotsDates, rout.Date, routSch.From, routSch.To, doctor.SlotSize, doctor.Gap, replace)
					for _, slot := range booked {
						bookedSlots[slot] = struct{}{}
					}

					// schedules
					newSchedules := createSchedules(routSch.From, routSch.To, doctor.SlotSize, doctor.Gap, nil, []int64{rout.Date})

					for i, sch := range newSchedules {
						for _, date := range sch.Dates {
							weekDay := int(time.UnixMilli(date).UTC().Weekday())
							taken[weekDay] = append(taken[weekDay], date)
						}

						if rout.Deleted {
							newSchedules[i].To = newSchedules[i].From

							recID, _ := strconv.Atoi(rout.RecurringEventID)
							empty[recID][timeStamp(sch.Dates[0], routSch.From*(1-i))] = struct{}{}
						}
					}

					schedules = append(schedules, newSchedules...)
				}
			}
		}

		for _, recSch := range doctor.DoctorSchedule {
			if rec := recSch.DoctorRecurringRoutine; rec != nil {
				days := daysFromRules(rec.Rrule)

				// create schedules
				newSchedules := createSchedules(recSch.From, recSch.To, doctor.SlotSize, doctor.Gap, days, nil)
				log.Print(newSchedules)
				deleted := empty[recSch.ID]

				for i, sch := range newSchedules {
					dates := make(map[int64]struct{})

					for _, day := range sch.Days {
						for _, date := range taken[day] {
							if date > rec.Date-7*allDayMilli { // fixme
								if _, ok := deleted[timeStamp(date, recSch.From*(1-i))]; !ok {
									dates[date] = struct{}{}
								}
							}
						}
					}

					datesSlice := make([]int64, 0, len(dates))
					for date := range dates {
						datesSlice = append(datesSlice, date)
					}

					newSchedules[i].Dates = datesSlice
				}

				schedules = append(schedules, newSchedules...)
			}
		}

		usedSlots := make([]int64, 0, len(bookedSlots))
		for slot := range bookedSlots {
			usedSlots = append(usedSlots, slot)
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
			UsedSlots: usedSlots,
			Slots:     schedules,
		}
	}

	return units
}

func daysFromRulesMap(rrule string) map[int]struct{} {
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

func daysFromRules(rrule string) []int {
	re := regexp.MustCompile(`BYDAY=([^;]+)`)
	matches := re.FindStringSubmatch(rrule)
	if len(matches) == 1 {
		return []int{}
	}

	weekDays := strings.Split(matches[1], ",")
	days := make([]int, 0, len(weekDays))
	for _, weekDay := range weekDays {
		if num, ok := week[strings.ToUpper(weekDay)]; ok {
			days = append(days, num)
		} else {
			log.Printf("WARN: invalid day abbreviation: %s", weekDay)
		}
	}

	return days
}

func createSchedules(from, to, size, gap int, days []int, dates []int64) []Schedule {
	schedules := make([]Schedule, 0, 2)
	if len(dates) == 0 && len(days) == 0 {
		// skip this rule as it is already expired
		return schedules
	}

	median := to
	segment := size + gap
	if allDay+size <= to {
		rem := (allDay - from) % segment
		median = allDay + (segment-rem)%segment
	}

	sch := newSchedule(from, median, size, gap, days, dates)
	schedules = append(schedules, *sch)

	if median+size <= to {
		newDays := make([]int, len(days))
		newDates := make([]int64, len(dates))

		for i, day := range days {
			newDays[i] = (day + 1) % 7
		}

		for i, date := range dates {
			newDates[i] = date + allDayMilli
		}

		sch := newSchedule(median-allDay, to-allDay, size, gap, newDays, newDates)
		schedules = append(schedules, *sch)
	}

	return schedules
}

func m2t(m int) string {
	hours := m / 60
	minutes := m % 60
	return fmt.Sprintf("%02d:%02d", hours, minutes)
}

// year, month, day, min
func newStamp(y int, m time.Month, d, min int) int64 {
	return time.Date(y, m, d, 0, min, 0, 0, time.UTC).UnixMilli()
}

func timeStamp(date int64, from int) int64 {
	return date + int64(from*minuteMilli)
}

func getBookedSlots(slots map[int64][]time.Time, date int64, from, to, size, gap int, replace bool) []int64 {
	if from+size > to {
		return []int64{}
	}

	current := time.UnixMilli(date).UTC()
	y, m, d := current.Date()

	slotsDate := slots[date-allDayMilli]                      // prev day
	slotsDate = append(slotsDate, slots[date]...)             // current
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
			diff := (7 + day - weekDay) % 7

			current := time.Date(y, m, d+diff, 0, 0, 0, 0, time.UTC)
			if _, ok := exts[timeStamp(current.UnixMilli(), from)]; ok {
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
	today := data.Now()

	if !date.After(today) {
		return nil
	}

	y, m, d := today.Date()
	weekDay := int(today.Weekday())

	empty := make([]int64, 0)
	for _, day := range days {
		next := (7 + day - weekDay) % 7
		for nextDate := time.Date(y, m, d+next, 0, 0, 0, 0, time.UTC); nextDate.Before(date); nextDate = nextDate.Add(7 * 24 * time.Hour) {
			empty = append(empty, nextDate.UnixMilli())
		}
	}

	return empty
}
