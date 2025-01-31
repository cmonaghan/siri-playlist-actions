// package main

// import (
// 	"encoding/json"
// 	"fmt"
// 	"io/ioutil"
// 	"log"
// 	"math/rand"
// 	"net/http"
// 	"net/url"
// 	"os"
// 	"strings"
// 	"text/template"
// 	"time"

// 	"github.com/gomodule/redigo/redis"
// 	"github.com/joho/godotenv"
// )

// type TokenData struct {
// 	AccessToken  string    `json:"access_token"`
// 	RefreshToken string    `json:"refresh_token"`
// 	ExpiresAt    time.Time `json:"expires_at"`
// }

// type SpotifyAccessToken struct {
// 	AccessToken  string `json:"access_token"`
// 	RefreshToken string `json:"refresh_token"`
// }

// var (
// 	spotifyClientID     string
// 	spotifyClientSecret string
// 	redirectURI         string
// 	redisPool           *redis.Pool
// )

// const spotifyAPIBaseURL = "https://api.spotify.com/v1"
// const spotifyAuthURL = "https://accounts.spotify.com/authorize"
// const spotifyTokenURL = "https://accounts.spotify.com/api/token"

// func main() {
// 	// Load environment variables from the .env file
// 	err := godotenv.Load()
// 	if err != nil {
// 		log.Fatal("Error loading .env file")
// 	}

// 	// Retrieve environment variables after loading them
// 	spotifyClientID = os.Getenv("SPOTIFY_CLIENT_ID")
// 	spotifyClientSecret = os.Getenv("SPOTIFY_CLIENT_SECRET")
// 	redirectURI = os.Getenv("REDIRECT_URI") // This should match the one set in Spotify Developer Dashboard
// 	redisURL := os.Getenv("KV_URL")

// 	initRedis(redisURL)
// 	defer redisPool.Close()

// 	// Check if Spotify credentials and API token are set in the environment variables
// 	if spotifyClientID == "" || spotifyClientSecret == "" || redirectURI == "" || redisURL == "" {
// 		log.Fatal("Missing required environment variables.")
// 		return
// 	}

// 	// endpoints for initial setup
// 	http.HandleFunc("/login", loginHandler)
// 	http.HandleFunc("/callback", callbackHandler)
// 	http.HandleFunc("/setup", completeSetupHandler)
// 	// endpoints for regular usage
// 	http.HandleFunc("/current-song", currentSongHandler)
// 	http.HandleFunc("/add-song-to-playlist", addSongToPlaylistHandler)
// 	http.HandleFunc("/remove-current-song", removeCurrentSongHandler)

// 	// Serve static files (e.g., images, stylesheets) from the "static" directory
// 	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

// 	// Start the server on localhost port 8080
// 	log.Println("Starting server on :8080...")
// 	if err := http.ListenAndServe(":8080", nil); err != nil {
// 		log.Fatal("Error starting server: ", err)
// 	}
// }

// func loginHandler(w http.ResponseWriter, r *http.Request) {
// 	// Generate the Spotify authorization URL
// 	authURL := fmt.Sprintf(
// 		"%s?client_id=%s&response_type=code&redirect_uri=%s&scope=%s",
// 		spotifyAuthURL,
// 		spotifyClientID,
// 		url.QueryEscape(redirectURI),
// 		url.QueryEscape("user-read-playback-state user-modify-playback-state playlist-modify-public playlist-modify-private"),
// 	)

// 	// Redirect the user to the Spotify authorization page
// 	http.Redirect(w, r, authURL, http.StatusFound)
// }

// func callbackHandler(w http.ResponseWriter, r *http.Request) {
// 	// Get the authorization code from the query string
// 	code := r.URL.Query().Get("code")
// 	if code == "" {
// 		http.Error(w, "Missing authorization code", http.StatusBadRequest)
// 		return
// 	}

// 	// Exchange the authorization code for an access token
// 	token, err := exchangeCodeForToken(code)
// 	if err != nil {
// 		http.Error(w, fmt.Sprintf("Error exchanging code for token: %v", err), http.StatusInternalServerError)
// 		return
// 	}

// 	// Fetch the Spotify user ID to identify the user
// 	userID, err := getSpotifyUserID(token.AccessToken)
// 	if err != nil {
// 		http.Error(w, fmt.Sprintf("Error fetching Spotify user ID: %v", err), http.StatusInternalServerError)
// 		return
// 	}

