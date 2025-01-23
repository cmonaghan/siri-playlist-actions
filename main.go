package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// RequestBody is the structure for the incoming POST request
type RequestBody struct {
	SongID     string `json:"song_id"`
	PlaylistID string `json:"playlist_id"`
}

func removeSongFromPlaylist(w http.ResponseWriter, r *http.Request) {
	// Only accept POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Decode the incoming JSON request body
	var requestBody RequestBody
	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		http.Error(w, "Failed to decode JSON", http.StatusBadRequest)
		return
	}

	// Print the data (simulating the removal of the song)
	log.Printf("Received request to remove song with ID: %s from playlist with ID: %s", requestBody.SongID, requestBody.PlaylistID)

	// Respond with a success message
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": fmt.Sprintf("Removed song %s from playlist %s", requestBody.SongID, requestBody.PlaylistID),
	})
}

func main() {
	http.HandleFunc("/remove-song", removeSongFromPlaylist)

	// Start the server on localhost port 8080
	log.Println("Starting server on :8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("Error starting server: ", err)
	}
}
