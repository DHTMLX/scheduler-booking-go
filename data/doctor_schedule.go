package data

import (
	"errors"

	"gorm.io/gorm"
)

type doctorsScheduleDAO struct {
	db *gorm.DB
}

func newDoctorsScheduleDAO(db *gorm.DB) *doctorsScheduleDAO {
	return &doctorsScheduleDAO{db}
}

func (d *doctorsScheduleDAO) GetOne(id int) (DoctorSchedule, error) {
	data := DoctorSchedule{}
	err := d.db.
		Preload("DoctorRoutine").
		Preload("DoctorRecurringRoutine").
		Find(&data, id).Error
	return data, err
}

func (d *doctorsScheduleDAO) GetAllSchedule() ([]DoctorSchedule, error) {
	sch := make([]DoctorSchedule, 0)

	err := d.db.
		Preload("DoctorRoutine").
		Preload("DoctorRecurringRoutine").
		Find(&sch).Error

	return sch, err
}

func (d *doctorsScheduleDAO) RecurringEvent(recID int) ([]DoctorRoutine, error) {
	data := make([]DoctorRoutine, 0)

	err := d.db.
		Find(&data, "recurring_event_id = ?", recID).Error

	return data, err
}

func (d *doctorsScheduleDAO) AddRoutineOnDate(doctorID, from, to int, date int64, rrule string, duration int, original string, recID string, deleted bool) (int, error) {
	if date == 0 {
		return 0, errors.New("date argument not defined")
	}

	var docRout *DoctorRoutine
	var docRecRout *DoctorRecurringRoutine

	if rrule == "" {
		docRout = &DoctorRoutine{
			Date:             date,
			OriginalStart:    original,
			RecurringEventID: recID,
			Deleted:          deleted,
		}
	} else {
		docRecRout = &DoctorRecurringRoutine{
			Date:     date,
			Rrule:    rrule,
			Duration: duration,
		}
	}

	schedule := DoctorSchedule{
		DoctorID:               doctorID,
		From:                   from,
		To:                     to,
		DoctorRoutine:          docRout,
		DoctorRecurringRoutine: docRecRout,
	}

	err := d.db.Save(&schedule).Error

	return schedule.ID, err
}

func (d *doctorsScheduleDAO) UpdateDateSchedule(id, doctorID, from, to int, date int64, rrule string, duration int, original string, recID string, deleted bool) (err error) {
	tx := d.db.Begin()
	defer func() {
		if err == nil {
			tx.Commit()
		} else {
			tx.Rollback()
		}
	}()

	schedule, err := d.GetOne(id)
	if err != nil {
		return err
	}

	// delete routine in schedule becaus we dont know routine id
	err = tx.Delete(&DoctorRoutine{}, "schedule_id = ?", id).Error
	if err != nil {
		return err
	}

	// delete routine in schedule becaus we dont know recurring routine id
	err = tx.Delete(&DoctorRecurringRoutine{}, "schedule_id = ?", id).Error
	if err != nil {
		return err
	}

	var docRout *DoctorRoutine
	var docRecRout *DoctorRecurringRoutine

	if rrule == "" {
		docRout = &DoctorRoutine{
			Date:             date,
			OriginalStart:    original,
			RecurringEventID: recID,
			Deleted:          deleted,
		}
	} else {
		docRecRout = &DoctorRecurringRoutine{
			Date:     date,
			Rrule:    rrule,
			Duration: duration,
		}
	}

	newSchedule := DoctorSchedule{
		DoctorID:               doctorID,
		From:                   from,
		To:                     to,
		DoctorRoutine:          docRout,
		DoctorRecurringRoutine: docRecRout,
	}

	if schedule.DoctorID == newSchedule.DoctorID {
		// add ID to update existing
		newSchedule.ID = schedule.ID
		err = tx.Save(&newSchedule).Error
	} else {
		// delete schedule at all for this old doctor and create schedule for doctorID
		err = tx.Delete(&DoctorSchedule{}, id).Error
		if err != nil {
			return err
		}

		err = tx.Create(&newSchedule).Error
	}

	return err
}

func (d *doctorsScheduleDAO) Delete(id int) (err error) {
	tx := d.db.Begin()
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	err = tx.Delete(&DoctorRoutine{}, "schedule_id = ?", id).Error
	if err != nil {
		return err
	}

	err = tx.Delete(&DoctorRecurringRoutine{}, "schedule_id = ?", id).Error
	if err != nil {
		return err
	}

	ids := []int{}
	err = tx.Model(&DoctorRoutine{}).Where("recurring_event_id = ?", id).Pluck("schedule_id", &ids).Error
	if err != nil {
		return err
	}

	err = tx.Delete(&DoctorRoutine{}, "recurring_event_id = ?", id).Error
	if err != nil {
		return err
	}

	err = tx.Delete(&DoctorSchedule{}, "id IN ?", append(ids, id)).Error
	if err != nil {
		return err
	}

	return nil
}
