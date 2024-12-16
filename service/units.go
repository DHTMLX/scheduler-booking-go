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

func createUnits(doctors []data.Doctor, rep bool) []Unit {
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

		availableSlots := make(map[int64][]int64) // dat = array[timestamp]

		schedules := make([]Schedule, 0)
		originals := make(map[int64]bool) // timestamps originalStart

		recMap := make(map[string]*data.DoctorRecurringRoutine) // map for recurring
		recDays := make(map[int][]*data.DoctorRecurringRoutine) // map for days recurring
		extensions := []data.DoctorRoutine{}
		routines := []data.DoctorRoutine{}

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

					nRecDate := time.Date(rY, rM, rD+i, 0, from, 0, 0, now.Location())
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
						if !recDate.After(date) {
							h, m, _ := date.Clock()
							stamp := h*60 + m

							if sch.From <= stamp && stamp+d.SlotSize <= sch.To {
								nDate := date.Add(time.Duration((sch.From - stamp)) * time.Minute).UnixMilli() // only date
								newStamps := TimeStamps(&date, stamp, sch.From, sch.To, d.SlotSize, d.Gap, rep)
								availableSlots[nDate] = append(availableSlots[nDate], newStamps...)
							}
						}
					}

					if sch.To > allDay {
						for _, date := range slots[(day+1)%7] {
							if !recDate.After(date) {
								h, m, _ := date.Clock()
								stamp := h*60 + m

								if 0 <= stamp && stamp+d.SlotSize <= sch.To-day {
									nDate := date.Add(time.Duration((-stamp)) * time.Minute).UnixMilli() // only date
									newStamps := TimeStamps(&date, stamp, 0, sch.To-day, d.SlotSize, d.Gap, rep)
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
						if !schDate.After(slotDate) && !slotDate.After(schDate.Add(time.Duration(to-from)*time.Minute)) { // FIXME
							nDate := schDate.UnixMilli()
							h, m, _ := slotDate.Clock()
							stamp := h*60 + m
							newStamps := TimeStamps(&slotDate, stamp, from, to, d.SlotSize, d.Gap, rep)
							availableSlots[nDate] = append(availableSlots[nDate], newStamps...)
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
			log.Print(values)
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
				log.Printf("invalid day abbreviation: %s", weekDay)
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

	sch := newSchedule(from, to, size, gap, days, dates)
	schedule = append(schedule, *sch)

	if to > allDay {
		newDays := make([]int, len(days))
		newDates := make([]int64, len(dates))

		if len(days) > 0 {
			for i, day := range days {
				newDays[i] = (day + 1) % 7
			}
		}

		if len(dates) > 0 {
			for i, date := range dates {
				y, m, d := time.UnixMilli(date).UTC().Date()
				newDates[i] = newStamp(y, m, d+1, 0)
			}
		}

		sch := newSchedule(0, to-allDay, size, gap, newDays, newDates)
		schedule = append(schedule, *sch)
	}

	return schedule
}

func m2t(m int) string {
	if m >= allDay {
		return "24:00"
	}

	hours := m / 60
	minutes := m % 60
	return fmt.Sprintf("%d:%02d", hours, minutes)
}

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

func TimeStamps(date *time.Time, ts, from, to, size, gap int, replace bool) []int64 {
	if date == nil || !(from <= ts && ts+size <= to) {
		return []int64{}
	}

	var (
		slot       = size + gap
		stamp      = ts - from
		timestamps = make([]int64, 0, 2)
	)

	if stamp%slot == 0 || !replace {
		timestamps = append(timestamps, date.UnixMilli())
	} else {
		begin := stamp / slot
		end := begin + 1

		y, m, d := date.Date()
		timestamps = append(timestamps, newStamp(y, m, d, slot*begin+from))
		if slot*end <= to-from-size {
			timestamps = append(timestamps, newStamp(y, m, d, slot*end+from))
		}
	}

	return timestamps
}
