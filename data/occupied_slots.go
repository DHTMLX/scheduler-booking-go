package data

import (
	"gorm.io/gorm"
)

type occupiedSlotsDAO struct {
	db *gorm.DB
}

type Query struct {
	DoctorID  int
	EqualDate int64
	MinDate   int64
	MaxDate   int64
}

func newOccupiedSlotsDAO(db *gorm.DB) *occupiedSlotsDAO {
	return &occupiedSlotsDAO{db}
}

func (d *occupiedSlotsDAO) GetOne(id int) (OccupiedSlot, error) {
	slot := OccupiedSlot{}
	err := d.db.Find(&slot, id).Error
	return slot, err
}

func (d *occupiedSlotsDAO) GetAll() ([]OccupiedSlot, error) {
	slots := make([]OccupiedSlot, 0)
	err := d.db.Find(&slots).Error
	return slots, err
}

func (d *occupiedSlotsDAO) GetUsedSlot(doctorId int, date int64) (OccupiedSlot, error) {
	slots := OccupiedSlot{}
	err := d.db.
		Limit(1).
		Find(&slots, " doctor_id = ? AND date = ?", doctorId, date).Error
	return slots, err
}

func (d *occupiedSlotsDAO) Add(doctor int, date int64, name, email, details string) (int, error) {
	record := OccupiedSlot{
		DoctorID:      doctor,
		Date:          date,
		ClientName:    name,
		ClientEmail:   email,
		ClientDetails: details,
	}
	err := d.db.Save(&record).Error
	return record.ID, err
}

func (d *occupiedSlotsDAO) Delete(id ...int) error {
	return d.db.Delete(&OccupiedSlot{}, id).Error
}
