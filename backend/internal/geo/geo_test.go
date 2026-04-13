package geo

import (
	"math"
	"testing"
)

func TestHaversine_DublinToLondon(t *testing.T) {
	// Dublin (53.3498, -6.2603) → London (51.5074, -0.1278) ≈ 464 km
	d := HaversineKm(53.3498, -6.2603, 51.5074, -0.1278)
	if math.Abs(d-464) > 5 {
		t.Errorf("expected ~464 km, got %.1f km", d)
	}
}

func TestHaversine_ZeroDistance(t *testing.T) {
	d := HaversineKm(53.3498, -6.2603, 53.3498, -6.2603)
	if d != 0 {
		t.Errorf("expected 0 km, got %.6f km", d)
	}
}

func TestBoundingBox_Sanity(t *testing.T) {
	lat, lon, radius := 53.3498, -6.2603, 25.0
	minLat, maxLat, minLon, maxLon := BoundingBox(lat, lon, radius)

	if minLat >= lat || maxLat <= lat {
		t.Errorf("lat bounds invalid: min=%f max=%f centre=%f", minLat, maxLat, lat)
	}
	if minLon >= lon || maxLon <= lon {
		t.Errorf("lon bounds invalid: min=%f max=%f centre=%f", minLon, maxLon, lon)
	}

	// Points at the edges should be roughly radiusKm away
	edgeDist := HaversineKm(lat, lon, maxLat, lon)
	if math.Abs(edgeDist-radius) > 1 {
		t.Errorf("expected edge distance ~%f km, got %.1f km", radius, edgeDist)
	}
}
