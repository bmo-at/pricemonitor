-- +goose Up
CREATE TABLE stations (
	"id" UUID PRIMARY KEY NOT NULL,
	"address" TEXT NOT NULL,
	"geo_location" TEXT NOT NULL,
	"brand" TEXT NOT NULL,
	UNIQUE(address, geo_location, brand)
);

CREATE TABLE samples (
	"id" UUID NOT NULL,
	"fuel_name" TEXT NOT NULL,
	"price" REAL NOT NULL,
	"time" TIMESTAMP WITH TIME ZONE NOT NULL,
	"station_id" UUID REFERENCES stations(id) NOT NULL
);

SELECT * FROM create_hypertable('samples', by_range('time'));

CREATE MATERIALIZED VIEW weekly_fuel_prices
WITH (timescaledb.continuous) AS
SELECT
  time_bucket('1w', time) AS week,
  fuel_name,
  min(price) AS minimum,
  avg(price) AS average
FROM samples
WHERE price > 0
GROUP BY week, fuel_name
WITH NO DATA;

SELECT add_continuous_aggregate_policy('weekly_fuel_prices',
  start_offset => NULL,
  end_offset => NULL,
  schedule_interval => INTERVAL '1d');

-- +goose Down
DROP TABLE samples;
DROP TABLE stations;
DROP MATERIALIZED VIEW weekly_fuel_prices;