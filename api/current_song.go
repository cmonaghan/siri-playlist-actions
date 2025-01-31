package handler

import (
	"encoding/json"
	"net/http"
	"siri-playlist-actions/utils"
)

// Handler for /api/current-song
func CurrentSongHandler(w http.ResponseWriter, r *http.Request) {
	apiKey := r.Header.Get("X-API-Key")
	if apiKey == "" {
		http.Error(w, "Missing API Key", http.StatusUnauthorized)
		return
	}

	tokenData, err := utils.GetTokenData(apiKey)
	if err != nil {
		http.Error(w, "Invalid API Key", http.StatusUnauthorized)
		return
	}

	_, songName, artistName, _, playlistName, err := utils.GetCurrentlyPlayingSong(tokenData.AccessToken)
	if err != nil {
		http.Error(w, "Error getting currently playing song", http.StatusInternalServerError)
		return
	}

	response := map[string]string{
		"current_song": songName,
		"artist_name":  artistName,
	}

	if playlistName != "" {
		response["playlist_name"] = playlistName
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
