package handler

import (
	"fmt"
	"net/http"
	"siri-playlist-actions/utils"
)

// RemoveSongHandler removes the currently playing song from the playlist
func RemoveSongHandler(w http.ResponseWriter, r *http.Request) {
	// Get the API Key from request header
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

	// Retrieve token data from Redis
	tokenData, err := utils.GetAPIKeyToTokenData(apiKey)
	if err != nil {
		http.Error(w, "Invalid API Key", http.StatusUnauthorized)
		return
	}

	// Get currently playing song
	songID, _, _, playlistID, playlistName, err := utils.GetCurrentlyPlayingSong(tokenData.AccessToken)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error retrieving currently playing song: %s", err), http.StatusInternalServerError)
		return
	}

	if songID == "" {
		http.Error(w, "No song is currently playing", http.StatusNotFound)
		return
	}

	if playlistID == "" {
		http.Error(w, "The song is not playing from a playlist, so it cannot be removed", http.StatusNotFound)
		return
	}

	// Check if the user owns the playlist
	isOwner, err := utils.IsPlaylistOwnedByUser(tokenData.AccessToken, playlistID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error checking playlist ownership: %s", err), http.StatusInternalServerError)
		return
	}
	if !isOwner {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("The current playlist is not owned by you, so we cannot remove this song"))
		return
	}

	// Remove the song from the playlist
	err = utils.RemoveSongFromPlaylist(tokenData.AccessToken, playlistID, songID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error removing song from playlist: %s", err), http.StatusInternalServerError)
		return
	}

	// Success response
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("This song was removed from your playlist %s", playlistName)))
}
