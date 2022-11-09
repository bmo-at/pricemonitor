package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type PriceSample struct {
	Prices      map[string]float32
	Time        time.Time
	Address     string
	GeoLocation string
}

type PriceDbEntry struct {
	FuelName    string
	Price       float32
	Time        time.Time
	Address     string
	GeoLocation string
}

func (PriceDbEntry) TableName() string {
	return "pricemonitor"
}

func main() {
	db := init_db()

	locations := []string{
		"10024747-saarbrucken-provinzialstr-2",
		"10024285-saarbruecken-lebacherstr",
		"10025246-saarbruecken-hochstr",
	}

	collect_channel := make(chan PriceSample)

	go collect_prices(collect_channel, db)

	for {
		for _, location := range locations {
			go scrape_price(fmt.Sprintf("https://find.shell.com/de/fuel/%s", location), collect_channel)
		}
		time.Sleep(time.Minute)
	}
}

func collect_prices(rx <-chan PriceSample, db *gorm.DB) {
	for {
		sample := <-rx
		for name, price := range sample.Prices {
			db.Create(PriceDbEntry{
				FuelName:    name,
				Price:       price,
				Time:        time.Now(),
				Address:     sample.Address,
				GeoLocation: sample.GeoLocation,
			})
		}
	}
}

func scrape_price(url string, tx chan<- PriceSample) {
	resp, err := http.Get(url)

	if err != nil {
		panic(err)
	}

	if resp.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", resp.StatusCode, resp.Status)
	}

	defer resp.Body.Close()

	document, err := goquery.NewDocumentFromReader(resp.Body)

	if err != nil {
		log.Fatal(err)
	}

	address := document.Find(".station-page-details__value[aria-labelledby=details-address]").Nodes[0].FirstChild.Data
	geolocation := document.Find(".station-page-details__value[aria-labelledby=details-lat_lng]").Nodes[0].FirstChild.Data

	prices := make(map[string]float32)

	for _, node := range document.Find(".station-page-fuel-prices__table-row").Nodes {
		trimmed := strings.TrimPrefix(strings.TrimSuffix(node.LastChild.FirstChild.Data, "/L"), "€")

		price, err := strconv.ParseFloat(trimmed, 32)

		if err != nil {
			fmt.Printf("Error while parsing price: %s", err.Error())
		}

		prices[node.FirstChild.FirstChild.Data] = float32(price)
	}

	price_sample := PriceSample{
		Prices:      prices,
		Time:        time.Now(),
		Address:     address,
		GeoLocation: geolocation,
	}

	tx <- price_sample
}

func init_db() *gorm.DB {
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

	db, err := gorm.Open(postgres.Open(fmt.Sprintf("host=%s user=%s password=%s dbname=postgres port=%s", db_host, db_user, db_password, db_port)), &gorm.Config{})

	if err != nil {
		log.Fatalf("failed to connect to database: %v", err.Error())
	}

	db.AutoMigrate(&PriceDbEntry{})

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

	return db
}
