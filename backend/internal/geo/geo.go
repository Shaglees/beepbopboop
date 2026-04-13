package geo

import "math"

const earthRadiusKm = 6371.0

// HaversineKm returns the great-circle distance in kilometres between two
// points specified in decimal degrees.
func HaversineKm(lat1, lon1, lat2, lon2 float64) float64 {
	dLat := toRad(lat2 - lat1)
	dLon := toRad(lon2 - lon1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusKm * c
}

// BoundingBox returns approximate min/max lat/lon for a circle of radiusKm
// around the given centre. Useful as a SQL pre-filter before Haversine.
func BoundingBox(lat, lon, radiusKm float64) (minLat, maxLat, minLon, maxLon float64) {
	latDelta := radiusKm / earthRadiusKm * (180.0 / math.Pi)
	lonDelta := radiusKm / (earthRadiusKm * math.Cos(toRad(lat))) * (180.0 / math.Pi)
	return lat - latDelta, lat + latDelta, lon - lonDelta, lon + lonDelta
}

func toRad(deg float64) float64 {
	return deg * math.Pi / 180.0
}
