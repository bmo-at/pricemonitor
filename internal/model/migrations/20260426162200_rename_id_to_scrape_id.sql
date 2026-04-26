-- +goose Up
ALTER TABLE pricemonitor_samples RENAME COLUMN id TO scrape_id;

-- +goose Down
ALTER TABLE pricemonitor_samples RENAME COLUMN scrape_id TO id;
