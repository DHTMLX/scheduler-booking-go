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
	allDay     = 24 * 60 // in minutes
	timeLayout = "15:04:00"
	dateLayout = "2006-01-02"
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
	now := time.Now().UTC()
	ny, nm, nd := now.Date()

	units := make([]Unit, len(doctors))
	for i, d := range doctors {
		slots := make(map[int][]time.Time) // search by days
		for _, slot := range d.OccupiedSlots {
			slotDate := time.UnixMilli(slot.Date).UTC()
			weekDay := int(slotDate.Weekday())
			slots[weekDay] = append(slots[weekDay], slotDate)
		}

		availableSlots := make(map[int64][]int64) // date = array[timestamp]

		schedules := make([]Schedule, 0)
		originals := make(map[int64]bool) // timestamps originalStart

		recMap := make(map[string]*data.DoctorRecurringRoutine) // map for recurring
		recDays := make(map[int][]*data.DoctorRecurringRoutine) // map for days recurring

		extensions := []data.DoctorRoutine{}
		routines := []data.DoctorRoutine{}

		// sort
		for _, sch := range d.DoctorSchedule {
			if rec := sch.DoctorRecurringRoutine; rec != nil {
				rec.DoctorSchedule.From = sch.From
				rec.DoctorSchedule.To = sch.To

				days := daysFromRules(rec.Rrule)
				newSchedules := createSchedules(sch.From, sch.To, d.SlotSize, d.Gap, days, nil)
				schedules = append(schedules, newSchedules...)

				recMap[strconv.Itoa(sch.ID)] = rec // for quick search
				for _, day := range days {
					recDays[day] = append(recDays[day], rec) // for quick search by days
				}

				// if rec event is created in the future and has weekdays less than now.WeekDay
				recDate := time.UnixMilli(rec.Date).UTC()
				rY, rM, rD := recDate.Date()

				for i, nsch := range newSchedules {
					from := sch.From
					if nsch.From == "0:00" {
						from = 0
					}

					nRecDate := time.Date(rY, rM, rD+i, 0, from, 0, 0, time.UTC)
					for date := now; date.Before(nRecDate); date = date.AddDate(0, 0, 7) {
						for _, day := range days {
							next := (7 + day - int(date.Weekday()) + i) % 7
							stamp := newStamp(ny, nm, nd+next, from)

							if now.UnixMilli() <= stamp && stamp < nRecDate.UnixMilli() {
								originals[stamp] = true
							}
						}
					}
				}

				// slots
				for _, day := range days {
					for _, date := range slots[day] {
						if recDate.Before(date.Add(time.Duration(d.SlotSize+d.Gap) * time.Minute)) {
							newStamps := getTimestamps(&date, sch.From, sch.To, d.SlotSize, d.Gap, replace)
							if len(newStamps) > 0 {
								nDate := date.Truncate(24 * time.Hour).UnixMilli()
								log.Print(nDate, " ", newStamps) // Add(time.Duration((sch.From - stamp)) * time.Minute).UnixMilli() // only date
								availableSlots[nDate] = append(availableSlots[nDate], newStamps...)
							}
						}
					}

					if sch.To > allDay {
						for _, date := range slots[(day+1)%7] {
							if recDate.Before(date.Add(time.Duration(d.SlotSize+d.Gap) * time.Minute)) {
								newStamps := getTimestamps(&date, 0, sch.To-allDay, d.SlotSize, d.Gap, replace)
								if len(newStamps) > 0 {
									nDate := date.Truncate(24 * time.Hour).UnixMilli() // only date
									availableSlots[nDate] = append(availableSlots[nDate], newStamps...)
								}
							}
						}
					}
				}
			}

			if rout := sch.DoctorRoutine; rout != nil {
				rout.DoctorSchedule.From = sch.From
				rout.DoctorSchedule.To = sch.To

				if rout.RecurringEventID != "" {
					extensions = append(extensions, *rout)
				} else {
					routines = append(routines, *rout)
				}
			}
		}

		for _, ext := range extensions {
			origStart, err := time.Parse("2006-01-02 15:04", ext.OriginalStart)
			if err != nil {
				log.Print(err)
				continue
			}

			if rec, ok := recMap[ext.RecurringEventID]; ok {
				weekDays := daysFromRules(rec.Rrule)
				origDay := int(origStart.Weekday())

				for _, weekDay := range weekDays {
					if origDay == weekDay {
						if time.UnixMilli(rec.Date).UTC().Format(timeLayout) == origStart.Format(timeLayout) {
							if !ext.Deleted {
								routines = append(routines, ext)
							}
							originals[origStart.UnixMilli()] = true
							origStart = origStart.Truncate(24 * time.Hour)
							delete(availableSlots, origStart.UnixMilli())

							// if the parent recurring event was more than a day
							if rec.DoctorSchedule.To > allDay {
								y, m, d := origStart.Date()
								stamp := newStamp(y, m, d+1, 0)
								originals[stamp] = true
								delete(availableSlots, stamp)
							}
						}

						break
					}
				}
			}
		}

		verified := make(map[string]bool)
		for _, rout := range routines {
			newSchedules := createSchedules(rout.DoctorSchedule.From, rout.DoctorSchedule.To, d.SlotSize, d.Gap, nil, []int64{rout.Date})
			schedules = append(schedules, newSchedules...)

			for _, sch := range newSchedules {
				for _, date := range sch.Dates {
					onlyDate := time.UnixMilli(date).UTC().Format(dateLayout)
					if !verified[onlyDate] {
						addSchedules := additionalEvents(recDays, date, originals, d.SlotSize, d.Gap)
						schedules = append(schedules, addSchedules...)

						verified[onlyDate] = true
					}
				}
			}

			for _, sch := range newSchedules {
				from := rout.DoctorSchedule.From
				to := rout.DoctorSchedule.To
				if to > allDay {
					from = 0
					to -= allDay
				}

				for _, date := range sch.Dates {
					schDate := time.UnixMilli(date).UTC()
					weekDay := int(schDate.Weekday())

					for _, slotDate := range slots[weekDay] {
						if schDate.Before(slotDate.Add(time.Duration(d.SlotSize+d.Gap) * time.Minute)) {
							newStamps := getTimestamps(&slotDate, from, to, d.SlotSize, d.Gap, replace)
							if len(newStamps) > 0 {
								nDate := schDate.Truncate(24 * time.Hour).UnixMilli()
								availableSlots[nDate] = append(availableSlots[nDate], newStamps...)
							}
						}
					}
				}
			}
		}

		for date := range originals {
			addSchedules := []Schedule{}

			onlyDate := time.UnixMilli(date).UTC().Format(dateLayout)
			if !verified[onlyDate] {
				addSchedules = additionalEvents(recDays, date, originals, d.SlotSize, d.Gap)
				schedules = append(schedules, addSchedules...)

				verified[onlyDate] = true
			}

			if len(addSchedules) == 0 {
				newSchedules := createSchedules(0, 0, 0, 0, nil, []int64{date})
				schedules = append(schedules, newSchedules...)
			}
		}

		usedSlots := []int64{}
		for _, values := range availableSlots {
			usedSlots = append(usedSlots, values...)
		}

		units[i] = Unit{
			ID:        d.ID,
			Title:     d.Name,
			Subtitle:  d.Details,
			Details:   d.Subtitle,
			Category:  d.Category,
			Price:     d.Price,
			Review:    d.Review,
			Preview:   d.ImageURL,
			UsedSlots: usedSlots,
			Slots:     schedules,
		}
	}

	// new
	for _, doctor := range doctors {
		slotsDays := make(map[int][]time.Time)    // search by days
		slotsDates := make(map[int64][]time.Time) // search by dates

		for _, slot := range doctor.OccupiedSlots {
			slotDate := time.UnixMilli(slot.Date).UTC()

			weekDay := int(slotDate.Weekday())
			slotsDays[weekDay] = append(slotsDays[weekDay], slotDate)

			date := slotDate.Truncate(24 * time.Hour).UnixMilli()
			slotsDates[date] = append(slotsDates[date], slotDate)
		}

		// daysPriority := make(map[int64]bool)
		// extensions := make(map[int64]int)              // timestamp -> from
		deleted := make(map[string]int64)                 // timestamp -> from
		original := make(map[string][]data.DoctorRoutine) // timestamp -> []rout

		availableSlots := make(map[int64]struct{}) // date = array[timestamp]
		availableSlots := make(map[int64]struct{})
		schedules := make([]Schedule, 0)

		for _, sch := range doctor.DoctorSchedule {
			if rout := sch.DoctorRoutine; rout != nil {
				if rout.Deleted {
					deleted[rout.RecurringEventID] = rout.Date
					continue
				}

				if rout.RecurringEventID != "" {
					original[rout.RecurringEventID] = append(original[rout.RecurringEventID], *rout)
					continue
				}

				booked := getBookedSlots(slotsDates, rout.Date, sch.From, sch.To, doctor.SlotSize, doctor.Gap, replace)
				for _, slot := range booked {
					availableSlots[slot] = struct{}{}
				}

				// schedules
				newSchedules := createSchedules(sch.From, sch.To, doctor.SlotSize, doctor.Gap, nil, []int64{rout.Date})
				schedules = append(schedules, newSchedules...)
			}
		}
	}

	return units
}

