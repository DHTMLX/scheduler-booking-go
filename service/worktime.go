package service

import (
	"fmt"
	"scheduler-booking/common"
	"scheduler-booking/data"
	"time"
)

type worktimeService struct {
	dao *data.DAO
}

type Worktime struct {
	DoctorID         int           `json:"doctor_id"`
	StartDate        *common.JDate `json:"start_date"`
	EndDate          *common.JDate `json:"end_date"`
	Rrule            string        `json:"rrule"`
	Duration         int           `json:"duration"` // in seconds
	RecurringEventID string        `json:"recurring_event_id"`
	OriginalStart    string        `json:"original_start"`
	Deleted          bool          `json:"deleted"`
}

type DoctorRoutineStr struct {
	ID               int    `json:"id"`
	DoctorID         int    `json:"doctor_id"`
	StartDate        string `json:"start_date,omitempty"`
	EndDate          string `json:"end_date,omitempty"`
	Rrule            string `json:"rrule,omitempty"`
	Duration         int    `json:"duration,omitempty"`
	RecurringEventID string `json:"recurring_event_id,omitempty"`
	OriginalStart    string `json:"original_start,omitempty"`
	Deleted          bool   `json:"deleted,omitempty"`
}

const strFormat = "2006-01-02 15:04:05"

// returns records for the Scheduler Doctors View
func (s *worktimeService) GetAll() ([]DoctorRoutineStr, error) {
	schedule, err := s.dao.DoctorsSchedule.GetAll()
	out := make([]DoctorRoutineStr, 0)

	for _, sch := range schedule {
		fh := sch.From / 60
		fm := sch.From % 60
		th := sch.To / 60
		tm := sch.To % 60

		y, m, d := time.UnixMilli(sch.Date).UTC().Date()

		end := data.EndDate
		if sch.Rrule == "" {
			end = time.Date(y, m, d, th, tm, 0, 0, time.UTC)
		}

		r := DoctorRoutineStr{
			ID:               sch.ID,
			DoctorID:         sch.DoctorID,
			StartDate:        time.Date(y, m, d, fh, fm, 0, 0, time.UTC).Format(strFormat),
			EndDate:          end.Format(strFormat),
			Rrule:            sch.Rrule,
			Duration:         sch.Duration,
			RecurringEventID: sch.RecurringEventID,
			OriginalStart:    sch.OriginalStart,
			Deleted:          sch.Deleted,
		}

		out = append(out, r)
	}

	return out, err
}

// adds doctor's schedule
func (s *worktimeService) Add(data Worktime) (int, error) {
	if err := data.validate(); err != nil {
		return 0, err
	}

	date := data.StartDate.Truncate(24 * time.Hour).UnixMilli()

	from := data.StartDate.Hour()*60 + data.StartDate.Minute()
	to := from + data.duration()

	id, err := s.dao.DoctorsSchedule.Add(
		data.DoctorID,
		from,
		to,
		date,
		data.Rrule,
		data.Duration,
		data.OriginalStart,
		data.RecurringEventID,
		data.Deleted,
	)
	return id, err
}

// updates doctor's schedule
func (s *worktimeService) Update(scheduleID int, data Worktime) error {
	schedule, err := s.dao.DoctorsSchedule.GetOne(scheduleID)
	if err != nil {
		return err
	}

	if schedule.ID == 0 {
		return fmt.Errorf("schedule with id %d not found", scheduleID)
	}

	if err := data.validate(); err != nil {
		return err
	}

	date := data.StartDate.Truncate(24 * time.Hour).UnixMilli()

	from := data.StartDate.Hour()*60 + data.StartDate.Minute()
	to := from + data.duration()

	err = s.dao.DoctorsSchedule.Update(
		scheduleID,
		data.DoctorID,
		from,
		to,
		date,
		data.Rrule,
		data.Duration,
		data.OriginalStart,
		data.RecurringEventID,
		data.Deleted,
	)
	return err
}

// delets doctor's schedule for the specific day
func (s *worktimeService) Delete(id int) error {
	return s.dao.DoctorsSchedule.Delete(id)
}

func (w Worktime) validate() error {
	if w.StartDate.UnixMilli() < data.Now().UnixMilli() {
		return fmt.Errorf("cannot set work time in the past")
	}
	if w.StartDate.UnixMilli() >= w.EndDate.UnixMilli() {
		return fmt.Errorf("invalid time interval")
	}
	return nil
}

// in minutes
func (w Worktime) duration() int {
	if w.Duration != 0 {
		return w.Duration / 60
	}

	diff := w.EndDate.Sub(w.StartDate.Time)
	return int(diff.Minutes())
}
