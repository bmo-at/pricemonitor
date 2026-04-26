package stations

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/antchfx/htmlquery"
	"github.com/google/uuid"
	"github.com/sethvargo/go-retry"
)

const BrandAral Brand = "aral"

type StationAral struct {
	brand       Brand
	urlMainPage string
	urlAPI      string
}

func (a StationAral) Identifier() string {
	return a.urlMainPage
}

func (a StationAral) ScrapePrices() (Sample, error) {
	req, err := http.NewRequest(http.MethodGet, a.urlMainPage, nil)
	if err != nil {
		return Sample{}, fmt.Errorf("could not create request for station data: %w", err)
	}

	var station_data_resp *http.Response

	if err := retry.Do(context.TODO(), newScrapeRetry(), func(ctx context.Context) error {
		var err error
		station_data_resp, err = http.DefaultClient.Do(req)

		if err != nil {
			return retry.RetryableError(fmt.Errorf("could not complete request for station data: %w", err))
		}

		if station_data_resp.StatusCode != http.StatusOK {
			defer station_data_resp.Body.Close()
			return retry.RetryableError(errors.New("request status was not '200 OK'"))
		}

		return nil
	}); err != nil {
		return Sample{}, fmt.Errorf("station page returned %s after the maximum number of attempts (%d): %s", station_data_resp.Status, MAX_RETRIES, a.Identifier())
	}

	bytes, err := io.ReadAll(station_data_resp.Body)
	defer station_data_resp.Body.Close()

	if err != nil {
		return Sample{}, fmt.Errorf("could not read station data: %w", err)
	}

	doc, err := htmlquery.Parse(strings.NewReader(string(bytes)))

	if err != nil {
		return Sample{}, fmt.Errorf("could not parse html for station data: %w", err)
	}

	script := htmlquery.FindOne(doc, `/html/head/script[2]/text()`)
	if script == nil {
		return Sample{}, fmt.Errorf("could not find fuel names script in station page: %s from %s", station_data_resp.Status, station_data_resp.Request.URL)
	}

	addressNode1 := htmlquery.FindOne(doc, `/html/body/main/header/div/div/div/div[2]/div[2]/div[1]/p[1]`)
	if addressNode1 == nil {
		return Sample{}, fmt.Errorf("could not find first part of address in station page")
	}

	addressNode2 := htmlquery.FindOne(doc, `/html/body/main/header/div/div/div/div[2]/div[2]/div[1]/p[2]`)
	if addressNode2 == nil {
		return Sample{}, fmt.Errorf("could not find second part of address in station page")
	}

	geolocationNode := htmlquery.FindOne(doc, `/html/body/main/header/div/div/div/div[2]/div[3]/div/a/@href`)
	if geolocationNode == nil {
		return Sample{}, fmt.Errorf("could not find geolocation in station page")
	}

	fuelResolutionMap := make(map[string]string)

	for _, line := range strings.Split(htmlquery.InnerText(script), ";") {
		if strings.Contains(line, "window.FUELS = ") {
			if err := json.Unmarshal([]byte(strings.Split(line, "window.FUELS = ")[1]), &fuelResolutionMap); err != nil {
				return Sample{}, fmt.Errorf("could not parse fuelname resolution map: %w", err)
			}
		}
	}
	req, err = http.NewRequest(http.MethodGet, a.urlAPI, nil)
	if err != nil {
		return Sample{}, fmt.Errorf("could not create request for price data: %w", err)
	}

	var price_data_resp *http.Response

	if err := retry.Do(context.TODO(), newScrapeRetry(), func(ctx context.Context) error {
		var err error
		price_data_resp, err = http.DefaultClient.Do(req)
		if err != nil {
			return retry.RetryableError(fmt.Errorf("could not complete request for price data: %w", err))
		}

		if price_data_resp.StatusCode != http.StatusOK {
			defer price_data_resp.Body.Close()
			return retry.RetryableError(errors.New("request status was not '200 OK'"))
		}

		return nil
	}); err != nil {
		return Sample{}, fmt.Errorf("price API returned %s after the maximum number of attempts (%d): %s", price_data_resp.Status, MAX_RETRIES, a.Identifier())
	}

	bytes, err = io.ReadAll(price_data_resp.Body)
	if err != nil {
		return Sample{}, fmt.Errorf("could read price data: %w", err)
	}

	//nolint:tagliatelle // We do not control the json in this case
	priceData := new(struct {
		Data struct {
			Prices     map[string]string `json:"prices"`
			LastUpdate time.Time         `json:"last_price_update"`
		} `json:"data"`
	})

	err = json.Unmarshal(bytes, priceData)

	if err != nil {
		return Sample{}, fmt.Errorf("could not parse price data: %w", err)
	}

	prices := make(map[string]float32)

	for key, value := range fuelResolutionMap {
		converted, err := strconv.ParseFloat(priceData.Data.Prices[key], 32)

		if err != nil {
			converted = 0.0
		}

		if converted == 0.0 {
			continue
		}

		prices[value] = float32(converted / 100)
	}

	return Sample{
		Prices:      prices,
		Time:        time.Now(),
		Address:     htmlquery.InnerText(addressNode1) + ", " + htmlquery.InnerText(addressNode2),
		GeoLocation: strings.Split(htmlquery.InnerText(geolocationNode), "&destination=")[1],
		Brand:       string(a.brand),
		ScrapeID:    uuid.New(),
	}, nil
}
