package service

import "scheduler-booking/data"

type ServiceAll struct {
	Doctors      *doctorsService
	Worktime     *worktimeService
	Reservations *reservationsService
	Units        *unitsService
}

func NewService(dao *data.DAO) *ServiceAll {
	return &ServiceAll{
		Doctors:      &doctorsService{dao},
		Reservations: &reservationsService{dao},
		Worktime:     &worktimeService{dao},
		Units:        &unitsService{dao},
	}
}
