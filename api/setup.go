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

	// Fetch currently playing song
	_, songName, artistName, playlistID, playlistName, err := utils.GetCurrentlyPlayingSong(tokenData.AccessToken)
	if err != nil {
		songName, artistName, playlistName, playlistID = "Not Available", "Not Available", "Not Available", "Not Available"
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
					margin: 10px 0;
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
				.example-img {
					margin-top: 20px;
					width: 100%;
					max-width: 1000px;
					border-radius: 8px;
					display: block;
				}
			</style>
		</head>
		<body>
			<h1>Spotify Setup Complete!</h1>

			<h2>Currently Playing</h2>
			<ul>
				<li><strong>Song:</strong> {{.CurrentSong}}</li>
				<li><strong>Artist:</strong> {{.ArtistName}}</li>
				<li><strong>Playlist:</strong> {{.PlaylistName}}</li>
				<li><strong>Playlist ID:</strong> {{.PlaylistID}}</li>
			</ul>

			<h2>Setup Steps</h2>
			<p>Your API key is:</p>
			<pre id="apiKey">{{.APIKey}}</pre>
			<button class="copy-button" onclick="copyApiKey()">Copy API Key</button>
			<p>Now, to use this API key in Siri Shortcuts:</p>
			<ol>
				<li>Open the Shortcuts app on your iPhone or macbook (setting the shortcut up on one will mirror to the other). These instructions assume iPhone.</li>
				<li>Tap "+" in the upper right.</li>
				<li>Search for "Get Contents of URL".</li>
				<li>Set the URL to <code>https://spotify.woolgathering.io/api/current-song</code>.</li>
				<li>Set "Method" to "POST".</li>
				<li>Set "Headers" to Key: <code>X-API-Key</code> and Text: <code>{{.APIKey}}</code></li>
				<li>Set the title of the shortcut to "Shortcut current song" or whatever Siri command you want to say to trigger the shortcut.</li>
				<li>You can now use this Shortcut to check the current song!</li>
			</ol>
			<p>You can repeat the same steps to setup shortcuts for:</p>
			<ul>
				<li><code>https://spotify.woolgathering.io/api/add-song</code></li>
				<li><code>https://spotify.woolgathering.io/api/remove-song</code></li>
			</ul>
			<p>(Note that <code>/api/add-song</code> will also need the header <code>X-Playlist-ID</code> to be set. You can find the correct Playlist ID value by playing a song from your desired target playlist and then refreshing this page to see the Playlist ID to be displayed above.)</p>

			<!-- Example Image -->
			<img class="example-img" src="/static/example.png" alt="Example Usage">

			<button class="revoke-button" onclick="confirmRevoke()">Revoke Access</button>

			<script>
				function copyApiKey() {
					const token = document.getElementById("apiKey").innerText;
					navigator.clipboard.writeText(token);
					alert("API Key copied to clipboard!");
				}

				function confirmRevoke() {
					if (confirm("Are you sure you want to revoke access? This action cannot be undone.")) {
						revokeAccess();
					}
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
