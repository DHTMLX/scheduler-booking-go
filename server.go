package main

import (
	"fmt"
	"log"
	"net/http"
	"scheduler-booking/api"
	"scheduler-booking/data"
	"scheduler-booking/service"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"github.com/jinzhu/configor"
)

var Config AppConfig

func main() {
	configor.New(&configor.Config{ENVPrefix: "APP", Silent: true}).Load(&Config, "config.yml")

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	fmt.Println(Config.Server.Cors)
	if len(Config.Server.Cors) > 0 {
		c := cors.New(cors.Options{
			AllowedOrigins:   Config.Server.Cors,
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "Remote-Token", "X-Requested-With"},
			AllowCredentials: true,
			MaxAge:           300,
		})
		r.Use(c.Handler)
	}

	dao := data.NewDAO(Config.DB)
	service := service.NewService(dao)
	api := api.NewAPI(service)

	api.InitRoutes(r)

	if Config.Server.ResetFrequence > 0 {
		go func() {
			f := time.Duration(Config.Server.ResetFrequence) * time.Minute
			ticker := time.NewTicker(f)
			for range ticker.C {
				log.Println("Reset data...")
				dao.RestartData()
			}
		}()
	}

	log.Printf("Starting webserver at port " + Config.Server.Port)
	err := http.ListenAndServe(Config.Server.Port, r)
	if err != nil {
		log.Println(err.Error())
	}
}
