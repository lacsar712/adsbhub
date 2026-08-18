// Package cpr implements Compact Position Reporting decode used by ADS-B.
package cpr

import (
	"fmt"
	"math"
)

const (
	Nb     = 17
	Nz     = 15
	MaxLat = 90.0
	MaxLon = 180.0
)

type Frame struct {
	Even bool
	Y    int
	X    int
}

func NL(lat float64) int {
	if lat < 0 {
		lat = -lat
	}
	if lat >= 87 {
		return 2
	}
	if lat <= 0 {
		return 59
	}
	a := 1 - math.Cos(math.Pi*2/float64(Nz))
	for nl := 59; nl >= 2; nl-- {
		v := math.Acos(1 - a/(math.Cos(math.Pi/180*lat)*math.Cos(math.Pi/180*lat)))
		if math.IsNaN(v) {
			return 2
		}
		if int(math.Floor(2*math.Pi/v)+0.5) == nl {
			return nl
		}
	}
	return 2
}

func dlat(even bool) float64 {
	nz := float64(Nz)
	if !even {
		nz = float64(Nz - 1)
	}
	return 360.0 / nz
}

func DecodeGlobal(even, odd Frame) (lat, lon float64, err error) {
	if even.Y < 0 || even.X < 0 || odd.Y < 0 || odd.X < 0 {
		return 0, 0, fmt.Errorf("cpr bin out of range")
	}
	den := float64(int(1) << Nb)
	j := math.Floor((float64(Nz-1)*float64(even.Y)-float64(Nz)*float64(odd.Y))/den + 0.5)
	rlat0 := dlat(true) * (mod(j, float64(Nz)) + float64(even.Y)/den)
	rlat1 := dlat(false) * (mod(j, float64(Nz-1)) + float64(odd.Y)/den)
	if rlat0 >= 270 {
		rlat0 -= 360
	}
	if rlat1 >= 270 {
		rlat1 -= 360
	}
	if NL(rlat0) != NL(rlat1) {
		return 0, 0, fmt.Errorf("cpr latitude zones disagree")
	}
	nl := NL(rlat0)
	ni0 := math.Max(float64(nl), 1)
	ni1 := math.Max(float64(nl-1), 1)
	m := math.Floor((float64(even.X)*(ni1)-float64(odd.X)*ni0)/den + 0.5)
	dlon0 := 360.0 / ni0
	lon0 := dlon0 * (mod(m, ni0) + float64(even.X)/den)
	if lon0 >= 180 {
		lon0 -= 360
	}
	if rlat0 < -MaxLat || rlat0 > MaxLat || lon0 < -MaxLon || lon0 > MaxLon {
		return 0, 0, fmt.Errorf("cpr decoded position out of range")
	}
	return rlat0, lon0, nil
}

func DecodeLocal(refLat, refLon float64, f Frame) (lat, lon float64, err error) {
	den := float64(int(1) << Nb)
	dLat := dlat(f.Even)
	j := math.Floor(refLat/dLat) + math.Floor(0.5+mod(refLat, dLat)/dLat-float64(f.Y)/den)
	lat = dLat * (j + float64(f.Y)/den)
	nl := float64(NL(lat))
	if !f.Even {
		nl = math.Max(nl-1, 1)
	} else if nl < 1 {
		nl = 1
	}
	dLon := 360.0 / nl
	m := math.Floor(refLon/dLon) + math.Floor(0.5+mod(refLon, dLon)/dLon-float64(f.X)/den)
	lon = dLon * (m + float64(f.X)/den)
	if lon >= 180 {
		lon -= 360
	}
	if lat < -MaxLat || lat > MaxLat {
		return 0, 0, fmt.Errorf("local cpr lat out of range")
	}
	return lat, lon, nil
}

func mod(a, n float64) float64 {
	if n == 0 {
		return a
	}
	return a - n*math.Floor(a/n)
}
