package stations

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Brand string

type Station interface {
	ScrapePrices() (Sample, error)
	Identifier() string
}

type Sample struct {
	Prices      map[string]float32
	Time        time.Time
	Address     string
	GeoLocation string
	ID          uuid.UUID
	Brand       string
}

var identifierRegex = regexp.MustCompile(`^(aral:[A-z-]+/[A-z0-9-]+/[0-9]+)|(shell:[0-9]+-[0-9A-z-]+)$`)

//nolint:ireturn // We need to return an interface here
func NewStation(identifier string) (Station, error) {
	identifier = strings.TrimSpace(identifier)

	if !identifierRegex.MatchString(identifier) {
		return nil, errors.New("identifier does not match the format " +
			"('brand:station-identifier'), i.e " +
			"shell:10027720-erfurt-bei-den-froschackern-2" +
			" or " +
			"aral:st-ingbert/ensheimer-strasse-152/18111200")
	}

	splitIdentifier := strings.Split(identifier, ":")
	brand := Brand(splitIdentifier[0])
	identifierWithoutBrand := splitIdentifier[1]

	switch brand {
	case BrandShell:
		return StationShell{
			url:   "https://find.shell.com/de/fuel/" + identifierWithoutBrand,
			brand: brand,
		}, nil
	case BrandAral:
		split := strings.Split(identifierWithoutBrand, "/")
		id := split[2]

		return StationAral{
			urlMainPage: "https://tankstelle.aral.de/" + identifierWithoutBrand,
			urlAPI:      "https://api.tankstelle.aral.de/api/v3/stations/" + id + "/prices",
			brand:       brand,
		}, nil
	default:
		return nil, errors.New("Unknown brand")
	}
}
