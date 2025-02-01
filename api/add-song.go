package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"siri-playlist-actions/utils"
)

// RequestBody defines the expected JSON payload
type RequestBody struct {
	PlaylistID string `json:"playlist_id"`
}

// Handler for /api/add-song
func AddSongHandler(w http.ResponseWriter, r *http.Request) {
	apiKey := r.Header.Get("X-API-Key")
	if apiKey == "" {
		http.Error(w, "Missing API Key header", http.StatusBadRequest)
		return
	}

	// Parse JSON request body
	var requestBody RequestBody
	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil || requestBody.PlaylistID == "" {
		http.Error(w, "Invalid JSON body: Missing 'playlist_id'", http.StatusBadRequest)
		return
	}
	destinationPlaylistID := requestBody.PlaylistID

	// Connect to database
	redisPool, err := utils.InitRedis()
	if err != nil {
		http.Error(w, "Error connecting to database", http.StatusInternalServerError)
		return
	}
	defer redisPool.Close()

	tokenData, err := utils.GetAPIKeyToTokenData(apiKey)
	if err != nil {
		http.Error(w, "Invalid API Key", http.StatusUnauthorized)
		return
	}

	songID, songName, _, _, _, err := utils.GetCurrentlyPlayingSong(tokenData.AccessToken)
	if err != nil || songID == "" {
		http.Error(w, "No song is currently playing", http.StatusNotFound)
		return
	}

	// Get the playlist name (optional)
	destinationPlaylistName, err := utils.GetPlaylistName(tokenData.AccessToken, destinationPlaylistID)
	if err != nil {
		// If we can't retrieve the name, default to "unknown"
		destinationPlaylistName = "unknown"
	}

	// Check if the song is already in the playlist
	isInPlaylist, err := utils.IsSongInPlaylist(tokenData.AccessToken, destinationPlaylistID, songID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error checking playlist: %s", err), http.StatusInternalServerError)
		return
	}

	if isInPlaylist {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("This song is already in your playlist '%s', so we skipped adding a duplicate.", destinationPlaylistName)))
		return
	}

	// Add song to playlist
	err = utils.AddSongToPlaylist(tokenData.AccessToken, destinationPlaylistID, songID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error adding song: %s", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("The song '%s' was added to your playlist '%s'", songName, destinationPlaylistName)))
}
