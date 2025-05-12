package handler

import (
	"encoding/json"
	"log"
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

	// connect to database
	redisPool, err := utils.InitRedis()
	if err != nil {
		http.Error(w, "Error connecting to database", http.StatusInternalServerError)
		return
	}
	defer redisPool.Close()

	userAuthData, err := utils.GetAPIKeyToUserAuthData(apiKey, redisPool.Get(), utils.RefreshSpotifyToken, utils.SetAPIKeyToUserAuthData)
	if err != nil {
		http.Error(w, "Invalid API Key", http.StatusUnauthorized)
		return
	}

	songID, songName, artistName, _, playlistName, err := utils.GetCurrentlyPlayingSong(userAuthData.AccessToken)
	if songID == "" {
		http.Error(w, "No song is currently playing", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Print(err)
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