// 	// Check if an API key already exists for this user
// 	apiKey, err := getAPIKeyByUserID(userID)
// 	if err != nil {
// 		http.Error(w, fmt.Sprintf("Error checking existing API key: %v", err), http.StatusInternalServerError)
// 		return
// 	}

// 	if apiKey == "" {
// 		// Generate a new API key if one doesn't exist
// 		apiKey = generateAPIKey()
// 	}

// 	// Update the token data in Redis
// 	err = storeAPIKey(apiKey, token)
// 	if err != nil {
// 		http.Error(w, fmt.Sprintf("Failed to store updated API key: %v", err), http.StatusInternalServerError)
// 		return
// 	}

// 	// Map the user ID to the API key
// 	err = mapUserIDToAPIKey(userID, apiKey)
// 	if err != nil {
// 		http.Error(w, fmt.Sprintf("Failed to map user ID to API key: %v", err), http.StatusInternalServerError)
// 		return
// 	}

// 	// Redirect the user to the /setup endpoint with the API key
// 	http.Redirect(w, r, fmt.Sprintf("/setup?api_key=%s", apiKey), http.StatusFound)
// }

// func getAPIKeyByUserID(userID string) (string, error) {
// 	conn := redisPool.Get()
// 	defer conn.Close()

// 	apiKey, err := redis.String(conn.Do("GET", fmt.Sprintf("user:%s", userID)))
// 	if err == redis.ErrNil {
// 		// No API key found for this user
// 		return "", nil
// 	}
// 	if err != nil {
// 		return "", fmt.Errorf("failed to retrieve API key by user ID: %v", err)
// 	}

// 	return apiKey, nil
// }

// func getSpotifyUserID(accessToken string) (string, error) {
// 	url := fmt.Sprintf("%s/me", spotifyAPIBaseURL)

// 	req, err := http.NewRequest("GET", url, nil)
// 	if err != nil {
// 		return "", err
// 	}
// 	req.Header.Add("Authorization", "Bearer "+accessToken)

// 	client := &http.Client{}
// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return "", err
// 	}
// 	defer resp.Body.Close()

// 	if resp.StatusCode != 200 {
// 		body, _ := ioutil.ReadAll(resp.Body)
// 		return "", fmt.Errorf("failed to fetch user ID: %s", body)
// 	}

// 	var data struct {
// 		ID string `json:"id"` // Spotify user ID
// 	}
// 	err = json.NewDecoder(resp.Body).Decode(&data)
// 	if err != nil {
// 		return "", err
// 	}

// 	return data.ID, nil
// }

// func mapUserIDToAPIKey(userID, apiKey string) error {
// 	conn := redisPool.Get()
// 	defer conn.Close()

// 	_, err := conn.Do("SET", fmt.Sprintf("user:%s", userID), apiKey)
// 	if err != nil {
// 		return fmt.Errorf("failed to map user ID to API key: %v", err)
// 	}

// 	return nil
// }

// func exchangeCodeForToken(code string) (*SpotifyAccessToken, error) {
// 	// Prepare the request data
// 	data := url.Values{}
// 	data.Set("grant_type", "authorization_code")
// 	data.Set("code", code)
// 	data.Set("redirect_uri", redirectURI)

// 	// Send the request to exchange the code for an access token
// 	req, err := http.NewRequest("POST", spotifyTokenURL, strings.NewReader(data.Encode()))
// 	if err != nil {
// 		return nil, err
// 	}
// 	req.SetBasicAuth(spotifyClientID, spotifyClientSecret)
// 	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

// 	client := &http.Client{}
// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer resp.Body.Close()

// 	// Parse the response to get the access token and refresh token
// 	var token SpotifyAccessToken
// 	body, err := ioutil.ReadAll(resp.Body)
// 	if err != nil {
// 		return nil, err
// 	}

// 	if resp.StatusCode != 200 {
// 		return nil, fmt.Errorf("failed to get access token: %s", body)
// 	}

// 	err = json.Unmarshal(body, &token)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return &token, nil
// }

// func completeSetupHandler(w http.ResponseWriter, r *http.Request) {
// 	apiKey := r.URL.Query().Get("api_key")
// 	if apiKey == "" {
// 		http.Error(w, "API key not found", http.StatusBadRequest)
// 		return
// 	}

