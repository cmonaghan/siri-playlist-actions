package handler

import (
	"fmt"
	"net/http"
	"siri-playlist-actions/utils"
)

// Handler for /api/add-song
func AddSongHandler(w http.ResponseWriter, r *http.Request) {
	apiKey := r.Header.Get("X-API-Key")
	playlistID := r.Header.Get("X-Playlist-ID")

	if apiKey == "" || playlistID == "" {
		http.Error(w, "Missing API Key or Playlist ID", http.StatusBadRequest)
		return
	}

	tokenData, err := utils.GetTokenData(apiKey)
	if err != nil {
		http.Error(w, "Invalid API Key", http.StatusUnauthorized)
		return
	}

	songID, _, _, _, _, err := utils.GetCurrentlyPlayingSong(tokenData.AccessToken)
	if err != nil || songID == "" {
		http.Error(w, "No song is currently playing", http.StatusNotFound)
		return
	}

	err = utils.AddSongToPlaylist(tokenData.AccessToken, playlistID, songID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error adding song: %s", err), http.StatusInternalServerError)
		return
	}

	w.Write([]byte("Song added successfully!"))
}
