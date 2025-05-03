package sql

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"strings"

	_ "modernc.org/sqlite"
)

var dbpath = "data/data.db"
var db *sql.DB

// GetDBPath returns the path to the database file
func GetDBPath() string {
	return dbpath
}

// Address represents an address record from the database
type Address struct {
	ID          int64   `json:"id"`
	Street      string  `json:"street"`
	HouseNumber string  `json:"house_number"`
	City        string  `json:"city"`
	Longitude   float64 `json:"longitude"`
	Latitude    float64 `json:"latitude"`
}

// Init initializes the database connection
func Init() error {
	var err error
	db, err = sql.Open("sqlite", dbpath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Set pragmas for optimal performance
	pragmas := []string{
		"PRAGMA page_size = 16384;",
		"PRAGMA cache_size = -4000;",      // Use up to 4GB of cache
		"PRAGMA journal_mode = WAL;",      // Use write-ahead logging for better concurrency
		"PRAGMA synchronous = NORMAL;",    // Less fsync for better performance
		"PRAGMA mmap_size = 30000000000;", // Use memory-mapped I/O
		"PRAGMA temp_store = MEMORY;",     // Store temp tables in memory
	}

	for _, pragma := range pragmas {
		_, err = db.Exec(pragma)
		if err != nil {
			db.Close()
			log.Printf("Warning: failed to set pragma %s: %v", pragma, err)
			return fmt.Errorf("failed to set pragma %s: %w", pragma, err)
		}
	}
	log.Println("Database initialized with optimizations.")
	return nil
}

// Close closes the database connection
func Close() error {
	if db != nil {
		return db.Close()
	}
	return nil
}

// SearchByAddress searches for addresses by street, house number, and/or city
func SearchByAddress(street, houseNumber, city string) ([]Address, error) {
	var addresses []Address
	var args []interface{}
	query := "SELECT id, street, house_number, city, longitude, latitude FROM addresses WHERE 1=1"

	if street != "" {
		query += " AND street LIKE ?"
		args = append(args, street+"%")
	}

	if houseNumber != "" {
		query += " AND house_number = ?"
		args = append(args, houseNumber)
	}

	if city != "" {
		query += " AND city LIKE ?"
		args = append(args, city+"%")
	}

	query += " LIMIT 100"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("search query failed: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var addr Address
		if err := rows.Scan(&addr.ID, &addr.Street, &addr.HouseNumber, &addr.City, &addr.Longitude, &addr.Latitude); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		addresses = append(addresses, addr)
	}

	return addresses, nil
}

// FulltextSearch performs a full-text search using the FTS5 virtual table
func FulltextSearch(query string) ([]Address, error) {
	var addresses []Address

	// Add an asterisk to each term to enable prefix matching
	// This allows partial word matches like "Hauptstraß*" to match "Hauptstraße"
	words := strings.Fields(query)
	for i, word := range words {
		words[i] = word + "*"
	}
	modifiedQuery := strings.Join(words, " ")

	sqlQuery := `
		SELECT a.id, a.street, a.house_number, a.city, a.longitude, a.latitude
		FROM address_fts
		JOIN addresses a ON address_fts.rowid = a.id
		WHERE address_fts MATCH ?
		ORDER BY rank
		LIMIT 100
	`

	rows, err := db.Query(sqlQuery, modifiedQuery)
	if err != nil {
		return nil, fmt.Errorf("fulltext search failed: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var addr Address
		if err := rows.Scan(&addr.ID, &addr.Street, &addr.HouseNumber, &addr.City, &addr.Longitude, &addr.Latitude); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		addresses = append(addresses, addr)
	}

	return addresses, nil
}

// GetAddressById retrieves an address by its ID
func GetAddressById(id int64) (*Address, error) {
	var addr Address

	err := db.QueryRow("SELECT id, street, house_number, city, longitude, latitude FROM addresses WHERE id = ?", id).
		Scan(&addr.ID, &addr.Street, &addr.HouseNumber, &addr.City, &addr.Longitude, &addr.Latitude)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No address found
		}
		return nil, fmt.Errorf("get address by id failed: %w", err)
	}

	return &addr, nil
}