// 	tokenData, err := getTokenData(apiKey)
// 	if err != nil {
// 		http.Error(w, "Invalid API Key", http.StatusUnauthorized)
// 		return
// 	}

// 	// Fetch the currently playing song details
// 	_, songName, artistName, playlistID, _, err := getCurrentlyPlayingSong(tokenData.AccessToken)
// 	if err != nil {
// 		http.Error(w, fmt.Sprintf("Error getting current song: %s", err), http.StatusInternalServerError)
// 		return
// 	}

// 	// Define the template for the setup page
// 	tmpl := `
// 		<!DOCTYPE html>
// 		<html lang="en">
// 		<head>
// 			<meta charset="UTF-8">
// 			<meta name="viewport" content="width=device-width, initial-scale=1.0">
// 			<title>Complete Setup</title>
// 			<style>
// 				body {
// 					font-family: Arial, sans-serif;
// 					margin: 20px;
// 					line-height: 1.6;
// 				}
// 				pre {
// 					background: #f4f4f4;
// 					padding: 10px;
// 					border: 1px solid #ddd;
// 					border-radius: 5px;
// 					font-size: 16px;
// 					overflow-x: auto;
// 				}
// 				button {
// 					background-color: #007BFF;
// 					color: white;
// 					border: none;
// 					padding: 10px 15px;
// 					font-size: 16px;
// 					border-radius: 5px;
// 					cursor: pointer;
// 				}
// 				button:hover {
// 					background-color: #0056b3;
// 				}
// 				img {
// 					margin-top: 20px;
// 					max-width: 100%;
// 				}
// 			</style>
// 		</head>
// 		<body>
// 			<h1>Spotify Setup Complete!</h1>
// 			<p>Your API key is:</p>
// 			<pre id="apiKey">{{.APIKey}}</pre>
// 			<button onclick="copyApiKey()">Copy API Key</button>
// 			<h2>Currently Playing</h2>
// 			<ul>
// 				<li><strong>Song:</strong> {{.CurrentSong}}</li>
// 				<li><strong>Artist:</strong> {{.ArtistName}}</li>
// 				<li><strong>Playlist ID:</strong> {{.PlaylistID}}</li>
// 			</ul>
// 			<p>Now, to use this API key in Siri Shortcuts:</p>
// 			<ol>
// 				<li>Open the Shortcuts app on your iPhone.</li>
// 				<li>Tap "+" in the upper right.</li>
// 				<li>Search for "Get Contents of URL".</li>
// 				<li>Set the URL to <code>http://localhost:8080/current-song</code>.</li>
// 				<li>Set "Method" to "POST".</li>
// 				<li>Set "Headers" to Key: <code>X-API-Key</code> and Text: <code>{{.APIKey}}</code></li>
// 				<li>You can now use this Shortcut to check the current song!</li>
// 			</ol>
// 			<img src="/static/example-shortcut.jpeg" alt="Apple Shortcut Example Setup" />
// 			<script>
// 				function copyApiKey() {
// 					// Get the API key text
// 					const token = document.getElementById("apiKey").innerText;

// 					// Copy the API key to clipboard
// 					navigator.clipboard.writeText(token);
// 				}
// 			</script>
// 		</body>
// 		</html>
// 	`

// 	// Create a template and execute it
// 	t, err := template.New("setup").Parse(tmpl)
// 	if err != nil {
// 		http.Error(w, fmt.Sprintf("Error parsing template: %s", err), http.StatusInternalServerError)
// 		return
// 	}

// 	// Render the page with the API key and song details
// 	err = t.Execute(w, map[string]interface{}{
// 		"APIKey":      apiKey,
// 		"CurrentSong": songName,
// 		"ArtistName":  artistName,
// 		"PlaylistID":  playlistID,
// 	})
// 	if err != nil {
// 		http.Error(w, fmt.Sprintf("Error rendering template: %s", err), http.StatusInternalServerError)
// 	}
// }

// func currentSongHandler(w http.ResponseWriter, r *http.Request) {
// 	apiKey := r.Header.Get("X-API-Key")
// 	if apiKey == "" {
// 		http.Error(w, "Missing API Key", http.StatusUnauthorized)
// 		return
// 	}

// 	tokenData, err := getTokenData(apiKey)
// 	if err != nil {
// 		http.Error(w, "Invalid API Key", http.StatusUnauthorized)
// 		return
// 	}

