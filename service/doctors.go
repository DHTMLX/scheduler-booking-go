package service

import (
	"scheduler-booking/data"
)

type doctorsService struct {
	dao *data.DAO
}

func (s *doctorsService) GetDoctorsList() ([]data.Doctor, error) {
	doctors, err := s.dao.Doctors.GetAll(false)
	return doctors, err
}