// FindAddressesInRadius finds addresses within a specified radius (in km) of a point
func FindAddressesInRadius(latitude, longitude float64, radiusKm float64) ([]Address, error) {
	var addresses []Address

	// Haversine formula in SQL to calculate distance
	query := `
		SELECT id, street, house_number, city, longitude, latitude,
		       (6371 * acos(cos(radians(?)) * cos(radians(latitude)) * 
		       cos(radians(longitude) - radians(?)) + 
		       sin(radians(?)) * sin(radians(latitude)))) AS distance 
		FROM addresses 
		WHERE (6371 * acos(cos(radians(?)) * cos(radians(latitude)) * 
		      cos(radians(longitude) - radians(?)) + 
		      sin(radians(?)) * sin(radians(latitude)))) < ? 
		ORDER BY distance 
		LIMIT 100
	`
	rows, err := db.Query(query, latitude, longitude, latitude, latitude, longitude, latitude, radiusKm)
	if err != nil {
		return nil, fmt.Errorf("radius search failed: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var addr Address
		var distance float64
		if err := rows.Scan(&addr.ID, &addr.Street, &addr.HouseNumber, &addr.City,
			&addr.Longitude, &addr.Latitude, &distance); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		addresses = append(addresses, addr)
	}

	return addresses, nil
}

// GetAddressesByCity gets addresses for a specific city with pagination
func GetAddressesByCity(city string, page, pageSize int) ([]Address, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 1000 {
		pageSize = 100 // Default page size with a maximum
	}

	offset := (page - 1) * pageSize

	var addresses []Address
	query := "SELECT id, street, house_number, city, longitude, latitude FROM addresses WHERE city = ? LIMIT ? OFFSET ?"

	rows, err := db.Query(query, city, pageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("get addresses by city failed: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var addr Address
		if err := rows.Scan(&addr.ID, &addr.Street, &addr.HouseNumber, &addr.City, &addr.Longitude, &addr.Latitude); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		addresses = append(addresses, addr)
	}

	return addresses, nil
}

// GetAddressCount returns the total count of addresses in the database
func GetAddressCount() (int64, error) {
	var count int64
	err := db.QueryRow("SELECT COUNT(*) FROM addresses").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count query failed: %w", err)
	}
	return count, nil
}

// GetCitySummary returns the count of addresses in each city
func GetCitySummary() (map[string]int64, error) {
	result := make(map[string]int64)

	rows, err := db.Query("SELECT city, COUNT(*) as count FROM addresses GROUP BY city ORDER BY count DESC LIMIT 1000")
	if err != nil {
		return nil, fmt.Errorf("city summary query failed: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var city string
		var count int64
		if err := rows.Scan(&city, &count); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		result[city] = count
	}

	return result, nil
}

// CalculateDistance calculates the distance between two coordinates using the Haversine formula
func CalculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371.0 // Earth radius in kilometers

	// Convert degrees to radians
	lat1Rad := lat1 * math.Pi / 180.0
	lon1Rad := lon1 * math.Pi / 180.0
	lat2Rad := lat2 * math.Pi / 180.0
	lon2Rad := lon2 * math.Pi / 180.0

	// Differences
	dLat := lat2Rad - lat1Rad
	dLon := lon2Rad - lon1Rad

	// Haversine formula
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c // Distance in kilometers
}

// HighlightedMatch represents a search result with highlighted matches
type HighlightedMatch struct {
	Address          Address `json:"address"`
	StreetMatch      string  `json:"street_match,omitempty"`
	HouseNumberMatch string  `json:"house_number_match,omitempty"`
	CityMatch        string  `json:"city_match,omitempty"`
}

// AdvancedFulltextSearch performs a column-specific fulltext search with highlighting
// Optional parameters:
// - street: search only in street column
// - houseNumber: search only in house_number column
// - city: search only in city column
// If parameter is empty, all columns will be searched
func AdvancedFulltextSearch(query string, limit int, highlight bool) ([]interface{}, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100 // Default limit with a maximum
	}

	// Add an asterisk to each term to enable prefix matching
	// This allows partial word matches like "Hauptstraß*" to match "Hauptstraße"
	words := strings.Fields(query)
	for i, word := range words {
		words[i] = word + "*"
	}
	modifiedQuery := strings.Join(words, " ")

	var results []interface{}
	var sqlQuery string
	if highlight {
		// Query with highlighting
		sqlQuery = `
			SELECT a.id, a.street, a.house_number, a.city, a.longitude, a.latitude,
				highlight(address_fts, 0, '<b>', '</b>') as street_match,
				highlight(address_fts, 1, '<b>', '</b>') as house_number_match,
				highlight(address_fts, 2, '<b>', '</b>') as city_match
			FROM address_fts
			JOIN addresses a ON address_fts.rowid = a.id
			WHERE address_fts MATCH ?
			ORDER BY rank
			LIMIT ?
		`

		rows, err := db.Query(sqlQuery, modifiedQuery, limit)
		if err != nil {
			return nil, fmt.Errorf("advanced fulltext search failed: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var result HighlightedMatch
			var addr Address
			if err := rows.Scan(
				&addr.ID, &addr.Street, &addr.HouseNumber, &addr.City,
				&addr.Longitude, &addr.Latitude,
				&result.StreetMatch, &result.HouseNumberMatch, &result.CityMatch,
			); err != nil {
				return nil, fmt.Errorf("scan failed: %w", err)
			}
			result.Address = addr
			results = append(results, result)
		}
	} else {
		// Query without highlighting
		sqlQuery = `
			SELECT a.id, a.street, a.house_number, a.city, a.longitude, a.latitude
			FROM address_fts
			JOIN addresses a ON address_fts.rowid = a.id
			WHERE address_fts MATCH ?
			ORDER BY rank
			LIMIT ?
		`

		rows, err := db.Query(sqlQuery, modifiedQuery, limit)
		if err != nil {
			return nil, fmt.Errorf("advanced fulltext search failed: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var addr Address
			if err := rows.Scan(&addr.ID, &addr.Street, &addr.HouseNumber, &addr.City,
				&addr.Longitude, &addr.Latitude); err != nil {
				return nil, fmt.Errorf("scan failed: %w", err)
			}
			results = append(results, addr)
		}
	}

	return results, nil
}
