package handler

import (
	"fmt"
	"log"
	"net/http"
	"siri-playlist-actions/utils"
)

// Handler for /api/callback
func CallbackHandler(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Missing authorization code", http.StatusBadRequest)
		return
	}

	redisPool, err := utils.InitRedis()
	if err != nil {
		http.Error(w, "Error connecting to database", http.StatusInternalServerError)
		return
	}
	defer redisPool.Close()

	// Exchange code for token
	token, err := utils.ExchangeCodeForToken(code)
	if err != nil {
		log.Print(err)
		http.Error(w, "Error exchanging code for token", http.StatusInternalServerError)
		return
	}

	// Fetch user ID
	userID, err := utils.GetSpotifyUserID(token.AccessToken)
	if err != nil {
		log.Print(err)
		http.Error(w, "Error fetching Spotify user ID", http.StatusInternalServerError)
		return
	}

	// Check if user already has an API key
	apiKey, err := utils.GetUserIDToAPIKey(userID, redisPool.Get())
	if err != nil {
		http.Error(w, "Error checking API key", http.StatusInternalServerError)
		return
	}

	if apiKey == "" {
		// no API key exists, let's generate one
		apiKey = utils.GenerateAPIKey()

		// Store token in Redis
		err = utils.SetAPIKeyToUserAuthData(apiKey, token, userID, redisPool.Get())
		if err != nil {
			http.Error(w, "Failed to store API key", http.StatusInternalServerError)
			return
		}

		// Map user ID to API key
		err = utils.SetUserIDToAPIKey(userID, apiKey, redisPool.Get())
		if err != nil {
			http.Error(w, "Failed to map user ID", http.StatusInternalServerError)
			return
		}
	}

	http.Redirect(w, r, fmt.Sprintf("/setup?api_key=%s", apiKey), http.StatusFound)
}
