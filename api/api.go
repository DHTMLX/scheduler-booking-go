package api

import (
	"fmt"
	"net/http"
	"scheduler-booking/service"

	"github.com/go-chi/chi"
	"github.com/unrolled/render"
)

var Debug = true

type API struct {
	sAll   *service.ServiceAll
	format *render.Render
}

func NewAPI(service *service.ServiceAll) *API {
	format := render.New()
	return &API{service, format}
}

func (api *API) InitRoutes(r chi.Router) {

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		api.response(w, "Server launched successfully!", nil)
	})

	r.Get("/units", func(w http.ResponseWriter, r *http.Request) {
		units, err := api.sAll.Units.GetAll()
		api.response(w, units, err)
	})

	r.Get("/doctors", func(w http.ResponseWriter, r *http.Request) {
		doctors, err := api.sAll.Doctors.GetDoctorsList()
		api.response(w, doctors, err)
	})

	r.Get("/doctors/worktime", func(w http.ResponseWriter, r *http.Request) {
		data, err := api.sAll.Worktime.GetAll()
		api.response(w, data, err)
	})

	r.Post("/doctors/worktime", func(w http.ResponseWriter, r *http.Request) {
		worktime := service.Worktime{}
		err := parseForm(w, r, &worktime)
		if err != nil {
			api.errResponse(w, err.Error())
			return
		}
		id, err := api.sAll.Worktime.Add(worktime)

		action := "inserted"
		if worktime.Deleted {
			action = "deleted"
		}

		api.response(w, &response{
			Action: action,
			ID:     id,
		}, err)
	})

	r.Put("/doctors/worktime/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := numberParam(r, "id")
		worktime := service.Worktime{}
		err := parseForm(w, r, &worktime)
		if err != nil {
			api.errResponse(w, err.Error())
			return
		}
		err = api.sAll.Worktime.Update(id, worktime)

		api.response(w, &response{Action: "updated"}, err)
	})

	r.Delete("/doctors/worktime/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := numberParam(r, "id")
		err := api.sAll.Worktime.Delete(id)
		api.response(w, &response{Action: "deleted"}, err)
	})

	r.Get("/doctors/reservations", func(w http.ResponseWriter, r *http.Request) {
		reservations, err := api.sAll.Reservations.GetAll()
		api.response(w, reservations, err)
	})

	r.Post("/doctors/reservations", func(w http.ResponseWriter, r *http.Request) {
		reservation := service.Reservation{}
		err := parseForm(w, r, &reservation)
		if err != nil {
			api.errResponse(w, err.Error())
			return
		}
		id, err := api.sAll.Reservations.Add(reservation)

		api.response(w, &response{ID: id}, err)
	})

	r.Delete("/doctors/reservations/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := numberParam(r, "id")
		err := api.sAll.Reservations.Delete(id)
		api.response(w, &response{ID: id}, err)
	})
}

func (api *API) response(w http.ResponseWriter, data any, err error) {
	if err != nil {
		api.errResponse(w, err.Error())
	} else {
		api.format.JSON(w, 200, data)
	}
}

func (api *API) errResponse(w http.ResponseWriter, msg string) {
	if Debug {
		fmt.Println(msg)
	}
	api.format.Text(w, 500, msg)
}
