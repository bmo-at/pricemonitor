package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/antchfx/htmlquery"
	model "github.com/bmo-at/pricemonitor/internal/model/generated"
	"github.com/bmo-at/pricemonitor/internal/model/migrations"
	"github.com/bmo-at/pricemonitor/internal/stations"
	"github.com/pressly/goose/v3"
	"go-simpler.org/env"

	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type PriceMonitorApplication struct {
	database *sql.DB
	pgx      *pgx.Conn
	queries  *model.Queries
	stations []stations.Station
	config   Config
}

type Config struct {
	Database struct {
		User     string `default:"postgres"  env:"USER"`
		Password string `default:"password"  env:"PASSWORD"`
		Host     string `default:"localhost" env:"HOST"`
		Port     uint16 `default:"5432"      env:"PORT"`
	} `env:"PRICEMONITOR_DATABASE_"`

	Stations string `env:"PRICEMONITOR_STATIONS"`
}

func NewPriceMonitorApplication() (*PriceMonitorApplication, error) {
	app := new(PriceMonitorApplication)

	if err := env.Load(&app.config, nil); err != nil {
		return nil, fmt.Errorf("could not load config: %w", err)
	}

	app.stations = make([]stations.Station, 0)

	if len(app.config.Stations) > 0 {
		for _, station := range strings.Split(app.config.Stations, ",") {
			station, err := stations.NewStation(station)

			if err != nil {
				return nil, err
			}

			app.stations = append(app.stations, station)
		}
	} else {
		log.Printf("Environment variable %s not set, not tracking any stations!", "PRICEMONITOR_STATIONS")
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=pricemonitor port=%d",
		app.config.Database.Host,
		app.config.Database.User,
		app.config.Database.Password,
		app.config.Database.Port,
	)

	sqlDB, err := sql.Open("pgx", dsn)

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	app.database = sqlDB

	if err := goose.SetDialect("postgres"); err != nil {
		return nil, fmt.Errorf("unable to set dialect for database migration: %w", err)
	}

	goose.SetBaseFS(migrations.FS)
	err = goose.Up(app.database, ".")

	if err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	conn, err := pgx.Connect(context.Background(), dsn)

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	app.queries = model.New(conn)
	app.pgx = conn

	return app, nil
}

type Location struct {
	Identifier string
}

func main() {
	app, err := NewPriceMonitorApplication()

	htmlquery.DisableSelectorCache = true

	if err != nil {
		panic(err)
	}

	funnel := make(chan stations.Sample)

	go app.collector(funnel)

	for {
		for _, station := range app.stations {
			go func() {
				sample, err := station.ScrapePrices()
				if err != nil {
					// TODO: Error should be better raised here
					log.Println(err)

					return
				}
				funnel <- sample
			}()
		}

		time.Sleep(time.Minute)
	}
}

func (app PriceMonitorApplication) collector(rx <-chan stations.Sample) {
	for {
		sample := <-rx
		for name, price := range sample.Prices {
			_, err := app.queries.CreateSampleAndStation(context.Background(), model.CreateSampleAndStationParams{
				ID:          sample.ID,
				FuelName:    name,
				Price:       price,
				Time:        sample.Time,
				Address:     sample.Address,
				GeoLocation: sample.GeoLocation,
				Brand:       sample.Brand,
			})

			if err != nil {
				log.Println(err)
			}
		}
	}
}
