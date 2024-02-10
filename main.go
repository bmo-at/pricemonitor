package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/antchfx/htmlquery"
	"github.com/bmo-at/pricemonitor/internal/constants"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"golang.org/x/net/html"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type PriceMonitorApplication struct {
	database        *gorm.DB
	locations       []Location
	browserless_url string
}

func NewPriceMonitorApplication(locations ...string) (*PriceMonitorApplication, error) {
	app := new(PriceMonitorApplication)

	for _, l := range locations {
		app.locations = append(app.locations, Location{
			Identifier: l,
		})
	}

	db_user := "postgres"

	if value, exists := os.LookupEnv("PRICEMONITOR_DB_USER"); exists {
		db_user = value
	} else {
		log.Printf("Environment variable %s not set, using default value %s", "PRICEMONITOR_DB_USER", db_user)
	}

	db_password := "pricemonitor_dev_password"

	if value, exists := os.LookupEnv("PRICEMONITOR_DB_PASSWORD"); exists {
		db_password = value
	} else {
		log.Printf("Environment variable %s not set, using default value %s", "PRICEMONITOR_DB_PASSWORD", db_password)
	}

	db_host := "localhost"

	if value, exists := os.LookupEnv("PRICEMONITOR_DB_HOST"); exists {
		db_host = value
	} else {
		log.Printf("Environment variable %s not set, using default value %s", "PRICEMONITOR_DB_HOST", db_host)
	}

	db_port := "5432"

	if value, exists := os.LookupEnv("PRICEMONITOR_DB_PORT"); exists {
		db_port = value
	} else {
		log.Printf("Environment variable %s not set, using default value %s", "PRICEMONITOR_DB_PORT", db_port)
	}

	app.browserless_url = "http://localhost:3000/content?token=dev_token"

	if value, exists := os.LookupEnv("PRICEMONITOR_BROWSERLESS_URL"); exists {
		app.browserless_url = value
	} else {
		log.Printf("Environment variable %s not set, using default value %s", "PRICEMONITOR_BROWSERLESS_URL", app.browserless_url)
	}

	db, err := gorm.Open(postgres.Open(fmt.Sprintf("host=%s user=%s password=%s dbname=postgres port=%s", db_host, db_user, db_password, db_port)), &gorm.Config{})

	if err != nil {
		return nil, errors.Join(fmt.Errorf("failed to connect to database: %v", err.Error()), err)
	}

	err = db.AutoMigrate(&PriceDbEntry{})

	if err != nil {
		return nil, errors.Join(fmt.Errorf("failed to automigrate database: %v", err.Error()), err)
	}

	var tables []struct {
		Name string
	}

	db.Raw("SELECT table_name AS name FROM _timescaledb_catalog.hypertable").Scan(&tables)

	hypertable_exists := false

	for _, table := range tables {
		if strings.Compare(table.Name, PriceDbEntry.TableName(PriceDbEntry{})) == 0 {
			hypertable_exists = true
		}
	}

	if !hypertable_exists {
		log.Printf("Hypertable %s does not yet exist, creating...", PriceDbEntry.TableName(PriceDbEntry{}))
		db.Exec("SELECT create_hypertable(?, 'time')", PriceDbEntry.TableName(PriceDbEntry{}))
	} else {
		log.Printf("Hypertable %s already exists, skipping creation", PriceDbEntry.TableName(PriceDbEntry{}))
	}

	app.database = db

	return app, nil
}

type PriceSample struct {
	Prices      map[string]float32
	Time        time.Time
	Address     string
	GeoLocation string
	Id          uuid.UUID
}

type PriceDbEntry struct {
	FuelName    string
	Price       float32
	Time        time.Time `gorm:"not null"`
	Address     string
	GeoLocation string
	Id          uuid.UUID `gorm:"type:uuid"`
}

type Location struct {
	Identifier string
}

func (PriceDbEntry) TableName() string {
	return "pricemonitor"
}

func main() {
	app, err := NewPriceMonitorApplication(
		"10024747-saarbrucken-provinzialstr-2",
		"10024285-saarbruecken-lebacherstr",
		"10025246-saarbruecken-hochstr",
		"10024753-neunkirchen-fernstr",
		"10026417-neunkirchen-untere-bliesstr",
		"10024295-ottweiler-bliesstr",
		"10025634-puettlingen-bahnhofstr-76",
		"10024748-voelklingen-karolinger-str",
		"10026410-voelklingen-voelklinger-str",
		"10025240-ueberherrn-altforw-landstr",
		"10024623-saarlouis-lisdorfer-str",
		"10025255-wallerfangen-hauptstr",
		"10024293-homburg-bexbacher-str-74",
		"10025261-kirkel-a-d-windschnorr",
		"10025851-st-ingbert-suedstr-64",
		"10024752-neunkirchen-ludwigsthaler-s",
		"10025982-kleinblittersdorf-konrad-adenauer",
		"10025258-gersheim-bliestalstr",
		"10025259-homburg-ri-wagner-str")

	htmlquery.DisableSelectorCache = true

	if err != nil {
		panic(err)
	}

	collect_channel := make(chan PriceSample)

	go app.station_names()

	go app.collect_prices(collect_channel)

	for {
		for _, location := range app.locations {
			go app.scrape_price(fmt.Sprintf(constants.SCRAPE_URL, location.Identifier), collect_channel)
		}
		time.Sleep(time.Minute)
	}
}

