package main

import (
	"github.com/danielgtaylor/huma/v2"
	"mnlr.de/addressserver/routes"
)

func RegisterApi(api huma.API) {

	// Register GET /search handler for fulltext search.
	huma.Get(api, "/search", routes.FulltextSearch)

	// Register GET /reverse handler for reverse geocoding.
	huma.Get(api, "/reverse", routes.ReverseGeocode)

}
