package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/joho/godotenv"
)

type TokenData struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type SpotifyAccessToken struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

var (
	spotifyClientID     string
	spotifyClientSecret string
	redirectURI         string
	redisPool           *redis.Pool
)

const spotifyAPIBaseURL = "https://api.spotify.com/v1"
const spotifyAuthURL = "https://accounts.spotify.com/authorize"
const spotifyTokenURL = "https://accounts.spotify.com/api/token"

func main() {
	// Load environment variables from the .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Retrieve environment variables after loading them
	spotifyClientID = os.Getenv("SPOTIFY_CLIENT_ID")
	spotifyClientSecret = os.Getenv("SPOTIFY_CLIENT_SECRET")
	redirectURI = os.Getenv("REDIRECT_URI") // This should match the one set in Spotify Developer Dashboard
	redisURL := os.Getenv("KV_URL")

	initRedis(redisURL)
	defer redisPool.Close()

	// Check if Spotify credentials and API token are set in the environment variables
	if spotifyClientID == "" || spotifyClientSecret == "" || redirectURI == "" || redisURL == "" {
		log.Fatal("Missing required environment variables.")
		return
	}

	// endpoints for initial setup
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/callback", callbackHandler)
	http.HandleFunc("/setup", completeSetupHandler)
	// endpoints for regular usage
	http.HandleFunc("/current-song", currentSongHandler)

	// Serve static files (e.g., images, stylesheets) from the "static" directory
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Start the server on localhost port 8080
	log.Println("Starting server on :8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("Error starting server: ", err)
	}
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	// Generate the Spotify authorization URL
	authURL := fmt.Sprintf("%s?client_id=%s&response_type=code&redirect_uri=%s&scope=user-read-playback-state user-modify-playback-state", spotifyAuthURL, spotifyClientID, url.QueryEscape(redirectURI))

	// Redirect the user to the Spotify authorization page
	http.Redirect(w, r, authURL, http.StatusFound)
}

func callbackHandler(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Missing authorization code", http.StatusBadRequest)
		return
	}

	token, err := exchangeCodeForToken(code)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error exchanging code for token: %v", err), http.StatusInternalServerError)
		return
	}

	apiKey := generateAPIKey()
	err = storeAPIKey(apiKey, token)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to store API key: %v", err), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/setup?api_key=%s", apiKey), http.StatusFound)
}

func exchangeCodeForToken(code string) (*SpotifyAccessToken, error) {
	// Prepare the request data
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)

	// Send the request to exchange the code for an access token
	req, err := http.NewRequest("POST", spotifyTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(spotifyClientID, spotifyClientSecret)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Parse the response to get the access token and refresh token
	var token SpotifyAccessToken
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get access token: %s", body)
	}

	err = json.Unmarshal(body, &token)
	if err != nil {
		return nil, err
	}

	return &token, nil
}

func completeSetupHandler(w http.ResponseWriter, r *http.Request) {
	apiKey := r.URL.Query().Get("api_key")
	if apiKey == "" {
		http.Error(w, "API key not found", http.StatusBadRequest)
		return
	}

	tokenData, err := getTokenData(apiKey)
	if err != nil {
		http.Error(w, "Invalid API Key", http.StatusUnauthorized)
		return
	}

	// Fetch the currently playing song details
	songID, songName, artistName, playlistID, err := getCurrentlyPlayingSong(tokenData.AccessToken)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting current song: %s", err), http.StatusInternalServerError)
		return
	}

	// Define the template for the setup page
	tmpl := `
		<!DOCTYPE html>
		<html lang="en">
		<head>
			<meta charset="UTF-8">
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
			<title>Complete Setup</title>
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
					background-color: #007BFF;
					color: white;
					border: none;
					padding: 10px 15px;
					font-size: 16px;
					border-radius: 5px;
					cursor: pointer;
				}
				button:hover {
					background-color: #0056b3;
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
			<button onclick="copyApiKey()">Copy API Key</button>
			<h2>Currently Playing</h2>
			<ul>
				<li><strong>Song:</strong> {{.CurrentSong}}</li>
				<li><strong>Artist:</strong> {{.ArtistName}}</li>
				<li><strong>Playlist ID:</strong> {{.PlaylistID}}</li>
			</ul>
			<p>Now, to use this API key in Siri Shortcuts:</p>
			<ol>
				<li>Open the Shortcuts app on your iPhone.</li>
				<li>Tap "+" in the upper right.</li>
				<li>Search for "Get Contents of URL".</li>
				<li>Set the URL to <code>http://localhost:8080/current-song</code>.</li>
				<li>Set "Method" to "POST".</li>
				<li>Set "Headers" to Key: <code>X-API-Key</code> and Text: <code>{{.APIKey}}</code></li>
				<li>You can now use this Shortcut to check the current song!</li>
			</ol>
			<img src="/static/example-shortcut.jpeg" alt="Apple Shortcut Example Setup" />
			<script>
				function copyApiKey() {
					// Get the API key text
					const token = document.getElementById("apiKey").innerText;
					
					// Copy the API key to clipboard
					navigator.clipboard.writeText(token);
				}
			</script>
		</body>
		</html>
	`

	// Create a template and execute it
	t, err := template.New("setup").Parse(tmpl)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error parsing template: %s", err), http.StatusInternalServerError)
		return
	}

	// Render the page with the API key and song details
	err = t.Execute(w, map[string]interface{}{
		"APIKey":      apiKey,
		"CurrentSong": songName,
		"ArtistName":  artistName,
		"PlaylistID":  playlistID,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("Error rendering template: %s", err), http.StatusInternalServerError)
	}

	// Use songID to satisfy the compiler, even though it's not needed anywhere in this function
	_ = songID
}

