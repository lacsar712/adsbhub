// Package geo validates WGS-84 positions used in ADS-B position reports.
package geo

import (
	"fmt"
	"math"
)

const (
	MinLat = -90.0
	MaxLat = 90.0
	MinLon = -180.0
	MaxLon = 180.0
)

type Point struct {
	Lat float64
	Lon float64
}

func Validate(lat, lon float64) error {
	if math.IsNaN(lat) || math.IsInf(lat, 0) {
		return fmt.Errorf("latitude is not a finite number")
	}
	if math.IsNaN(lon) || math.IsInf(lon, 0) {
		return fmt.Errorf("longitude is not a finite number")
	}
	if lat < MinLat || lat > MaxLat {
		return fmt.Errorf("latitude %v outside [-90,90]", lat)
	}
	if lon < MinLon || lon > MaxLon {
		return fmt.Errorf("longitude %v outside [-180,180]", lon)
	}
	return nil
}

func NormalizeLon(lon float64) float64 {
	for lon < -180 {
		lon += 360
	}
	for lon > 180 {
		lon -= 360
	}
	return lon
}

func HaversineNM(a, b Point) float64 {
	const earthNM = 3440.065
	lat1 := a.Lat * math.Pi / 180
	lat2 := b.Lat * math.Pi / 180
	dLat := (b.Lat - a.Lat) * math.Pi / 180
	dLon := (b.Lon - a.Lon) * math.Pi / 180
	sinDLat := math.Sin(dLat / 2)
	sinDLon := math.Sin(dLon / 2)
	h := sinDLat*sinDLat + math.Cos(lat1)*math.Cos(lat2)*sinDLon*sinDLon
	return 2 * earthNM * math.Asin(math.Min(1, math.Sqrt(h)))
}

func InBox(p Point, south, west, north, east float64) bool {
	if south > north {
		south, north = north, south
	}
	if p.Lat < south || p.Lat > north {
		return false
	}
	if west <= east {
		return p.Lon >= west && p.Lon <= east
	}
	return p.Lon >= west || p.Lon <= east
}
