// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: queries.sql

package model

import (
	"context"
	"time"

	"github.com/google/uuid"
)

const createSampleAndStation = `-- name: CreateSampleAndStation :one
WITH station_create AS (
    INSERT INTO pricemonitor_stations (id, address, geo_location, brand)
    VALUES (gen_random_uuid(), $5, $6, $7)
    ON CONFLICT (address, geo_location, brand) DO NOTHING
    RETURNING id
)
INSERT INTO pricemonitor_samples (id, fuel_name, price, time, station_id)
VALUES (
    $1, 
    $2, 
    $3, 
    $4, 
    COALESCE(
        (SELECT id FROM station_create), 
        (SELECT id FROM pricemonitor_stations WHERE pricemonitor_stations.address = $5 AND pricemonitor_stations.geo_location = $6 AND pricemonitor_stations.brand = $7)
    )
) RETURNING id, fuel_name, price, time, station_id
`

type CreateSampleAndStationParams struct {
	ID          uuid.UUID `json:"id"`
	FuelName    string    `json:"fuel_name"`
	Price       float32   `json:"price"`
	Time        time.Time `json:"time"`
	Address     string    `json:"address"`
	GeoLocation string    `json:"geo_location"`
	Brand       string    `json:"brand"`
}

func (q *Queries) CreateSampleAndStation(ctx context.Context, arg CreateSampleAndStationParams) (PricemonitorSample, error) {
	row := q.db.QueryRow(ctx, createSampleAndStation,
		arg.ID,
		arg.FuelName,
		arg.Price,
		arg.Time,
		arg.Address,
		arg.GeoLocation,
		arg.Brand,
	)
	var i PricemonitorSample
	err := row.Scan(
		&i.ID,
		&i.FuelName,
		&i.Price,
		&i.Time,
		&i.StationID,
	)
	return i, err
}
