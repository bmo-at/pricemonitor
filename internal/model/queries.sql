-- name: CreateSampleAndStation :one
WITH station_create AS (
    INSERT INTO stations (id, address, geo_location, brand)
    VALUES (gen_random_uuid(), sqlc.arg(address), sqlc.arg(geo_location), sqlc.arg(brand))
    ON CONFLICT (address, geo_location, brand) DO NOTHING
    RETURNING id
)
INSERT INTO samples (id, fuel_name, price, time, station_id)
VALUES (
    sqlc.arg(id), 
    sqlc.arg(fuel_name), 
    sqlc.arg(price), 
    sqlc.arg(time), 
    COALESCE(
        (SELECT id FROM station_create), 
        (SELECT id FROM stations WHERE stations.address = sqlc.arg(address) AND stations.geo_location = sqlc.arg(geo_location) AND stations.brand = sqlc.arg(brand))
    )
) RETURNING *;

-- name: RefreshWeeklyFuelPrices :exec