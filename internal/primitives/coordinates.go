package primitives

import "fmt"

// Coordinates is a geo position of a route point.
type Coordinates struct {
	Latitude  float64
	Longitude float64
}

// NewCoordinates validates and builds coordinates.
func NewCoordinates(latitude, longitude float64) (Coordinates, error) {
	if latitude < -90 || latitude > 90 {
		return Coordinates{}, fmt.Errorf("latitude out of range: %f", latitude)
	}
	if longitude < -180 || longitude > 180 {
		return Coordinates{}, fmt.Errorf("longitude out of range: %f", longitude)
	}
	return Coordinates{Latitude: latitude, Longitude: longitude}, nil
}