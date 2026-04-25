package stations

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/antchfx/htmlquery"
	"github.com/google/uuid"
)

const (
	maxRetries     = 3
	baseRetryDelay = 2 * time.Second
)

// retryBackoff returns a jittered exponential backoff duration for the given attempt (0-indexed).
func retryBackoff(attempt int) time.Duration {
	backoff := float64(baseRetryDelay) * math.Pow(2, float64(attempt))
	jitter := rand.Float64() * 0.5 * backoff
	return time.Duration(backoff + jitter)
}

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

	var resp *http.Response
	for attempt := range maxRetries {
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			return Sample{}, fmt.Errorf("could not complete request for station data: %w", err)
		}
		if resp.StatusCode == http.StatusOK {
			break
		}
		resp.Body.Close()
		if attempt == maxRetries-1 {
			return Sample{}, fmt.Errorf("station page returned %s after %d attempts: %s", resp.Status, maxRetries, a.Identifier())
		}
		delay := retryBackoff(attempt)
		slog.Info("status for station was not OK, retrying", "station_identifier", a.Identifier(), "status", resp.Status, "attempt", attempt+1, "delay", delay)
		time.Sleep(delay)
	}

	bytes, err := io.ReadAll(resp.Body)

	resp.Body.Close()

	if err != nil {
		return Sample{}, fmt.Errorf("could not read station data: %w", err)
	}

	doc, err := htmlquery.Parse(strings.NewReader(string(bytes)))

	if err != nil {
		return Sample{}, fmt.Errorf("could not parse html for station data: %w", err)
	}

	script := htmlquery.FindOne(doc, `/html/head/script[2]/text()`)
	if script == nil {
		return Sample{}, fmt.Errorf("could not find fuel names script in station page: %s from %s", resp.Status, resp.Request.URL)
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

	for attempt := range maxRetries {
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			return Sample{}, fmt.Errorf("could not complete request for price data: %w", err)
		}
		if resp.StatusCode == http.StatusOK {
			break
		}
		resp.Body.Close()
		if attempt == maxRetries-1 {
			return Sample{}, fmt.Errorf("price API returned %s after %d attempts: %s", resp.Status, maxRetries, a.Identifier())
		}
		delay := retryBackoff(attempt)
		slog.Info("status for price API was not OK, retrying", "station_identifier", a.Identifier(), "status", resp.Status, "attempt", attempt+1, "delay", delay)
		time.Sleep(delay)
	}

	bytes, err = io.ReadAll(resp.Body)

	resp.Body.Close()

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
		ID:          uuid.New(),
	}, nil
}
