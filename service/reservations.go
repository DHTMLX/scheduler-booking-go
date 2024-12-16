package service

import (
	"fmt"
	"scheduler-booking/data"
	"time"
)

type reservationsService struct {
	dao *data.DAO
}

type ReservationForm struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Details string `json:"details"`
}

type Reservation struct {
	DoctorID int             `json:"doctor"`
	Date     int64           `json:"date"`
	Form     ReservationForm `json:"form"`
}

func (s *reservationsService) GetAll() ([]data.OccupiedSlot, error) {
	records, err := s.dao.OccupiedSlots.GetAll()
	if err != nil {
		return nil, err
	}

	doctors, err := s.dao.Doctors.GetAll(true)
	if err != nil {
		return nil, err
	}

	mapRecords := make(map[int64]data.OccupiedSlot)
	for _, record := range records {
		mapRecords[record.Date] = record
	}

	availableSlots := []data.OccupiedSlot{}
	units := createUnits(doctors, false)
	for _, unit := range units {
		for _, uslots := range unit.UsedSlots {
			record := mapRecords[uslots]
			if record.DoctorID == unit.ID {
				availableSlots = append(availableSlots, record)
			}
		}
	}

	return availableSlots, nil
}

func (s *reservationsService) Add(r Reservation) (int, error) {
	// check if reservation time is available and has not expired yet
	err := s.checkIfReservationIsAvailable(r.DoctorID, r.Date)
	if err != nil {
		return 0, err
	}

	id, err := s.dao.OccupiedSlots.Add(
		r.DoctorID,
		r.Date,
		r.Form.Name,
		r.Form.Email,
		r.Form.Details,
	)

	return id, err
}

func (s *reservationsService) Delete(id int) error {
	slot, err := s.dao.OccupiedSlots.GetOne(id)
	if err != nil {
		return err
	}
	if slot.Date < time.Now().Add(-12*time.Hour).UnixMilli() {
		return fmt.Errorf("cannot delete reservation that time has expired")
	}
	return s.dao.OccupiedSlots.Delete(id)
}

func (s *reservationsService) checkIfReservationIsAvailable(doctorId int, date int64) error {
	slot, err := s.dao.OccupiedSlots.GetUsedSlot(doctorId, date)
	if err != nil {
		return err
	}
	if slot.ID != 0 {
		return fmt.Errorf("this time is already booked")
	}
	if date < time.Now().Add(-12*time.Hour).UnixMilli() {
		return fmt.Errorf("booking time has expired")
	}

	return err
}
