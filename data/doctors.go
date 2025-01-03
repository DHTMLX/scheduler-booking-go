package data

import (
	"gorm.io/gorm"
)

type doctorsDAO struct {
	db *gorm.DB
}

func newDoctorsDAO(db *gorm.DB) *doctorsDAO {
	return &doctorsDAO{db}
}

func (d *doctorsDAO) GetOne(id int) (Doctor, error) {
	doctor := Doctor{}
	err := d.db.Find(&doctor, id).Error
	return doctor, err
}

func (d *doctorsDAO) GetAll(preload bool) ([]Doctor, error) {
	doctors := make([]Doctor, 0)
	var err error
	if !preload {
		err = d.db.Find(&doctors).Error
	} else {
		date := DateNow().UnixMilli()
		err = d.db.
			Preload("Review").
			Preload("OccupiedSlots", "date >= ?", date).
			Preload("DoctorSchedule").
			Preload("DoctorSchedule.DoctorRecurringRoutine").
			Preload("DoctorSchedule.DoctorRoutine", "date >= ?", date).
			Find(&doctors).Error
	}

	return doctors, err
}
