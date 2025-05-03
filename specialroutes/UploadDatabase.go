package specialroutes

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"mnlr.de/addressserver/sql"
)

type UploadResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func FileUploadHandler(w http.ResponseWriter, r *http.Request) {
	// Defer error recovery to ensure we always return a valid JSON response
	defer func() {
		if r := recover(); r != nil {
			sendJSONResponse(w, UploadResponse{
				Success: false,
				Message: fmt.Sprintf("Internal server error: %v", r),
			})
		}
	}()

	if r.Method != http.MethodPost {
		sendJSONResponse(w, UploadResponse{
			Success: false,
			Message: "Invalid request method. Only POST is allowed.",
		})
		return
	}

	// Validate that we're getting a multipart form
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		sendJSONResponse(w, UploadResponse{
			Success: false,
			Message: "Invalid Content-Type. Expected multipart/form-data",
		})
		return
	}

	// Parse the multipart form with 32MB max memory
	const maxMemory = 32 << 20 // 32 MB
	if err := r.ParseMultipartForm(maxMemory); err != nil {
		sendJSONResponse(w, UploadResponse{
			Success: false,
			Message: "Failed to parse form: " + err.Error(),
		})
		return
	}

	// Get the file from the form
	file, header, err := r.FormFile("dbFile")
	if err != nil {
		sendJSONResponse(w, UploadResponse{
			Success: false,
			Message: "Failed to get the uploaded file: " + err.Error(),
		})
		return
	}

	// Verify that the file has a .db extension
	if !strings.HasSuffix(strings.ToLower(header.Filename), ".db") {
		sendJSONResponse(w, UploadResponse{
			Success: false,
			Message: "Invalid file type. Only .db files are allowed.",
		})
		return
	}

	// Create data directory if it doesn't exist
	if err := os.MkdirAll("data", 0755); err != nil {
		sendJSONResponse(w, UploadResponse{
			Success: false,
			Message: "Failed to create data directory: " + err.Error(),
		})
		return
	}

	// Create a new file to save the upload
	uploadPath := filepath.Join("data", "upload.db")
	out, err := os.Create(uploadPath)
	if err != nil {
		sendJSONResponse(w, UploadResponse{
			Success: false,
			Message: "Failed to create upload file: " + err.Error(),
		})
		return
	}

	// Stream the file to disk
	_, err = io.Copy(out, file)
	if err != nil {
		sendJSONResponse(w, UploadResponse{
			Success: false,
			Message: "Failed to write file: " + err.Error(),
		})
		return
	}
	out.Close()
	file.Close()
	fmt.Println("File uploaded successfully")

	// Close the current database connection
	if err := sql.Close(); err != nil {
		sendJSONResponse(w, UploadResponse{
			Success: false,
			Message: "Failed to close current database: " + err.Error(),
		})
		return
	}

	fmt.Println("Current database closed successfully")

	// Move the current database to a backup location
	backupPath := filepath.Join("data", "backup.db")
	if err := os.Rename(sql.GetDBPath(), backupPath); err != nil {
		sendJSONResponse(w, UploadResponse{
			Success: false,
			Message: "Failed to backup current database: " + err.Error(),
		})
		return
	}

	fmt.Println("Current database backed up successfully")

	// Move the uploaded file to replace the current database
	currentDB := filepath.Join("data", "data.db")
	if err := os.Rename(uploadPath, currentDB); err != nil {
		fmt.Print("Failed to replace database file: ", err)
		sendJSONResponse(w, UploadResponse{
			Success: false,
			Message: "Failed to replace database file: " + err.Error(),
		})
		return
	}

	fmt.Println("Database file replaced successfully")

	// Reinitialize the database connection
	if err := sql.Init(); err != nil {

		// Restore from backup if initialization fails (remove data.db and rename backup.db back to data.db)
		if err := os.Remove(currentDB); err != nil {
			sendJSONResponse(w, UploadResponse{
				Success: false,
				Message: fmt.Sprintf("New DB are invalid ,Failed to restore backup database file:  %v", err),
			})
			return
		}
		if err := os.Rename(backupPath, currentDB); err != nil {
			sendJSONResponse(w, UploadResponse{
				Success: false,
				Message: fmt.Sprintf("Failed to restore backup database file:  %v", err),
			})
			return
		}

		// Reinitialize the database connection
		if err := sql.Init(); err != nil {
			sendJSONResponse(w, UploadResponse{
				Success: false,
				Message: fmt.Sprintf("Failed to reinitialize database after restoring backup: %v", err),
			})
			return
		}

		sendJSONResponse(w, UploadResponse{
			Success: false,
			Message: "New DB are invalid, database restored from backup",
		})

		return
	}

	fmt.Println("New database initialized successfully")

	sendJSONResponse(w, UploadResponse{
		Success: true,
		Message: "Database updated successfully",
	})
	RemoveBackupFile()
}

func RemoveBackupFile() {
	backupPath := filepath.Join("data", "backup.db")
	if _, err := os.Stat(backupPath); err == nil {
		if err := os.Remove(backupPath); err != nil {
			fmt.Println("Failed to remove backup file:", err)
		} else {
			fmt.Println("Backup file removed successfully")
		}
	} else if os.IsNotExist(err) {
		fmt.Println("No backup file found to remove")
	} else {
		fmt.Println("Error checking for backup file:", err)
	}
}

func sendJSONResponse(w http.ResponseWriter, response UploadResponse) {
	// Set headers first
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")

	// Then set status code
	if !response.Success {
		w.WriteHeader(http.StatusBadRequest) // Using 400 for client errors
	} else {
		w.WriteHeader(http.StatusOK)
	}

	// Print error message to console for debugging
	if !response.Success {
		fmt.Println("Error:", response.Message)
	}

	// Marshal and write response
	jsonData, err := json.Marshal(response)
	if err != nil {
		// If JSON marshaling fails, send a basic error response
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"success":false,"message":"Internal Server Error"}`))
		return
	}

	w.Write(jsonData)
}