// 	songID, songName, artistName, _, playlistName, err := getCurrentlyPlayingSong(tokenData.AccessToken)
// 	if err != nil {
// 		http.Error(w, fmt.Sprintf("Error getting currently playing song: %s", err), http.StatusInternalServerError)
// 		return
// 	}

// 	if songID == "" {
// 		http.Error(w, "No song is currently playing", http.StatusNotFound)
// 		return
// 	}

// 	// Build the response
// 	response := map[string]string{
// 		"current_song": songName,
// 		"artist_name":  artistName,
// 	}

// 	if playlistName != "" {
// 		response["playlist_name"] = playlistName
// 	}

// 	w.WriteHeader(http.StatusOK)
// 	w.Header().Set("Content-Type", "application/json")
// 	json.NewEncoder(w).Encode(response)
// }

// func getCurrentlyPlayingSong(accessToken string) (string, string, string, string, string, error) {
// 	// Make the API request to get the currently playing track
// 	req, err := http.NewRequest("GET", spotifyAPIBaseURL+"/me/player/currently-playing", nil)
// 	if err != nil {
// 		return "", "", "", "", "", err
// 	}

// 	req.Header.Add("Authorization", "Bearer "+accessToken)

// 	client := &http.Client{}
// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return "", "", "", "", "", err
// 	}
// 	defer resp.Body.Close()

// 	body, err := ioutil.ReadAll(resp.Body)
// 	if err != nil {
// 		return "", "", "", "", "", err
// 	}

// 	// Parse JSON response
// 	var data map[string]interface{}
// 	err = json.Unmarshal(body, &data)
// 	if err != nil {
// 		return "", "", "", "", "", err
// 	}

// 	var songID, songName, artistName, playlistID, playlistName string

// 	// Extract song details
// 	if item, exists := data["item"].(map[string]interface{}); exists {
// 		// Get the song ID and name
// 		if id, exists := item["id"].(string); exists {
// 			songID = id
// 		}
// 		if name, exists := item["name"].(string); exists {
// 			songName = name
// 		}

// 		// Extract artist name
// 		if artists, exists := item["artists"].([]interface{}); exists && len(artists) > 0 {
// 			if firstArtist, ok := artists[0].(map[string]interface{}); ok {
// 				if artistNameValue, exists := firstArtist["name"].(string); exists {
// 					artistName = artistNameValue
// 				}
// 			}
// 		}
// 	}

// 	// Extract playlist ID from context
// 	if context, exists := data["context"].(map[string]interface{}); exists {
// 		if uri, exists := context["uri"].(string); exists {
// 			if strings.HasPrefix(uri, "spotify:playlist:") {
// 				playlistID = strings.TrimPrefix(uri, "spotify:playlist:")
// 			} else {
// 				// Log unexpected context URI (e.g., album or track)
// 				log.Printf("Current playback is not from a playlist, context URI: %s", uri)
// 			}
// 		}
// 	}

// 	// Get playlist name if a playlist ID exists
// 	if playlistID != "" {
// 		playlistName, err = getPlaylistName(accessToken, playlistID)
// 		if err != nil {
// 			return "", "", "", "", "", fmt.Errorf("failed to retrieve playlist name: %v", err)
// 		}
// 	}

// 	// Ensure we return the artist even if there's no playlist
// 	if songID == "" || songName == "" || artistName == "" {
// 		return "", "", "", "", "", fmt.Errorf("could not find the song ID, name, or artist")
// 	}

// 	return songID, songName, artistName, playlistID, playlistName, nil
// }

// func generateAPIKey() string {
// 	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
// 	const length = 32
// 	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
// 	apiKey := make([]byte, length)
// 	for i := range apiKey {
// 		apiKey[i] = charset[seededRand.Intn(len(charset))]
// 	}
// 	return string(apiKey)
// }

// func storeAPIKey(apiKey string, token *SpotifyAccessToken) error {
// 	conn := redisPool.Get()
// 	defer conn.Close()

// 	tokenData := TokenData{
// 		AccessToken:  token.AccessToken,
// 		RefreshToken: token.RefreshToken,
// 		ExpiresAt:    time.Now().Add(time.Hour), // Spotify tokens typically expire after 1 hour
// 	}

// 	data, err := json.Marshal(tokenData)
// 	if err != nil {
// 		return fmt.Errorf("failed to marshal token data: %v", err)
// 	}

