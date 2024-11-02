-- +goose Up
CREATE TABLE pricemonitor_stations (
	"id" UUID PRIMARY KEY NOT NULL,
	"address" TEXT NOT NULL,
	"geo_location" TEXT NOT NULL,
	"brand" TEXT NOT NULL,
	UNIQUE(address, geo_location, brand)
);

CREATE TABLE pricemonitor_samples (
	"id" UUID NOT NULL,
	"fuel_name" TEXT NOT NULL,
	"price" REAL NOT NULL,
	"time" TIMESTAMP WITH TIME ZONE NOT NULL,
	"station_id" UUID REFERENCES pricemonitor_stations(id) NOT NULL
);

SELECT * FROM create_hypertable('pricemonitor_samples', by_range('time'));

CREATE MATERIALIZED VIEW pricemonitor_weekly_fuel_prices
WITH (timescaledb.continuous) AS
SELECT
  time_bucket('1w', time) AS week,
  fuel_name,
  min(price) AS minimum,
  avg(price) AS average
FROM pricemonitor_samples
WHERE price > 0
GROUP BY week, fuel_name
WITH NO DATA;

SELECT add_continuous_aggregate_policy('pricemonitor_weekly_fuel_prices',
  start_offset => NULL,
  end_offset => NULL,
  schedule_interval => INTERVAL '1d');

-- +goose Down
DROP TABLE pricemonitor_samples;
DROP TABLE pricemonitor_stations;
DROP MATERIALIZED VIEW pricemonitor_weekly_fuel_prices;