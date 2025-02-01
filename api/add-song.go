package handler

import (
	"fmt"
	"net/http"
	"siri-playlist-actions/utils"
)

// Handler for /api/add-song
func AddSongHandler(w http.ResponseWriter, r *http.Request) {
	apiKey := r.Header.Get("X-API-Key")
	destinationPlaylistID := r.Header.Get("X-Playlist-ID")

	if apiKey == "" {
		http.Error(w, "Missing API Key header", http.StatusBadRequest)
		return
	}
	if destinationPlaylistID == "" {
		http.Error(w, "Missing Playlist ID header", http.StatusBadRequest)
		return
	}

	// connect to database
	redisPool, err := utils.InitRedis()
	if err != nil {
		http.Error(w, "Error connecting to database", http.StatusInternalServerError)
		return
	}
	defer redisPool.Close()

	tokenData, err := utils.GetTokenData(apiKey)
	if err != nil {
		http.Error(w, "Invalid API Key", http.StatusUnauthorized)
		return
	}

	songID, songName, _, _, _, err := utils.GetCurrentlyPlayingSong(tokenData.AccessToken)
	if err != nil || songID == "" {
		http.Error(w, "No song is currently playing", http.StatusNotFound)
		return
	}

	// Get the playlist name
	destinationPlaylistName, err := utils.GetPlaylistName(tokenData.AccessToken, destinationPlaylistID)
	if err != nil {
		// this is cosmetic, so let the request continue even if this cannot be found
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

	err = utils.AddSongToPlaylist(tokenData.AccessToken, destinationPlaylistID, songID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error adding song: %s", err), http.StatusInternalServerError)
		return
	}

	w.Write([]byte(fmt.Sprintf("The song '%s' was added to your playlist '%s", songName, destinationPlaylistID)))
}
