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

	userAuthData, err := utils.GetAPIKeyToUserAuthData(apiKey, redisPool.Get(), utils.RefreshSpotifyToken, utils.SetAPIKeyToUserAuthData)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid API Key: %s", err), http.StatusUnauthorized)
		return
	}

	// Fetch currently playing song
	_, songName, artistName, playlistID, playlistName, err := utils.GetCurrentlyPlayingSong(userAuthData.AccessToken)
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
			<p>Below are a few shortcuts that allow you to control Spotify via voice command. You can pick and choose which commands you wish to add as they all function independently.</p>
			
			<h3>Shortcut 1: Add song to a designated playlist</h3>
			<p>This shortcut allows you to add a song to a pre-designated playlist via a Siri voice command. For example, imagine you have a playlist called "Hot Stuff." Setting up this shortcut would allow you to add songs to this playlist via a Siri voice command, such as when you have your hands full while driving, or cooking, or while juggling way too many flaming bowling pins.</p>
			<ol>
				<li>Play a song from the playlist you wish to add songs to, then refresh this page</li>
				<li>Open the Shortcuts app on your iPhone or macbook (setting the shortcut up on one will mirror to the other). These instructions assume iPhone.</li>
				<li>Tap "+" in the upper right</li>
				<li>Search for "Get Contents of URL"</li>
				<li>Set the URL to <code>https://spotify.woolgathering.io/api/add-song</code></li>
				<li>Set "Method" to "POST"</li>
				<li>Set "Headers" to Key: <code>X-API-Key</code> and Text: <code>{{.APIKey}}</code></li>
				<li>Set "Request Body" to "JSON"</li>
				<li>Key: <code>playlist_id</code>, Type: <code>Text</code>, Text: <code>{{.PlaylistID}}</code></li>
				<li>Set the title of the shortcut to "Add song to playlist" or whatever Siri command you want to say to trigger the shortcut</li>
				<li>All done! Try it out by speaking your voice command to Siri.</li>
			</ol>
			
			<!-- Example Image -->
			<img class="example-img" src="/static/add-song.png" alt="Add Song example">

			<h3>Shortcut 2: Remove song from the current playlist</h3>
			<p>This shortcut allows you to remove a song from the currently playing playlist via a Siri voice command. This only works on playlists that you created.</p>
			<ol>
				<li>Open the Shortcuts app on your iPhone or macbook (setting the shortcut up on one will mirror to the other). These instructions assume iPhone.</li>
				<li>Tap "+" in the upper right.</li>
				<li>Search for "Get Contents of URL".</li>
				<li>Set the URL to <code>https://spotify.woolgathering.io/api/remove-song</code>.</li>
				<li>Set "Method" to "POST".</li>
				<li>Set "Headers" to Key: <code>X-API-Key</code> and Text: <code>{{.APIKey}}</code></li>
				<li>Set the title of the shortcut to "Remove song from playlist" or whatever Siri command you want to say to trigger the shortcut.</li>
				<li>All done! Try it out by speaking your voice command to Siri.</li>
			</ol>
			
			<!-- Example Image -->
			<img class="example-img" src="/static/remove-song.png" alt="Remove Song example">

			<button class="revoke-button" onclick="confirmRevoke()">Revoke Access</button>
			<p>If you wish to disconnect your Spotify account and delete your user from the system, click the Revoke Access button.</p>

			<script>
				function copyApiKey() {
					const token = document.getElementById("apiKey").innerText;
					navigator.clipboard.writeText(token);
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
						window.location.href = "/";
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