func (app PriceMonitorApplication) station_names() {
	for {
		var count int64
		app.database.Raw("SELECT COUNT(*) FROM pg_matviews WHERE matviewname = 'pricemonitor_station_product_names';").Count(&count)

		if count == 0 {
			tx := app.database.Exec(`CREATE MATERIALIZED VIEW pricemonitor_station_product_names AS
			SELECT DISTINCT address, fuel_name
			FROM pricemonitor;`)

			if err := tx.Error; err != nil {
				fmt.Printf("Materialized view for station/product names could not be created '%s'", err.Error())
			}
		}

		tomorrow := time.Now().Add(time.Hour * 24)

		next_midnight := time.Until(time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 0, 0, 0, 0, time.Local))

		fmt.Printf("Created/refreshed materialized view for station/product names , sleeping until %s", next_midnight.String())

		time.Sleep(next_midnight)

		tx := app.database.Exec(`REFRESH MATERIALIZED VIEW my_materialized_view;`)
		if err := tx.Error; err != nil {
			fmt.Printf("Materialized view for station/product names could not be refreshed '%s'", err.Error())
		}
	}

}

func (app PriceMonitorApplication) collect_prices(rx <-chan PriceSample) {
	for {
		sample := <-rx
		for name, price := range sample.Prices {
			app.database.Model(&PriceDbEntry{}).Create(&PriceDbEntry{
				FuelName:    name,
				Price:       price,
				Time:        sample.Time,
				Address:     sample.Address,
				GeoLocation: sample.GeoLocation,
				Id:          sample.Id,
			})
		}
	}
}

func (app PriceMonitorApplication) scrape_price(url string, tx chan<- PriceSample) {
	bytes, err := json.Marshal(map[string]any{
		"url": url,
	})
	if err != nil {
		panic(err)
	}

	req, err := http.NewRequest(http.MethodPost, app.browserless_url, strings.NewReader(string(bytes)))
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Cache-Control", "no-cache")

	if err != nil {
		panic(err)
	}

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		retry_counter := 0

		for {
			fmt.Println("Encountered error making a request to browserless, retrying...")
			resp, err = http.DefaultClient.Do(req)

			if err != nil && retry_counter < 5 {
				retry_counter++
				continue
			} else {
				break
			}
		}

		if err != nil {
			panic(err)
		}
	}

	bytes, err = io.ReadAll(resp.Body)

	if err != nil {
		panic(err)
	}

	doc, err := htmlquery.Parse(strings.NewReader(string(bytes)))

	if err != nil {
		panic(err)
	}

	fuel_names := htmlquery.Find(doc, `//*[@class="station-page-fuel-prices__fuel-name"]`)
	fuel_prices := htmlquery.Find(doc, `//*[@class="station-page-fuel-prices__fuel-price"]`)

	if fuel_names == nil || fuel_prices == nil {
		fmt.Printf("Could not find fuel names or prices, aborting...\nReceived raw html: '%s'", string(bytes))
		return
	}

	address := htmlquery.FindOne(doc, `//*[@id="details"]/div/div[1]/div`).FirstChild.Data
	geolocation := htmlquery.FindOne(doc, `//*[@id="details"]/div/div[2]/div`).FirstChild.Data

	zipped_nodes := lo.Zip2(fuel_names, fuel_prices)

	prices := lo.SliceToMap(zipped_nodes, func(t lo.Tuple2[*html.Node, *html.Node]) (string, float32) {
		trimmed := strings.TrimPrefix(strings.TrimSuffix(t.B.FirstChild.Data, "/L"), "€")

		if t.A.FirstChild.Data == "Shell Recharge" {
			return t.A.FirstChild.Data, 0.0
		}

		price, err := strconv.ParseFloat(trimmed, 32)

		if err != nil {
			fmt.Printf("Error while parsing price: %s\n", err.Error())
		}

		return t.A.FirstChild.Data, float32(price)
	})

	price_sample := PriceSample{
		Prices:      prices,
		Time:        time.Now(),
		Address:     address,
		GeoLocation: geolocation,
		Id:          uuid.New(),
	}

	tx <- price_sample
}
