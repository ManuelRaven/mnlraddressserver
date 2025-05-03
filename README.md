

# Go Address Server

A high-performance address lookup server with geocoding capabilities built in Go.

## Overview

Go Address Server provides a fast, lightweight solution for address management and geocoding operations. Built with a SQLite database backend for simplicity and performance, it offers a RESTful API for address searches and reverse geocoding.

## Features

- **Fulltext Address Search**: Quickly find addresses using text queries
- **Reverse Geocoding**: Find addresses near specific coordinates
- **Web Interface**: Simple web UI for searching addresses
- **Admin Panel**: Upload and manage address databases
- **Multi-platform Support**: Runs on Windows, macOS, and Linux
- **Docker Support**: Easy deployment using containers
- **Database Optimization**: High-performance SQLite configuration

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/manuelraven/go-address-server.git
cd go-address-server

# Build the executable
go build -o mnlraddressserver
```

### Using Docker

```bash
docker pull ghcr.io/manuelraven/mnlraddressserver:latest
```

## Running the Server

### From Binary

```bash
# Build and run
go build
./mnlraddressserver
```

Or run directly with Go:

```bash
go run main.go
```

The server will start on port 8809 by default.

### Using Docker

```bash
docker run -d \
  --name mnlraddressserver \
  -p 8809:8809 \
  -v ./data:/data \
  ghcr.io/manuelraven/mnlraddressserver:latest
```

## API Endpoints

### Address Search

```
GET /api/search?q=query
```

Parameters:
- `q`: Search query (required)

Example:
```
GET /api/search?q=Hauptstraße Berlin
```

Returns address matches based on a fulltext search algorithm, with results sorted by relevance.

### Reverse Geocoding

```
GET /api/reverse?lat=latitude&lon=longitude&radius=1.0&limit=10
```

Parameters:
- `lat`: Latitude coordinate (required)
- `lon`: Longitude coordinate (required)
- `radius`: Search radius in kilometers (default: 1.0, min: 0.01, max: 10.0)
- `limit`: Maximum number of results (default: 10, max: 100)

Example:
```
GET /api/reverse?lat=52.520008&lon=13.404954&radius=0.5
```

Returns addresses nearest to the given coordinates, sorted by distance.

## Web Interface

The server includes a web interface for searching addresses:

- **Main Interface**: Access at `http://localhost:8809/`
- **Admin Panel**: Access at `http://localhost:8809/admin.html`

## Database Management

The Address Server allows uploading and switching database files without restarting:

1. Navigate to the Admin interface at `http://localhost:8809/admin.html`
2. Click "Choose File" and select your SQLite database file (must have a `.db` extension)
3. Click "Upload Database" to start the upload process
4. The server will:
   - Upload and validate the file
   - Close the current database connection
   - Back up the current database (if any)
   - Replace the database file
   - Initialize a new connection to the new database

### Large Database Files

The system is designed to handle multi-gigabyte database files:

- File uploads are streamed to disk to minimize memory usage
- The server uses optimized SQLite settings for performance with large databases:
  - WAL journal mode
  - Memory-mapped I/O
  - Increased cache size
  - Single connection to prevent locking issues

## Building with GoReleaser

The project includes a GoReleaser configuration for building cross-platform binaries:

```bash
goreleaser build --snapshot --clean
```

## Performance Considerations

For large databases, consider adjusting the cache and memory-mapped I/O settings in the code according to your available memory.

## License

[MIT](LICENSE)