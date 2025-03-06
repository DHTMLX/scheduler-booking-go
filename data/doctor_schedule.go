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
	err := d.db.Find(&data, id).Error
	return data, err
}

func (d *doctorsScheduleDAO) GetAll() ([]DoctorSchedule, error) {
	sch := make([]DoctorSchedule, 0)
	err := d.db.Find(&sch).Error
	return sch, err
}

func (d *doctorsScheduleDAO) Add(doctorID, from, to int, date int64, rrule string, duration int, original string, recID string, deleted bool) (int, error) {
	if date == 0 {
		return 0, errors.New("date argument not defined")
	}

	schedule := DoctorSchedule{
		DoctorID:         doctorID,
		From:             from,
		To:               to,
		Date:             date,
		Rrule:            rrule,
		RecurringEventID: recID,
		OriginalStart:    original,
		Duration:         duration,
		Deleted:          deleted,
	}

	err := d.db.Save(&schedule).Error

	return schedule.ID, err
}

func (d *doctorsScheduleDAO) Update(id, doctorID, from, to int, date int64, rrule string, duration int, original string, recID string, deleted bool) (err error) {
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

	newSchedule := DoctorSchedule{
		DoctorID:         doctorID,
		From:             from,
		To:               to,
		Date:             date,
		Rrule:            rrule,
		RecurringEventID: recID,
		OriginalStart:    original,
		Duration:         duration,
		Deleted:          deleted,
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

	err = tx.Delete(&DoctorSchedule{}, "recurring_event_id = ?", id).Error
	if err != nil {
		return err
	}

	err = tx.Delete(&DoctorSchedule{}, "id = ?", id).Error
	if err != nil {
		return err
	}

	return nil
}
