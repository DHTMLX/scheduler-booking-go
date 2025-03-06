package service

import (
	"log"
	"regexp"
	"scheduler-booking/common"
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

// booking schedule
type Schedule struct {
	From  common.JTime `json:"from"`
	To    common.JTime `json:"to"`
	Size  int          `json:"size"`
	Gap   int          `json:"gap"`
	Days  []int        `json:"days,omitempty"`
	Dates []int64      `json:"dates,omitempty"`
}

const (
	allDay      = 24 * 60              // in minutes
	minuteMilli = 60 * 1000            // in millisecond
	allDayMilli = allDay * minuteMilli // in millisecond

	oneDay = 24 * time.Hour
)

var week = map[string]int{"SU": 0, "MO": 1, "TU": 2, "WE": 3, "TH": 4, "FR": 5, "SA": 6}

func (s *unitsService) GetAll() ([]Unit, error) {
	doctors, err := s.dao.Doctors.GetAll(true)
	if err != nil {
		return nil, err
	}

	return createUnits(doctors, true), nil
}

func createUnits(doctors []data.Doctor, replace bool) []Unit {
	today := data.DateNow() // date only
	todayMilli := today.UnixMilli()
	tWeekDay := int(today.Weekday())

	units := make([]Unit, len(doctors))
	for i, doctor := range doctors {
		slotsDays := make(map[int][]time.Time)    // search by days
		slotsDates := make(map[int64][]time.Time) // search by dates

		// organization occupied slots
		for _, occupiedSlot := range doctor.OccupiedSlots {
			slot := time.UnixMilli(occupiedSlot.Date).UTC()

			slotDay := int(slot.Weekday())
			slotsDays[slotDay] = append(slotsDays[slotDay], slot)

			slotDate := slot.Truncate(oneDay).UnixMilli()
			slotsDates[slotDate] = append(slotsDates[slotDate], slot)
		}

		routines := make([]data.DoctorSchedule, 0, len(doctor.DoctorSchedule))  // []sch
		recurring := make([]data.DoctorSchedule, 0, len(doctor.DoctorSchedule)) // []sch
		extensions := make(map[string][]data.DoctorSchedule)                    // recID -> []sch
		empty := make(map[string]map[int64]struct{})                            // recID -> map timestamp

		bookedSlots := make(map[int64]struct{})
		schedules := make([]Schedule, 0)

		// separation of events
		for _, sch := range doctor.DoctorSchedule {
			if sch.Rrule == "" {
				if sch.RecurringEventID != "" {
					extensions[sch.RecurringEventID] = append(extensions[sch.RecurringEventID], sch)
					continue
				}

				routines = append(routines, sch)
			} else {
				recurring = append(recurring, sch)
			}
		}

		// extensions events
		for _, recSch := range recurring {
			recID := strconv.Itoa(recSch.ID)
			recDays := daysFromRules(recSch.Rrule)

			// check extensions
			empty[recID] = make(map[int64]struct{})
			for _, extSch := range extensions[recID] {
				original, err := time.Parse("2006-01-02 15:04", extSch.OriginalStart)
				if err != nil {
					log.Printf("failed to parse original start time: %v", err)
					continue
				}

				origFrom := original.Hour()*60 + original.Minute()
				origDate := original.Truncate(oneDay).UnixMilli()

				if recSch.From == origFrom && recSch.Date <= origDate {
					origDay := int(original.Weekday())
					for _, recDay := range recDays {
						if recDay == origDay {
							// extension
							if !extSch.Deleted {
								routines = append(routines, extSch)
							}

							// deleted
							emptySch := createEmpty(recSch.From, recSch.To, origDate, recID)
							routines = append(routines, *emptySch)
							break
						}
					}
				}
			}

			// empty schedules
			for _, day := range recDays {
				offset := (day - tWeekDay) % 7
				for date := newStamp(todayMilli, offset*allDay); date < recSch.Date; date = newStamp(date, 7*allDay) {
					// deleted
					emptySch := createEmpty(recSch.From, recSch.To, date, recID)
					routines = append(routines, *emptySch)
				}
			}
		}

		weekDates := make(map[int][]int64, 7)   // dates by days of the week
		activeDates := make(map[int64]struct{}) // dates without deleted

		// routine events
		for _, routSch := range routines {
			if todayMilli >= newStamp(routSch.Date, routSch.To) {
				continue
			}

			// booked slots
			if !routSch.Deleted {
				booked := getRoutBookedSlots(slotsDates, routSch.Date, routSch.From, routSch.To, doctor.SlotSize, doctor.Gap, replace)
				for _, slot := range booked {
					bookedSlots[slot] = struct{}{}
				}
			}

			// create schedules
			newSchedules := createSchedules(routSch.From, routSch.To, doctor.SlotSize, doctor.Gap, nil, []int64{routSch.Date})
			for _, sch := range newSchedules {
				date := sch.Dates[0]

				weekDay := int(time.UnixMilli(date).UTC().Weekday())
				weekDates[weekDay] = append(weekDates[weekDay], date)

				if routSch.Deleted {
					empty[routSch.RecurringEventID][newStamp(date, sch.From.Get())] = struct{}{}
				} else {
					activeDates[date] = struct{}{}
					schedules = append(schedules, sch)
				}
			}
		}

		// recurring events
		for _, recSch := range recurring {
			recID := strconv.Itoa(recSch.ID)
			recDays := daysFromRules(recSch.Rrule)

			// booked slots
			booked := getRecBookedSlots(slotsDays, recDays, recSch.Date, recSch.From, recSch.To, doctor.SlotSize, doctor.Gap, empty[recID], replace)
			for _, slot := range booked {
				bookedSlots[slot] = struct{}{}
			}

			// create schedules
			deleted := empty[recID]
			newSchedules := createSchedules(recSch.From, recSch.To, doctor.SlotSize, doctor.Gap, recDays, nil)
			for _, sch := range newSchedules {
				from := sch.From.Get()

				// additional for recurring
				sch.Dates = additionalDates(sch.Days, weekDates, deleted, from)
				schedules = append(schedules, sch)

				// empty for recurring
				emptyDates := emptyDates(deleted, activeDates, from)
				if len(emptyDates) > 0 {
					emptySch := newSchedule(from, from, sch.Size, sch.Gap, []int{}, emptyDates)
					schedules = append(schedules, *emptySch)
				}
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

// helper functions

func daysFromRules(rrule string) []int {
	re := regexp.MustCompile(`BYDAY=([^;]+)`)
	matches := re.FindStringSubmatch(rrule)
	if len(matches) < 2 {
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

func createEmpty(from, to int, date int64, recID string) *data.DoctorSchedule {
	return &data.DoctorSchedule{
		From:             from,
		To:               to,
		Date:             date,
		Deleted:          true,
		RecurringEventID: recID,
	}
}

// booked slots

func getRoutBookedSlots(slots map[int64][]time.Time, date int64, from, to, size, gap int, replace bool) []int64 {
	return getBookedSlots(slots, nil, []int{-1}, date, from, to, size, gap, nil, replace)
}

func getRecBookedSlots(slots map[int][]time.Time, days []int, date int64, from, to, size, gap int, exts map[int64]struct{}, replace bool) []int64 {
	return getBookedSlots(nil, slots, days, date, from, to, size, gap, exts, replace)
}

func getBookedSlots(slotsDates map[int64][]time.Time, slotsDays map[int][]time.Time, days []int, date int64, from, to, size, gap int, exts map[int64]struct{}, replace bool) []int64 {
	if from+size > to {
		return []int64{}
	}

	current := time.UnixMilli(date).UTC()
	currentDate := date

	segment := size + gap
	newTo := to - (to-from)%segment

	if newTo+size <= to {
		newTo += segment
	}

	prev := from-segment < 0 // prev day
	next := newTo > allDay   // next day

	var exists bool
	bookedSlots := make([]int64, 0, len(slotsDates))
	for _, day := range days {
		slots := getSlots(slotsDates, slotsDays, day, date, prev, next)
		for _, slot := range slots {
			if exts != nil {
				current, currentDate, exists = checkExtension(day, from, slot, exts)
				if exists {
					continue
				}
			}

			ts := int(slot.Sub(current).Minutes())
			if !replace {
				// for client reservation
				if from <= ts && ts+size <= to {
					bookedSlots = append(bookedSlots, newStamp(currentDate, ts))
				}
				continue
			}

			// for booking
			rem := (segment + (ts-from)%segment) % segment

			before := ts - rem
			if from < before+segment && before < newTo {
				bookedSlots = append(bookedSlots, newStamp(currentDate, before))
			}

			if rem != 0 {
				after := ts + segment - rem
				if from < after+segment && after < newTo {
					bookedSlots = append(bookedSlots, newStamp(currentDate, after))
				}
			}
		}
	}

	return bookedSlots
}

func getSlots(slotsDates map[int64][]time.Time, slotsDays map[int][]time.Time, day int, date int64, prev, next bool) []time.Time {
	if len(slotsDates) > 0 {
		slots := slotsDates[date]
		if prev {
			slots = append(slots, slotsDates[date-allDayMilli]...)
		}
		if next {
			slots = append(slots, slotsDates[date+allDayMilli]...)
		}

		return slots
	}

	if len(slotsDays) > 0 {
		slots := slotsDays[day]
		if prev {
			slots = append(slots, slotsDays[(day+6)%7]...)
		}
		if next {
			slots = append(slots, slotsDays[(day+1)%7]...)
		}

		return slots
	}

	return nil
}

func newStamp(date int64, from int) int64 {
	return date + int64(from*minuteMilli)
}

func checkExtension(day, from int, slot time.Time, exts map[int64]struct{}) (time.Time, int64, bool) {
	diff := day - int(slot.Weekday())

	current := slot.Truncate(oneDay).AddDate(0, 0, diff)
	currentDate := current.UnixMilli()

	_, exists := exts[newStamp(currentDate, from)]
	return current, currentDate, exists
}

// booking schedules for events

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

func newSchedule(from, to, size, gap int, days []int, dates []int64) *Schedule {
	return &Schedule{
		From:  common.NewJTime(from),
		To:    common.NewJTime(to),
		Size:  size,
		Gap:   gap,
		Days:  days,
		Dates: dates,
	}
}

// helper dates for recurring event

func additionalDates(days []int, weekDates map[int][]int64, deleted map[int64]struct{}, from int) []int64 {
	dates := make(map[int64]struct{}) // duplication of routines events

	for _, day := range days {
		for _, date := range weekDates[day] {
			if _, exists := deleted[newStamp(date, from)]; !exists {
				dates[date] = struct{}{}
			}
		}
	}

	addDates := make([]int64, 0, len(dates))
	for date := range dates {
		addDates = append(addDates, date)
	}

	return addDates
}

func emptyDates(deleted, activeDates map[int64]struct{}, from int) []int64 {
	emptyDates := make([]int64, 0, len(deleted))
	for stamp := range deleted {
		empty := time.UnixMilli(stamp).UTC()

		emptyFrom := empty.Hour()*60 + empty.Minute()
		emptyDate := empty.Truncate(oneDay).UnixMilli()

		if _, exists := activeDates[emptyDate]; !exists && from == emptyFrom {
			emptyDates = append(emptyDates, emptyDate)
		}
	}

	return emptyDates
}
