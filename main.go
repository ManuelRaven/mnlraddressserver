package main

import (
	"embed"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
	"mnlr.de/addressserver/specialroutes"
	"mnlr.de/addressserver/sql"
)

//go:embed public
var publicFS embed.FS

func main() {

	// Create relative data directory if it doesn't exist
	if _, err := os.Stat("./data"); os.IsNotExist(err) {
		if err := os.Mkdir("./data", 0755); err != nil {
			panic("Failed to create data directory: " + err.Error())
		}
	}

	// Initialize the database
	if err := sql.Init(); err != nil {
		panic("Failed to initialize database: " + err.Error())
	}
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		sql.Close()
		os.Exit(1)
	}()

	// Create a new router & API
	mux := http.NewServeMux()
	mux.HandleFunc("/adminapi/database/upload", specialroutes.FileUploadHandler)
	mux.HandleFunc("/adminapi/hello", specialroutes.Hellohandler)
	config := huma.DefaultConfig("My API", "1.0.0")
	config.Servers = []*huma.Server{{URL: "/api"}}
	api := humago.NewWithPrefix(mux, "/api", config)
	publicDir, err := fs.Sub(publicFS, "public")
	if err != nil {
		panic("Failed to access public directory: " + err.Error())
	}
	fs := http.FileServer(http.FS(publicDir))
	mux.Handle("/", fs)

	RegisterApi(api)

	// Start the server!
	http.ListenAndServe("0.0.0.0:8809", mux)

}
