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

	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
)

var (
	spotifyClientID     string
	spotifyClientSecret string
	redirectURI         string
	apiToken            string
	sessionStore        = sessions.NewCookieStore([]byte("super-secret-key"))
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
	http.HandleFunc("/current-song", currentSongHandler) // New /current-song handler
	http.HandleFunc("/remove-song", validateAPIToken(removeSongFromPlaylist))

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
	// Get the authorization code from the query string
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Missing authorization code", http.StatusBadRequest)
		return
	}

	// Exchange the authorization code for an access token
	token, err := exchangeCodeForToken(code)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error exchanging code for token: %v", err), http.StatusInternalServerError)
		return
	}

	// Store the access token in a session
	session, _ := sessionStore.Get(r, "spotify-session")
	session.Values["access_token"] = token.AccessToken
	session.Values["refresh_token"] = token.RefreshToken
	session.Save(r, w)

	// Redirect the user to the /current-song endpoint
	http.Redirect(w, r, "/current-song", http.StatusFound)
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

func currentSongHandler(w http.ResponseWriter, r *http.Request) {
	var accessToken string

	// Check if the access token is provided in the Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
		// Extract the token from the Authorization header
		accessToken = strings.TrimPrefix(authHeader, "Bearer ")
	} else {
		// If the token is not in the header, fallback to the session store
		session, _ := sessionStore.Get(r, "spotify-session")
		accessToken = session.Values["access_token"].(string)
	}

	// If no valid access token is found
	if accessToken == "" {
		http.Error(w, "No valid access token found", http.StatusUnauthorized)
		return
	}

	// Get the currently playing song's ID and name
	songID, songName, err := getCurrentlyPlayingSong(accessToken)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting currently playing song: %s", err), http.StatusInternalServerError)
		return
	}

	// If no song is currently playing
	if songID == "" {
		http.Error(w, "No song is currently playing", http.StatusNotFound)
		return
	}

	// Display the currently playing song
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":    "success",
		"song_id":   songID,
		"song_name": songName,
	})
}

func getCurrentlyPlayingSong(accessToken string) (string, string, error) {
	// Make the API request to get the currently playing track
	req, err := http.NewRequest("GET", spotifyAPIBaseURL+"/me/player/currently-playing", nil)
	if err != nil {
		return "", "", err
	}

	// Set the Authorization header
	req.Header.Add("Authorization", "Bearer "+accessToken)

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	// Check if there is no track playing
	if resp.StatusCode == 204 {
		return "", "", nil // No track playing
	}

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	// Parse the currently playing track
	var data map[string]interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return "", "", err
	}

	// Extract song ID and name
	if item, exists := data["item"].(map[string]interface{}); exists {
		if id, exists := item["id"].(string); exists {
			if name, exists := item["name"].(string); exists {
				return id, name, nil
			}
		}
	}

	return "", "", fmt.Errorf("could not find the song ID or name")
}

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

func removeSongFromPlaylist(w http.ResponseWriter, r *http.Request) {
	// Get the session and access token
	session, _ := sessionStore.Get(r, "spotify-session")
	accessToken := session.Values["access_token"]
	if accessToken == nil {
		http.Error(w, "No valid access token found", http.StatusUnauthorized)
		return
	}

	// Get the currently playing song's ID
	songID, songName, err := getCurrentlyPlayingSong(accessToken.(string))
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
		"message": fmt.Sprintf("Currently playing song ID: %s, with name: %s", songID, songName),
	})
}

type SpotifyAccessToken struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}