func currentSongHandler(w http.ResponseWriter, r *http.Request) {
	apiKey := r.Header.Get("X-API-Key")
	if apiKey == "" {
		http.Error(w, "Missing API Key", http.StatusUnauthorized)
		return
	}

	tokenData, err := getTokenData(apiKey)
	if err != nil {
		http.Error(w, "Invalid API Key", http.StatusUnauthorized)
		return
	}

	// Refresh token if expired
	if time.Now().After(tokenData.ExpiresAt) {
		newToken, err := refreshSpotifyToken(tokenData.RefreshToken)
		if err != nil {
			http.Error(w, "Failed to refresh access token", http.StatusInternalServerError)
			return
		}

		tokenData.AccessToken = newToken.AccessToken
		tokenData.ExpiresAt = time.Now().Add(time.Hour)

		// Persist the updated token data
		err = storeAPIKey(apiKey, &SpotifyAccessToken{
			AccessToken:  tokenData.AccessToken,
			RefreshToken: tokenData.RefreshToken,
		})
		if err != nil {
			http.Error(w, "Failed to persist updated token", http.StatusInternalServerError)
			return
		}
	}

	// Fetch the current song and playlist ID
	songID, songName, artistName, playlistID, err := getCurrentlyPlayingSong(tokenData.AccessToken)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting current song: %s", err), http.StatusInternalServerError)
		return
	}

	if songID == "" {
		http.Error(w, "No song is currently playing", http.StatusNotFound)
		return
	}

	// Use playlistID to satisfy the compiler, even though it's not included in the response
	_ = playlistID

	// Respond with the song and playlist details
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"current_song": songName,
		"artist_name":  artistName,
	})
}

func getCurrentlyPlayingSong(accessToken string) (string, string, string, string, error) {
	// Make the API request to get the currently playing track
	req, err := http.NewRequest("GET", spotifyAPIBaseURL+"/me/player/currently-playing", nil)
	if err != nil {
		return "", "", "", "", err
	}

	// Set the Authorization header
	req.Header.Add("Authorization", "Bearer "+accessToken)

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", "", "", err
	}
	defer resp.Body.Close()

	// Check if there is no track playing
	if resp.StatusCode == 204 {
		return "", "", "", "", nil // No track playing
	}

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", "", "", "", err
	}

	// Parse the currently playing track
	var data map[string]interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return "", "", "", "", err
	}

	// Extract song ID, name, artist name, and playlist ID
	var songID, songName, artistName, playlistID string

	if item, exists := data["item"].(map[string]interface{}); exists {
		// Get the song ID and name
		if id, exists := item["id"].(string); exists {
			songID = id
		}
		if name, exists := item["name"].(string); exists {
			songName = name
		}

		// Get the artist name
		if artists, exists := item["artists"].([]interface{}); exists && len(artists) > 0 {
			if firstArtist, ok := artists[0].(map[string]interface{}); ok {
				if artistNameValue, exists := firstArtist["name"].(string); exists {
					artistName = artistNameValue
				}
			}
		}
	}

	// Extract playlist ID from context
	if context, exists := data["context"].(map[string]interface{}); exists {
		if uri, exists := context["uri"].(string); exists && strings.HasPrefix(uri, "spotify:playlist:") {
			playlistID = strings.TrimPrefix(uri, "spotify:playlist:")
		}
	}

	if songID == "" || songName == "" {
		return "", "", "", "", fmt.Errorf("could not find the song ID or name")
	}

	return songID, songName, artistName, playlistID, nil
}

func generateAPIKey() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const length = 32
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	apiKey := make([]byte, length)
	for i := range apiKey {
		apiKey[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(apiKey)
}

func storeAPIKey(apiKey string, token *SpotifyAccessToken) error {
	conn := redisPool.Get()
	defer conn.Close()

	tokenData := TokenData{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Hour), // Spotify tokens typically expire after 1 hour
	}

	data, err := json.Marshal(tokenData)
	if err != nil {
		return fmt.Errorf("failed to marshal token data: %v", err)
	}

	_, err = conn.Do("SET", apiKey, data, "EX", 3600*24*30) // Set expiration to 30 days
	if err != nil {
		return fmt.Errorf("failed to store API key in Redis: %v", err)
	}

	return nil
}

func refreshSpotifyToken(refreshToken string) (*SpotifyAccessToken, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)

	req, err := http.NewRequest("POST", spotifyTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(spotifyClientID, spotifyClientSecret)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var token SpotifyAccessToken
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to refresh token: %s", body)
	}

	err = json.Unmarshal(body, &token)
	if err != nil {
		return nil, err
	}

	return &token, nil
}

func getTokenData(apiKey string) (*TokenData, error) {
	conn := redisPool.Get()
	defer conn.Close()

	data, err := redis.Bytes(conn.Do("GET", apiKey))
	if err == redis.ErrNil {
		return nil, fmt.Errorf("API key not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve API key from Redis: %v", err)
	}

	var tokenData TokenData
	err = json.Unmarshal(data, &tokenData)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal token data: %v", err)
	}

	return &tokenData, nil
}

func initRedis(redisURL string) {
	if redisURL == "" {
		log.Fatal("KV_URL environment variable is not set")
	}

	redisPool = &redis.Pool{
		MaxIdle:   10,
		MaxActive: 100,
		Wait:      true,
		Dial: func() (redis.Conn, error) {
			c, err := redis.DialURL(redisURL)
			if err != nil {
				return nil, fmt.Errorf("failed to connect to Redis: %v", err)
			}
			return c, nil
		},
	}
}
