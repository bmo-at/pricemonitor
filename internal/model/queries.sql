-- name: UpsertStation :one
INSERT INTO pricemonitor_stations (id, address, geo_location, brand)
    VALUES (gen_random_uuid(), sqlc.arg(address), sqlc.arg(geo_location), sqlc.arg(brand))
    ON CONFLICT (address, geo_location, brand)
        DO UPDATE SET address = EXCLUDED.address  -- no-op, ensures RETURNING always fires
    RETURNING id;

-- name: CreateSamples :copyfrom
INSERT INTO pricemonitor_samples (id, fuel_name, price, time, station_id)
VALUES (
    sqlc.arg(id), 
    sqlc.arg(fuel_name), 
    sqlc.arg(price), 
    sqlc.arg(time),
    sqlc.arg(station_id) 
);
