package handler

import (
	"fmt"
	"log"
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
	userAuthData, err := utils.GetAPIKeyToUserAuthData(apiKey, redisPool.Get(), utils.RefreshSpotifyToken, utils.SetAPIKeyToUserAuthData)
	if err != nil {
		http.Error(w, "Invalid API Key", http.StatusUnauthorized)
		return
	}

	// Get currently playing song
	songID, _, _, playlistID, playlistName, err := utils.GetCurrentlyPlayingSong(userAuthData.AccessToken)
	if err != nil {
		log.Print(err)
		http.Error(w, "Error retrieving currently playing song", http.StatusInternalServerError)
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
	isOwner, err := utils.IsPlaylistOwnedByUser(userAuthData.AccessToken, playlistID)
	if err != nil {
		log.Print(err)
		http.Error(w, "Error checking playlist ownership. Do you own this playlist?", http.StatusInternalServerError)
		return
	}
	if !isOwner {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("The current playlist is not owned by you, so we cannot remove this song"))
		return
	}

	// Remove the song from the playlist
	err = utils.RemoveSongFromPlaylist(userAuthData.AccessToken, playlistID, songID)
	if err != nil {
		log.Print(err)
		http.Error(w, "Error removing song from playlist", http.StatusInternalServerError)
		return
	}
	err = utils.SkipSong(userAuthData.AccessToken)
	if err != nil {
		log.Printf("Failed to skip song with error: %s", err)
		// continue
	}

	// Success response
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("Song removed from %s", playlistName)))
}
