-- name: CreateSampleAndStation :one
WITH station_create AS (
    INSERT INTO pricemonitor_stations (id, address, geo_location, brand)
    VALUES (gen_random_uuid(), sqlc.arg(address), sqlc.arg(geo_location), sqlc.arg(brand))
    ON CONFLICT (address, geo_location, brand) DO NOTHING
    RETURNING id
)
INSERT INTO pricemonitor_samples (id, fuel_name, price, time, station_id)
VALUES (
    sqlc.arg(id), 
    sqlc.arg(fuel_name), 
    sqlc.arg(price), 
    sqlc.arg(time), 
    COALESCE(
        (SELECT id FROM station_create), 
        (SELECT id FROM pricemonitor_stations WHERE pricemonitor_stations.address = sqlc.arg(address) AND pricemonitor_stations.geo_location = sqlc.arg(geo_location) AND pricemonitor_stations.brand = sqlc.arg(brand))
    )
) RETURNING *;