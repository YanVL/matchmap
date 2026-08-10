package location

import "github.com/umahmood/haversine"

func IsNearby(a, b Coordinates, radius float64) bool {
	_, kilometers := haversine.Distance(
		haversine.Coord{Lat: a.Latitude, Lon: a.Longitude},
		haversine.Coord{Lat: b.Latitude, Lon: b.Longitude},
	)

	meters := kilometers * 1000

	return meters <= radius
}
