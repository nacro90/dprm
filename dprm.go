package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	observatoryURL                = "http://www.koeri.boun.edu.tr/scripts/lst4.asp"
	defaultMaxDepth       float32 = 70
	defaultMinMagnitude           = 4.5
	earthquakeLinePattern         = `(\d{4}\.\d{2}\.\d{2})\s(\d{2}:\d{2}:\d{2})\s+(\d+\.\d+)\s+(\d+\.\d+)\s+(\d+\.\d+)\s+[^\s]+\s+(\d+\.\d+)\s+[^\s]+\s+(\w+-(\w+)?) ?\(\w+\)`
)

var eqLineRegex = regexp.MustCompile(earthquakeLinePattern)

type Config struct {
	All          bool
	MaxDepth     float32
	MinMagnitude float32
}

type Earthquake struct {
	Location  string
	Latitude  float64
	Longitude float64
	Time      time.Time
	Magnitude float32
	Depth     float32
}

func main() {
	cfg := newConfig()
	earthquakes := getEarthquakes(cfg)
	printEarthquakes(earthquakes)
}

func newConfig() Config {
	all := flag.Bool("a", false, "do not filter unimportant earthqakes")
	maxDepth := flag.Float64(
		"d",
		float64(defaultMaxDepth),
		"max depth of an important earthquake in kilometers",
	)
	minMagnitude := flag.Float64(
		"m",
		float64(defaultMinMagnitude),
		"min magnitude of an important earthquake",
	)
	flag.Parse()
	return Config{
		All:          *all,
		MaxDepth:     float32(*maxDepth),
		MinMagnitude: float32(*minMagnitude),
	}
}

func getEarthquakes(cfg Config) []Earthquake {
	page, err := getObservatoryPage(observatoryURL)
	if err != nil {
		fmt.Printf("error while getting observatory page: %s", err)
		os.Exit(1)
	}
	var eqs []Earthquake
	for _, line := range strings.Split(page, "\n") {
		if !eqLineRegex.MatchString(line) {
			continue
		}
		eq, err := parseLine(line)
		if err != nil {
			fmt.Printf("error while parsing earthquake line line=%s: %s", line, err)
			continue
		}
		if !cfg.All && !isImportant(cfg, eq) {
			continue
		}
		eqs = append(eqs, eq)
	}
	return eqs
}

func getObservatoryPage(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf(
			"error while getting earthquakes from observatory, url=%s: %w",
			url,
			err,
		)
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error while reading response from observatory, url=%s: %w", url, err)
	}
	return string(bodyBytes), nil
}

func parseLine(line string) (Earthquake, error) {
	matches := eqLineRegex.FindStringSubmatch(line)
	datetimeStr := fmt.Sprintf("%s %s", matches[1], matches[2])
	turkeyLoc, err := time.LoadLocation("Europe/Istanbul")
	if err != nil {
		return Earthquake{}, fmt.Errorf("error while parsing location string: %s", err)
	}
	datetime, err := time.ParseInLocation("2006.01.02 15:04:05", datetimeStr, turkeyLoc)
	if err != nil {
		return Earthquake{}, fmt.Errorf(
			"error while parsing date of the earthquake datetimeStr=%s: %w",
			datetimeStr,
			err,
		)
	}
	latStr := matches[3]
	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		return Earthquake{}, fmt.Errorf(
			"error while parsing latitude of the earthquake latStr=%s: %w",
			latStr,
			err,
		)
	}
	longStr := matches[4]
	long, err := strconv.ParseFloat(longStr, 64)
	if err != nil {
		return Earthquake{}, fmt.Errorf(
			"error while parsing longitude of the earthquake latStr=%s: %w",
			longStr,
			err,
		)
	}
	depthStr := matches[5]
	depth, err := strconv.ParseFloat(depthStr, 32)
	if err != nil {
		return Earthquake{}, fmt.Errorf(
			"error while parsing depth of the earthquake depthStr=%s: %w",
			depthStr,
			err,
		)
	}
	magStr := matches[6]
	mag, err := strconv.ParseFloat(magStr, 32)
	if err != nil {
		return Earthquake{}, fmt.Errorf(
			"error while parsing magnitude of the earthquake magStr=%s: %w",
			magStr,
			err,
		)
	}
	epicenter := matches[7]
	province := matches[8]
	localLoc, err := time.LoadLocation("Local")
	if err != nil {
		return Earthquake{}, fmt.Errorf("error while parsing time location: %s", err)
	}
	return Earthquake{
		Location:  fmt.Sprintf("%s %s", province, epicenter),
		Latitude:  lat,
		Longitude: long,
		Time:      datetime.In(localLoc),
		Magnitude: float32(mag),
		Depth:     float32(depth),
	}, nil
}

func isImportant(cfg Config, eq Earthquake) bool {
	return eq.Magnitude > cfg.MinMagnitude && eq.Depth < cfg.MaxDepth
}

func printEarthquakes(eqs []Earthquake) {
	if len(eqs) == 0 {
		fmt.Println("No important earthquakes recently")
		return
	}
	maxLocLength := 0
	for _, eq := range eqs {
		if maxLocLength < len(eq.Location) {
			maxLocLength = len(eq.Location)
		}
	}
	for _, eq := range eqs {
		formatStr := fmt.Sprintf("%%-%ds\t%%1.1fM\t%%02.1fkm\t%%s\n", maxLocLength)
		fmt.Printf(formatStr, eq.Location, eq.Magnitude, eq.Depth, eq.Time.Format(time.DateTime))
	}
}
