package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"sync"
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
		User         string        `default:"postgres"  env:"USER"`
		Password     string        `default:"password"  env:"PASSWORD"`
		Host         string        `default:"localhost" env:"HOST"`
		Port         uint16        `default:"5432"      env:"PORT"`
		BatchTimeout time.Duration `default:"15s"       env:"BATCH_TIMEOUT"`
	} `env:"PRICEMONITOR_DATABASE_"`

	Logger struct {
		Level string `default:"INFO" env:"LEVEL"`
		// Format string `default:"text" env:"FORMAT"`
	} `env:"PRICEMONITOR_LOGGER_"`

	Stations string `env:"PRICEMONITOR_STATIONS"`
}

func NewPriceMonitorApplication() (*PriceMonitorApplication, error) {
	app := new(PriceMonitorApplication)

	if err := env.Load(&app.config, nil); err != nil {
		return nil, fmt.Errorf("could not load config: %w", err)
	}

	if app.config.Logger.Level == "debug" || app.config.Logger.Level == "DEBUG" {
		slog.SetLogLoggerLevel(slog.LevelDebug)
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
		slog.Warn("Environment variable 'PRICEMONITOR_STATIONS' not set, not tracking any stations!")
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=postgres port=%d",
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
		start := time.Now()
		wg := new(sync.WaitGroup)
		done := make(chan bool)
		work := make(chan stations.Station)

		for worker_id := range 5 {
			wg.Add(1)

			go func() {
				defer wg.Done()

			out:
				for {
					select {
					case station := <-work:
						slog.Debug("received station in worker", "station", station.Identifier(), "worker_id", worker_id)
						sample, err := station.ScrapePrices()
						if err != nil {
							slog.Error(err.Error())
						}
						funnel <- sample

						continue
					case <-done:
						break out
					}
				}
			}()
		}

		for _, station := range app.stations {
			slog.Debug("putting station into the work pool", "station", station.Identifier())
			work <- station
		}

		for range 5 {
			done <- true
		}

		wg.Wait()
		slog.Debug("work is done", "duration", time.Since(start).Seconds())

		time.Sleep(time.Minute)
	}
}

func (app PriceMonitorApplication) collector(rx <-chan stations.Sample) {
	for {
		processed_samples := 0

		sample := <-rx
		first_sample_time := time.Now()

		samples := make([]model.CreateSamplesParams, 0, len(app.stations)*10)

		for {
			station_id, err := app.queries.UpsertStation(context.Background(), model.UpsertStationParams{
				Address:     sample.Address,
				GeoLocation: sample.GeoLocation,
				Brand:       sample.Brand,
			})

			if err != nil {
				slog.Error(err.Error())
			}

			for name, price := range sample.Prices {
				samples = append(samples, model.CreateSamplesParams{
					ID:        sample.ID,
					FuelName:  name,
					Price:     price,
					Time:      sample.Time,
					StationID: station_id,
				})
			}

			processed_samples++

			// Buffer is more than 80% full or more than 10 seconds have passed since first sample OR all samples are in
			if processed_samples == len(app.stations) {
				slog.Info(fmt.Sprintf("%d/%d samples are in, writing...", processed_samples, len(app.stations)))
				break
			}

			if len(samples) >= ((cap(samples) * 8) / 10) {
				slog.Info(fmt.Sprintf("Buffer is at over 80 percent capacity (%d/%d), writing...", len(samples), cap(samples)))
				break
			}

			if time.Since(first_sample_time) > app.config.Database.BatchTimeout {
				slog.Info(fmt.Sprintf("Time since first sample exceeded %s timeout (%s), writing...", app.config.Database.BatchTimeout.String(), time.Since(first_sample_time).String()))
				break
			}

			sample = <-rx
		}

		_, err := app.queries.CreateSamples(context.Background(), samples)

		if err != nil {
			slog.Error(err.Error())
		}
	}
}
