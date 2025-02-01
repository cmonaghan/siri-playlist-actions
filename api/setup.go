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

	// Connect to Redis
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

	// Define the HTML template inline
	tmpl := `
		<!DOCTYPE html>
		<html lang="en">
		<head>
			<meta charset="UTF-8">
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
			<title>Spotify Setup Complete</title>
			<style>
				body {
					font-family: Arial, sans-serif;
					margin: 20px;
					line-height: 1.6;
				}
				pre {
					background: #f4f4f4;
					padding: 10px;
					border: 1px solid #ddd;
					border-radius: 5px;
					font-size: 16px;
					overflow-x: auto;
				}
				button {
					color: white;
					border: none;
					padding: 10px 15px;
					font-size: 16px;
					border-radius: 5px;
					cursor: pointer;
					margin: 10px;
				}
				.copy-button {
					background-color: #007BFF;
				}
				.copy-button:hover {
					background-color: #0056b3;
				}
				.revoke-button {
					background-color: #dc3545;
				}
				.revoke-button:hover {
					background-color: #c82333;
				}
				img {
					margin-top: 20px;
					max-width: 100%;
				}
			</style>
		</head>
		<body>
			<h1>Spotify Setup Complete!</h1>
			<p>Your API key is:</p>
			<pre id="apiKey">{{.APIKey}}</pre>
			<button class="copy-button" onclick="copyApiKey()">Copy API Key</button>

			<h2>Currently Playing</h2>
			<ul>
				<li><strong>Song:</strong> {{.CurrentSong}}</li>
				<li><strong>Artist:</strong> {{.ArtistName}}</li>
				<li><strong>Playlist:</strong> {{.PlaylistName}}</li>
				<li><strong>Playlist ID:</strong> {{.PlaylistID}}</li>
			</ul>

			<button class="revoke-button" onclick="revokeAccess()">Revoke Access</button>

			<script>
				function copyApiKey() {
					const token = document.getElementById("apiKey").innerText;
					navigator.clipboard.writeText(token);
					alert("API Key copied to clipboard!");
				}

				function revokeAccess() {
					fetch('/api/revoke', {
						method: 'POST',
						headers: {
							'X-API-Key': '{{.APIKey}}'
						}
					}).then(response => response.text())
					.then(data => {
						alert(data);
						window.location.href = "/api/login";
					})
					.catch(error => alert("Error revoking access: " + error));
				}
			</script>
		</body>
		</html>
	`

	// Parse and execute the template
	t, err := template.New("setup").Parse(tmpl)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error parsing template: %s", err), http.StatusInternalServerError)
		return
	}

	// Render the template with the API key and song details
	err = t.Execute(w, map[string]string{
		"APIKey":       apiKey,
		"CurrentSong":  songName,
		"ArtistName":   artistName,
		"PlaylistName": playlistName,
		"PlaylistID":   playlistID,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("Error rendering template: %s", err), http.StatusInternalServerError)
	}
}
