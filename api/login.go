package handler

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
)

// Handler for /api/login
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	spotifyClientID := os.Getenv("SPOTIFY_CLIENT_ID")
	redirectURI := os.Getenv("REDIRECT_URI")
	log.Printf("REDIRECT_URI=%s", redirectURI)

	authURL := fmt.Sprintf(
		"https://accounts.spotify.com/authorize?client_id=%s&response_type=code&redirect_uri=%s&scope=%s",
		spotifyClientID,
		url.QueryEscape(redirectURI),
		url.QueryEscape("user-read-playback-state user-modify-playback-state playlist-modify-public playlist-modify-private"),
	)

	http.Redirect(w, r, authURL, http.StatusFound)
}