func daysFromRules(rrule string) []int {
	re := regexp.MustCompile(`BYDAY=([^;]+)`)
	matches := re.FindStringSubmatch(rrule)

	days := []int{}
	if len(matches) > 1 {
		weekDays := strings.Split(matches[1], ",")

		for _, weekDay := range weekDays {
			if num, ok := week[strings.ToUpper(weekDay)]; ok {
				days = append(days, num)
			} else {
				log.Printf("WARN: invalid day abbreviation: %s", weekDay)
			}
		}
	}

	return days
}

func daysMapsFromRules(rrule string) map[int]struct{} {
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

func additionalEvents(recurring map[int][]*data.DoctorRecurringRoutine, date int64, originals map[int64]bool, size, gap int) []Schedule {
	timeDate := time.UnixMilli(date).UTC()
	y, m, d := timeDate.Date()
	origDay := int(timeDate.Weekday())

	schedule := []Schedule{}
	for _, rec := range recurring[origDay] {
		from := rec.DoctorSchedule.From
		to := rec.DoctorSchedule.To

		stamp := newStamp(y, m, d, from)

		if !originals[stamp] {
			sch := newSchedule(from, to, size, gap, []int{}, []int64{stamp})
			schedule = append(schedule, *sch)
		} else {
			delete(originals, stamp)
		}
	}

	for _, rec := range recurring[(6+origDay)%7] {
		if rec.DoctorSchedule.To > allDay {
			stamp := newStamp(y, m, d, 0)

			if !originals[stamp] {
				sch := newSchedule(0, rec.DoctorSchedule.To-allDay, size, gap, []int{}, []int64{stamp})
				schedule = append(schedule, *sch)
			} else {
				delete(originals, stamp)
			}
		}
	}

	return schedule
}

func getTimestamps(date *time.Time, from, to, size, gap int, replace bool) []int64 {
	var (
		y, m, d   = date.Date()
		h, min, _ = date.Clock()
		slot      = size + gap
		stamp     = h*60 + min
		tsRem     = stamp % slot
		rem       = from % slot
	)

	stamps := make([]int64, 0, 2)
	addStamp := func(ts int) {
		if from <= ts && ts+size <= to {
			stamps = append(stamps, newStamp(y, m, d, ts))
		}
	}

	if rem == tsRem || !replace {
		addStamp(stamp)
	} else {
		prev := (tsRem - rem + slot) % slot

		first := stamp - prev
		second := stamp + slot - prev

		addStamp(first)
		addStamp(second)
	}

	return stamps
}

func getBookedSlots(slots map[int64][]time.Time, date int64, from, to, size, gap int, replace bool) []int64 {
	today := time.UnixMilli(date).UTC()
	y, m, d := today.Date()

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
		ts := int(slot.Sub(today).Minutes())
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
