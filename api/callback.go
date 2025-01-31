package handler

import (
	"fmt"
	"net/http"
	"os"
	"siri-playlist-actions/utils"

	"github.com/gomodule/redigo/redis"
)

var redisPool *redis.Pool

// Handler for /api/callback
func CallbackHandler(w http.ResponseWriter, r *http.Request) {
	redisURL := os.Getenv("KV_URL")
	utils.InitRedis(redisURL)
	defer redisPool.Close()

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Missing authorization code", http.StatusBadRequest)
		return
	}

	// Exchange code for token
	token, err := utils.ExchangeCodeForToken(code)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error exchanging code for token: %v", err), http.StatusInternalServerError)
		return
	}

	// Fetch user ID
	userID, err := utils.GetSpotifyUserID(token.AccessToken)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching Spotify user ID: %v", err), http.StatusInternalServerError)
		return
	}

	// Check if user already has an API key
	apiKey, err := utils.GetAPIKeyByUserID(userID)
	if err != nil {
		http.Error(w, "Error checking API key", http.StatusInternalServerError)
		return
	}

	if apiKey == "" {
		apiKey = utils.GenerateAPIKey()
	}

	// Store token in Redis
	err = utils.StoreAPIKey(apiKey, token)
	if err != nil {
		http.Error(w, "Failed to store API key", http.StatusInternalServerError)
		return
	}

	// Map user ID to API key
	err = utils.MapUserIDToAPIKey(userID, apiKey)
	if err != nil {
		http.Error(w, "Failed to map user ID", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/api/setup?api_key=%s", apiKey), http.StatusFound)
}