// 	_, err = conn.Do("SET", apiKey, data, "EX", 3600*24*30) // Set expiration to 30 days
// 	if err != nil {
// 		return fmt.Errorf("failed to store API key in Redis: %v", err)
// 	}

// 	return nil
// }

// func refreshSpotifyToken(apiKey string) (*SpotifyAccessToken, error) {
// 	// Retrieve token data using the API key
// 	tokenData, err := getTokenData(apiKey)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to retrieve token data: %v", err)
// 	}

// 	// Refresh the access token using the refresh token
// 	data := url.Values{}
// 	data.Set("grant_type", "refresh_token")
// 	data.Set("refresh_token", tokenData.RefreshToken)

// 	req, err := http.NewRequest("POST", spotifyTokenURL, strings.NewReader(data.Encode()))
// 	if err != nil {
// 		return nil, err
// 	}
// 	req.SetBasicAuth(spotifyClientID, spotifyClientSecret)
// 	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

// 	client := &http.Client{}
// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer resp.Body.Close()

// 	var newToken SpotifyAccessToken
// 	body, err := ioutil.ReadAll(resp.Body)
// 	if err != nil {
// 		return nil, err
// 	}

// 	if resp.StatusCode != 200 {
// 		return nil, fmt.Errorf("failed to refresh token: %s", body)
// 	}

// 	err = json.Unmarshal(body, &newToken)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// Update the token data in Redis
// 	tokenData.AccessToken = newToken.AccessToken
// 	if newToken.RefreshToken != "" {
// 		tokenData.RefreshToken = newToken.RefreshToken
// 	}
// 	tokenData.ExpiresAt = time.Now().Add(time.Hour) // Tokens usually expire in 1 hour

// 	err = storeAPIKey(apiKey, &SpotifyAccessToken{
// 		AccessToken:  tokenData.AccessToken,
// 		RefreshToken: tokenData.RefreshToken,
// 	})
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to update token data in Redis: %v", err)
// 	}

// 	return &newToken, nil
// }

// func getTokenData(apiKey string) (*TokenData, error) {
// 	conn := redisPool.Get()
// 	defer conn.Close()

// 	data, err := redis.Bytes(conn.Do("GET", apiKey))
// 	if err == redis.ErrNil {
// 		return nil, fmt.Errorf("API key not found")
// 	}
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to retrieve API key from Redis: %v", err)
// 	}

// 	var tokenData TokenData
// 	err = json.Unmarshal(data, &tokenData)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to unmarshal token data: %v", err)
// 	}

// 	return &tokenData, nil
// }

// func initRedis(redisURL string) {
// 	if redisURL == "" {
// 		log.Fatal("KV_URL environment variable is not set")
// 	}

// 	redisPool = &redis.Pool{
// 		MaxIdle:   10,
// 		MaxActive: 100,
// 		Wait:      true,
// 		Dial: func() (redis.Conn, error) {
// 			c, err := redis.DialURL(redisURL)
// 			if err != nil {
// 				return nil, fmt.Errorf("failed to connect to Redis: %v", err)
// 			}
// 			return c, nil
// 		},
// 	}
// }

// func addSongToPlaylistHandler(w http.ResponseWriter, r *http.Request) {
// 	apiKey := r.Header.Get("X-API-Key")
// 	if apiKey == "" {
// 		http.Error(w, "Missing API Key", http.StatusUnauthorized)
// 		return
// 	}

// 	playlistID := r.Header.Get("X-Playlist-ID")
// 	if playlistID == "" {
// 		http.Error(w, "Missing Playlist ID", http.StatusBadRequest)
// 		return
// 	}

// 	tokenData, err := getTokenData(apiKey)
// 	if err != nil {
// 		http.Error(w, "Invalid API Key", http.StatusUnauthorized)
// 		return
// 	}

// 	// Get the currently playing song
// 	songID, _, _, _, _, err := getCurrentlyPlayingSong(tokenData.AccessToken)
// 	if err != nil {
// 		http.Error(w, fmt.Sprintf("Error getting currently playing song: %s", err), http.StatusInternalServerError)
// 		return
// 	}
// 	if songID == "" {
// 		http.Error(w, "No song is currently playing", http.StatusNotFound)
// 		return
// 	}

