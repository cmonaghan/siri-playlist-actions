package handler

import (
	"fmt"
	"net/http"
	"siri-playlist-actions/utils"
	"text/template"
)

// Handler for /api/setup
func SetupHandler(w http.ResponseWriter, r *http.Request) {
	apiKey := r.URL.Query().Get("api_key")
	if apiKey == "" {
		http.Error(w, "API key not found", http.StatusBadRequest)
		return
	}

	// connect to database
	redisPool, err := utils.InitRedis()
	if err != nil {
		http.Error(w, "Error connecting to database", http.StatusInternalServerError)
		return
	}
	defer redisPool.Close()

	tokenData, err := utils.GetAPIKeyToTokenData(apiKey)
	if err != nil {
		http.Error(w, "Invalid API Key", http.StatusUnauthorized)
		return
	}

	_, songName, artistName, playlistID, playlistName, err := utils.GetCurrentlyPlayingSong(tokenData.AccessToken)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting current song: %s", err), http.StatusInternalServerError)
		return
	}

	tmpl := `<html><body>
	<h1>Spotify Setup Complete!</h1>
	<p>Your API key: <strong>{{.APIKey}}</strong></p>
	<h2>Currently Playing</h2>
	<ul>
		<li><strong>Song:</strong> {{.CurrentSong}}</li>
		<li><strong>Artist:</strong> {{.ArtistName}}</li>
		<li><strong>Playlist Name:</strong> {{.PlaylistName}}</li>
		<li><strong>Playlist ID:</strong> {{.PlaylistID}}</li>
	</ul>
	</body></html>`

	t, _ := template.New("setup").Parse(tmpl)
	t.Execute(w, map[string]string{
		"APIKey":       apiKey,
		"CurrentSong":  songName,
		"ArtistName":   artistName,
		"PlaylistName": playlistName,
		"PlaylistID":   playlistID,
	})
}
