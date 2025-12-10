-- +goose Up
CREATE MATERIALIZED VIEW IF NOT EXISTS pricemonitor_daily_fuel_prices
WITH (timescaledb.continuous) AS
SELECT
  time_bucket('1d', time) AS day,
  fuel_name,
  min(price) AS minimum,
  avg(price) AS average
FROM pricemonitor_samples
WHERE price > 0
GROUP BY day, fuel_name
WITH NO DATA;

SELECT add_continuous_aggregate_policy('pricemonitor_daily_fuel_prices',
  start_offset => NULL,
  end_offset => NULL,
  schedule_interval => INTERVAL '1h');

-- +goose Down
DROP MATERIALIZED VIEW pricemonitor_daily_fuel_prices;