// 	// Get the playlist name
// 	playlistName, err := getPlaylistName(tokenData.AccessToken, playlistID)
// 	if err != nil {
// 		http.Error(w, fmt.Sprintf("Error retrieving playlist name: %s", err), http.StatusInternalServerError)
// 		return
// 	}

// 	// Check if the song is already in the playlist
// 	isInPlaylist, err := isSongInPlaylist(tokenData.AccessToken, playlistID, songID)
// 	if err != nil {
// 		http.Error(w, fmt.Sprintf("Error checking playlist: %s", err), http.StatusInternalServerError)
// 		return
// 	}

// 	if isInPlaylist {
// 		w.WriteHeader(http.StatusOK)
// 		w.Write([]byte(fmt.Sprintf("This song is already in your playlist %s, so we skipped adding a duplicate.", playlistName)))
// 		return
// 	}

// 	// Add the song to the playlist
// 	err = addSongToPlaylist(tokenData.AccessToken, playlistID, songID)
// 	if err != nil {
// 		http.Error(w, fmt.Sprintf("Error adding song to playlist: %s", err), http.StatusInternalServerError)
// 		return
// 	}

// 	w.WriteHeader(http.StatusOK)
// 	w.Write([]byte(fmt.Sprintf("This song was added to your playlist %s.", playlistName)))
// }

// func isSongInPlaylist(accessToken, playlistID, songID string) (bool, error) {
// 	url := fmt.Sprintf("%s/playlists/%s/tracks", spotifyAPIBaseURL, playlistID)

// 	req, err := http.NewRequest("GET", url, nil)
// 	if err != nil {
// 		return false, err
// 	}
// 	req.Header.Add("Authorization", "Bearer "+accessToken)

// 	client := &http.Client{}
// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return false, err
// 	}
// 	defer resp.Body.Close()

// 	if resp.StatusCode != 200 {
// 		body, _ := ioutil.ReadAll(resp.Body)
// 		return false, fmt.Errorf("failed to retrieve playlist tracks: %s", body)
// 	}

// 	var data struct {
// 		Items []struct {
// 			Track struct {
// 				ID string `json:"id"`
// 			} `json:"track"`
// 		} `json:"items"`
// 	}

// 	err = json.NewDecoder(resp.Body).Decode(&data)
// 	if err != nil {
// 		return false, err
// 	}

// 	for _, item := range data.Items {
// 		if item.Track.ID == songID {
// 			return true, nil
// 		}
// 	}

// 	return false, nil
// }

// func addSongToPlaylist(accessToken, playlistID, songID string) error {
// 	url := fmt.Sprintf("%s/playlists/%s/tracks", spotifyAPIBaseURL, playlistID)

// 	body := map[string]interface{}{
// 		"uris": []string{fmt.Sprintf("spotify:track:%s", songID)},
// 	}

// 	jsonBody, err := json.Marshal(body)
// 	if err != nil {
// 		return err
// 	}

// 	req, err := http.NewRequest("POST", url, strings.NewReader(string(jsonBody)))
// 	if err != nil {
// 		return err
// 	}
// 	req.Header.Add("Authorization", "Bearer "+accessToken)
// 	req.Header.Add("Content-Type", "application/json")

// 	client := &http.Client{}
// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return err
// 	}
// 	defer resp.Body.Close()

// 	if resp.StatusCode != 201 {
// 		body, _ := ioutil.ReadAll(resp.Body)
// 		return fmt.Errorf("failed to add song to playlist: %s", body)
// 	}

// 	return nil
// }

// func getPlaylistName(accessToken, playlistID string) (string, error) {
// 	url := fmt.Sprintf("%s/playlists/%s", spotifyAPIBaseURL, playlistID)

// 	req, err := http.NewRequest("GET", url, nil)
// 	if err != nil {
// 		return "", err
// 	}
// 	req.Header.Add("Authorization", "Bearer "+accessToken)

// 	client := &http.Client{}
// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return "", err
// 	}
// 	defer resp.Body.Close()

// 	if resp.StatusCode != 200 {
// 		body, _ := ioutil.ReadAll(resp.Body)
// 		return "", fmt.Errorf("failed to retrieve playlist name: %s", body)
// 	}

// 	var data struct {
// 		Name string `json:"name"`
// 	}

// 	err = json.NewDecoder(resp.Body).Decode(&data)
// 	if err != nil {
// 		return "", err
// 	}

// 	return data.Name, nil
// }

