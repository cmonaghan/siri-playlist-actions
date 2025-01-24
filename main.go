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
	"sync"
	"text/template"
	"time"

	"github.com/joho/godotenv"
)

type TokenData struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

type SpotifyAccessToken struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

var (
	apiKeys    = make(map[string]*TokenData)
	apiKeysMux sync.Mutex
)

var (
	spotifyClientID     string
	spotifyClientSecret string
	redirectURI         string
	apiToken            string
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
	apiToken = os.Getenv("API_TOKEN")

	// Check if Spotify credentials and API token are set in the environment variables
	if spotifyClientID == "" || spotifyClientSecret == "" || apiToken == "" || redirectURI == "" {
		log.Fatal("Spotify Client ID, Client Secret, API Token, and Redirect URI must be set as environment variables.")
		return
	}

	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/callback", callbackHandler)
	http.HandleFunc("/current-song", currentSongHandler)
	http.HandleFunc("/setup", completeSetupHandler)

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
	storeAPIKey(apiKey, token)

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
			<p>Now, to use this api key in Siri Shortcuts:</p>
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
					// Get the token text
					const token = document.getElementById("apiKey").innerText;
					
					// Copy the token to clipboard
					navigator.clipboard.writeText(token);
				}
			</script>
		</body>
		</html>
	`

	t, err := template.New("setup").Parse(tmpl)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error parsing template: %s", err), http.StatusInternalServerError)
		return
	}

	err = t.Execute(w, map[string]string{"APIKey": apiKey})
	if err != nil {
		http.Error(w, fmt.Sprintf("Error rendering template: %s", err), http.StatusInternalServerError)
	}
}

func currentSongHandler(w http.ResponseWriter, r *http.Request) {
	apiKey := r.Header.Get("X-API-Key")
	if apiKey == "" {
		http.Error(w, "Missing API Key", http.StatusUnauthorized)
		return
	}

	apiKeysMux.Lock()
	tokenData, exists := apiKeys[apiKey]
	apiKeysMux.Unlock()

	if !exists {
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
	}

	// Fetch the current song
	songID, songName, artistName, err := getCurrentlyPlayingSong(tokenData.AccessToken)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting current song: %s", err), http.StatusInternalServerError)
		return
	}

	if songID == "" {
		http.Error(w, "No song is currently playing", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"current_song": songName,
		"artist_name":  artistName,
	})
}

func getCurrentlyPlayingSong(accessToken string) (string, string, string, error) {
	// Make the API request to get the currently playing track
	req, err := http.NewRequest("GET", spotifyAPIBaseURL+"/me/player/currently-playing", nil)
	if err != nil {
		return "", "", "", err
	}

	// Set the Authorization header
	req.Header.Add("Authorization", "Bearer "+accessToken)

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()

	// Check if there is no track playing
	if resp.StatusCode == 204 {
		return "", "", "", nil // No track playing
	}

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", "", "", err
	}

	// Parse the currently playing track
	var data map[string]interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return "", "", "", err
	}

	// Extract song ID, name, and artist name
	var songID, songName, artistName string

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

	if songID == "" || songName == "" {
		return "", "", "", fmt.Errorf("could not find the song ID or name")
	}

	return songID, songName, artistName, nil
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

func storeAPIKey(apiKey string, token *SpotifyAccessToken) {
	apiKeysMux.Lock()
	defer apiKeysMux.Unlock()
	apiKeys[apiKey] = &TokenData{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Hour), // Spotify tokens typically expire after 1 hour
	}
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
