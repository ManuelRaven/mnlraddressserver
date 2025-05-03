package routes

import (
	"context"
	"fmt"

	"mnlr.de/addressserver/sql"
)

// ReverseGeocodeInput represents the input for reverse geocoding.
type ReverseGeocodeInput struct {
	Latitude  float64 `query:"lat" example:"49.4521" doc:"Latitude coordinate"`
	Longitude float64 `query:"lon" example:"11.0767" doc:"Longitude coordinate"`
	RadiusKm  float64 `query:"radius" default:"1.0" min:"0.01" max:"10.0" doc:"Search radius in kilometers"`
	Limit     int     `query:"limit" default:"10" min:"1" max:"100" doc:"Maximum number of results to return"`
}

// ReverseGeocodeOutput represents the reverse geocode operation response.
type ReverseGeocodeOutput struct {
	Body struct {
		Addresses []sql.Address `json:"addresses" doc:"Addresses found near the coordinates"`
	}
}

// ReverseGeocode takes coordinates and returns addresses near that location.
func ReverseGeocode(ctx context.Context, input *ReverseGeocodeInput) (*ReverseGeocodeOutput, error) {
	// Input validation
	if input.Latitude < -90 || input.Latitude > 90 {
		return nil, fmt.Errorf("latitude must be between -90 and 90")
	}
	if input.Longitude < -180 || input.Longitude > 180 {
		return nil, fmt.Errorf("longitude must be between -180 and 180")
	}

	// Use default values if not specified
	radiusKm := input.RadiusKm
	if radiusKm == 0 {
		radiusKm = 1.0
	}

	// Find addresses in the specified radius
	addresses, err := sql.FindAddressesInRadius(input.Latitude, input.Longitude, radiusKm)
	if err != nil {
		return nil, fmt.Errorf("reverse geocoding failed: %w", err)
	}

	// Apply limit if provided
	if input.Limit > 0 && input.Limit < len(addresses) {
		addresses = addresses[:input.Limit]
	}

	// Return results
	resp := &ReverseGeocodeOutput{}
	resp.Body.Addresses = addresses
	return resp, nil
}
