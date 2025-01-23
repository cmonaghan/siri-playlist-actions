package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// SpotifyAccessToken holds the response from the token request
type SpotifyAccessToken struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// Retrieve credentials from environment variables
var spotifyClientID string
var spotifyClientSecret string
var apiToken string

const spotifyAPIBaseURL = "https://api.spotify.com/v1"

// Middleware to validate the Authorization token
func validateAPIToken(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract the Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "Unauthorized: Missing or invalid Authorization header", http.StatusUnauthorized)
			return
		}

		// Validate the API token
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token != apiToken {
			http.Error(w, "Unauthorized: Invalid API token", http.StatusUnauthorized)
			return
		}

		// If valid, proceed to the next handler
		next(w, r)
	}
}

func getSpotifyAccessToken() (string, error) {
	// Request body for the Client Credentials flow
	data := url.Values{}
	data.Set("grant_type", "client_credentials")

	// Create the HTTP request
	req, err := http.NewRequest("POST", "https://accounts.spotify.com/api/token", strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}

	// Set the Authorization header for Spotify API
	req.SetBasicAuth(spotifyClientID, spotifyClientSecret)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	// Make the HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Read and parse the response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("failed to get access token: %s", body)
	}

	// Parse the access token from the response body
	var token SpotifyAccessToken
	err = json.Unmarshal(body, &token)
	if err != nil {
		return "", err
	}

	return token.AccessToken, nil
}

func getCurrentlyPlayingSong(accessToken string) (string, error) {
	// Make the API request to get the currently playing track
	req, err := http.NewRequest("GET", spotifyAPIBaseURL+"/me/player/currently-playing", nil)
	if err != nil {
		return "", err
	}

	// Set the Authorization header
	req.Header.Add("Authorization", "Bearer "+accessToken)

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Check if there is no track playing
	if resp.StatusCode == 204 {
		return "", nil // No track playing
	}

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Parse the currently playing track
	var data map[string]interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return "", err
	}

	// Extract song ID
	if item, exists := data["item"].(map[string]interface{}); exists {
		if id, exists := item["id"].(string); exists {
			return id, nil
		}
	}

	return "", fmt.Errorf("could not find the song ID")
}

func removeSongFromPlaylist(w http.ResponseWriter, r *http.Request) {
	// Get Spotify access token
	accessToken, err := getSpotifyAccessToken()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting Spotify access token: %s", err), http.StatusInternalServerError)
		return
	}

	// Get the currently playing song's ID
	songID, err := getCurrentlyPlayingSong(accessToken)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting currently playing song: %s", err), http.StatusInternalServerError)
		return
	}

	// If no song is currently playing
	if songID == "" {
		http.Error(w, "No song is currently playing", http.StatusNotFound)
		return
	}

	// Simulate removing the song (you could remove it from a playlist here)
	log.Printf("Currently playing song ID: %s", songID)

	// Respond with success
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": fmt.Sprintf("Currently playing song ID: %s", songID),
	})
}

func main() {
	// Load environment variables from the .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	spotifyClientID = os.Getenv("SPOTIFY_CLIENT_ID")
	spotifyClientSecret = os.Getenv("SPOTIFY_CLIENT_SECRET")
	apiToken = os.Getenv("API_TOKEN")

	// Check if Spotify credentials and API token are set in the environment variables
	if spotifyClientID == "" || spotifyClientSecret == "" || apiToken == "" {
		log.Fatal("Spotify Client ID, Client Secret, and API Token must be set as environment variables.")
		return
	}

	http.HandleFunc("/remove-song", validateAPIToken(removeSongFromPlaylist))

	// Start the server on localhost port 8080
	log.Println("Starting server on :8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("Error starting server: ", err)
	}
}