// func removeCurrentSongHandler(w http.ResponseWriter, r *http.Request) {
// 	apiKey := r.Header.Get("X-API-Key")
// 	if apiKey == "" {
// 		http.Error(w, "Missing API Key", http.StatusUnauthorized)
// 		return
// 	}

// 	tokenData, err := getTokenData(apiKey)
// 	if err != nil {
// 		http.Error(w, "Invalid API Key", http.StatusUnauthorized)
// 		return
// 	}

// 	// Updated: Correctly handle all 6 returned values
// 	songID, _, _, playlistID, playlistName, err := getCurrentlyPlayingSong(tokenData.AccessToken)
// 	if err != nil {
// 		http.Error(w, fmt.Sprintf("Error retrieving currently playing song: %s", err), http.StatusInternalServerError)
// 		return
// 	}

// 	if songID == "" {
// 		http.Error(w, "No song is currently playing", http.StatusNotFound)
// 		return
// 	}

// 	if playlistID == "" {
// 		http.Error(w, "The song is not playing from a playlist so it cannot be removed", http.StatusNotFound)
// 		return
// 	}

// 	// Check playlist ownership
// 	isOwner, err := isPlaylistOwnedByUser(tokenData.AccessToken, playlistID)
// 	if err != nil {
// 		http.Error(w, fmt.Sprintf("Error checking playlist ownership: %s", err), http.StatusInternalServerError)
// 		return
// 	}
// 	if !isOwner {
// 		w.WriteHeader(http.StatusForbidden)
// 		w.Write([]byte("The current playlist is not owned by you, so we cannot remove this song"))
// 		return
// 	}

// 	// Remove the song from the playlist
// 	err = removeSongFromPlaylist(tokenData.AccessToken, playlistID, songID)
// 	if err != nil {
// 		http.Error(w, fmt.Sprintf("Error removing song from playlist: %s", err), http.StatusInternalServerError)
// 		return
// 	}

// 	w.WriteHeader(http.StatusOK)
// 	w.Write([]byte(fmt.Sprintf("This song has been removed from your playlist %s", playlistName)))
// }

// func isPlaylistOwnedByUser(accessToken, playlistID string) (bool, error) {
// 	url := fmt.Sprintf("%s/playlists/%s", spotifyAPIBaseURL, playlistID)

// 	req, err := http.NewRequest("GET", url, nil)
// 	if err != nil {
// 		return false, err
// 	}
// 	req.Header.Add("Authorization", "Bearer "+accessToken)

// 	client := &http.Client{}
// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return false, err
// 	}
// 	defer resp.Body.Close()

// 	if resp.StatusCode != 200 {
// 		body, _ := ioutil.ReadAll(resp.Body)
// 		return false, fmt.Errorf("failed to retrieve playlist details: %s", body)
// 	}

// 	var data struct {
// 		Owner struct {
// 			ID string `json:"id"`
// 		} `json:"owner"`
// 	}

// 	err = json.NewDecoder(resp.Body).Decode(&data)
// 	if err != nil {
// 		return false, err
// 	}

// 	// Fetch the user's ID
// 	userID, err := getSpotifyUserID(accessToken)
// 	if err != nil {
// 		return false, err
// 	}

// 	return data.Owner.ID == userID, nil
// }

// func removeSongFromPlaylist(accessToken, playlistID, songID string) error {
// 	url := fmt.Sprintf("%s/playlists/%s/tracks", spotifyAPIBaseURL, playlistID)

// 	body := map[string]interface{}{
// 		"tracks": []map[string]string{
// 			{
// 				"uri": fmt.Sprintf("spotify:track:%s", songID),
// 			},
// 		},
// 	}

// 	jsonBody, err := json.Marshal(body)
// 	if err != nil {
// 		return err
// 	}

// 	req, err := http.NewRequest("DELETE", url, strings.NewReader(string(jsonBody)))
// 	if err != nil {
// 		return err
// 	}
// 	req.Header.Add("Authorization", "Bearer "+accessToken)
// 	req.Header.Add("Content-Type", "application/json")

// 	client := &http.Client{}
// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return err
// 	}
// 	defer resp.Body.Close()

// 	if resp.StatusCode != 200 {
// 		body, _ := ioutil.ReadAll(resp.Body)
// 		return fmt.Errorf("failed to remove song from playlist: %s", body)
// 	}

// 	return nil
// }
