package routes

import (
	"context"
	"fmt"
	"strings"

	"mnlr.de/addressserver/sql"
)

// FulltextSearchInput represents the input for fulltext search.
type FulltextSearchInput struct {
	Query string `query:"q" example:"main street" doc:"The search query"`
}

// FulltextSearchOutput represents the fulltext search operation response.
type FulltextSearchOutput struct {
	Body struct {
		Addresses []sql.Address `json:"addresses" doc:"Matching addresses"`
	}
}

// FulltextSearch performs a fulltext search on the address database.
func FulltextSearch(ctx context.Context, input *FulltextSearchInput) (*FulltextSearchOutput, error) {
	if input.Query == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}

	// Replace commas with spaces in the query
	input.Query = strings.ReplaceAll(input.Query, ",", " ")

	addresses, err := sql.FulltextSearch(input.Query)
	if err != nil {
		return nil, fmt.Errorf("fulltext search failed: %w", err)
	}

	resp := &FulltextSearchOutput{}
	resp.Body.Addresses = addresses
	return resp, nil
}
