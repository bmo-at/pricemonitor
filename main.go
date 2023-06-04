package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

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
	Time        time.Time
	Address     string
	GeoLocation string
	Id          uuid.UUID `gorm:"type:uuid"`
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
		"10025259-homburg-ri-wagner-str",
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
			db.Exec("INSERT INTO 'pricemonitor' ('fuel_name','price','time','address','geo_location','id') VALUES (?,'?',?,?,?,?)", name, price, sample.Time, sample.Address, sample.GeoLocation, sample.Id)
		}
	}
}

func scrape_price(url string, tx chan<- PriceSample) {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	var address string
	var geolocation string
	var fuel_name_nodes []*cdp.Node
	var fuel_price_nodes []*cdp.Node
	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.Text(`.station-page-details__value[aria-labelledby=details-address]`, &address, chromedp.NodeVisible),
		chromedp.Text(`.station-page-details__value[aria-labelledby=details-lat_lng]`, &geolocation, chromedp.NodeVisible),
		chromedp.Nodes(`.station-page-fuel-prices__fuel-name`, &fuel_name_nodes, chromedp.NodeVisible),
		chromedp.Nodes(`.station-page-fuel-prices__fuel-price`, &fuel_price_nodes, chromedp.NodeVisible),
	)
	if err != nil {
		log.Fatal(err)
	}

	zipped_nodes := lo.Zip2(fuel_name_nodes, fuel_price_nodes)

	prices := lo.SliceToMap(zipped_nodes, func(t lo.Tuple2[*cdp.Node, *cdp.Node]) (string, float32) {
		trimmed := strings.TrimPrefix(strings.TrimSuffix(t.B.Children[len(t.B.Children)-1].NodeValue, "/L"), "€")

		price, err := strconv.ParseFloat(trimmed, 32)

		if err != nil {
			fmt.Printf("Error while parsing price: %s", err.Error())
		}

		return t.A.Children[len(t.A.Children)-1].NodeValue, float32(price)
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

	err = db.AutoMigrate(&PriceDbEntry{})

	if err != nil {
		log.Fatalf("failed to automigrate database: %s", err.Error())
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

	return db
}
