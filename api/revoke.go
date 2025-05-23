package handler

import (
	"fmt"
	"net/http"
	"siri-playlist-actions/utils"
)

// RevokeHandler removes the user's session from Redis
func RevokeHandler(w http.ResponseWriter, r *http.Request) {
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

	setFn := func(apiKey string, token *utils.SpotifyAccessToken, userID string) error {
		return utils.SetAPIKeyToUserAuthData(apiKey, token, userID, redisPool.Get())
	}
	userAuthData, err := utils.GetAPIKeyToUserAuthData(apiKey, redisPool.Get(), utils.RefreshSpotifyToken, setFn)
	if err != nil {
		http.Error(w, "Invalid API Key", http.StatusUnauthorized)
		return
	}

	// Delete the API key from Redis
	err = utils.DeleteAPIKey(apiKey, redisPool.Get())
	if err != nil {
		http.Error(w, fmt.Sprintf("Error revoking session: %s", err), http.StatusInternalServerError)
		return
	}

	// Remove the user-to-API key mapping
	err = utils.DeleteUserID(userAuthData.UserID, redisPool.Get())
	if err != nil {
		http.Error(w, fmt.Sprintf("Error removing user session: %s", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Your session has been revoked successfully."))
}
