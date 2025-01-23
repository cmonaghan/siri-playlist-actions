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
	"text/template"

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

	// Redirect the user to the /setup endpoint
	http.Redirect(w, r, "/setup", http.StatusFound)
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
	// Retrieve the access token from the session
	session, _ := sessionStore.Get(r, "spotify-session")
	accessToken := session.Values["access_token"]

	// If no valid access token is found
	if accessToken == nil {
		http.Error(w, "No valid access token found", http.StatusUnauthorized)
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
			<p>Your Spotify access token is:</p>
			<pre id="accessToken">{{.AccessToken}}</pre>
			<button onclick="copyToken()">Copy Token</button>
			<p>Now, to use this token in Siri Shortcuts:</p>
			<ol>
				<li>Open the Shortcuts app on your iPhone.</li>
				<li>Tap "+" in the upper right.</li>
				<li>Search for "Get Contents of URL".</li>
				<li>Set the URL to <code>http://localhost:8080/current-song</code>.</li>
				<li>Set "Method" to "POST".</li>
				<li>Set "Headers" to Key=<code>Authorization</code> and Text=<code>Bearer YOUR_ACCESS_TOKEN</code>.</li>
				<li>You can now use this Shortcut to check the current song!</li>
			</ol>
			<img src="/static/example-shortcut.jpeg" alt="Apple Shortcut Example Setup" />
			<script>
				function copyToken() {
					// Get the token text
					const token = document.getElementById("accessToken").innerText;
					
					// Copy the token to clipboard
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

	// Render the page with the access token
	err = t.Execute(w, map[string]interface{}{
		"AccessToken": accessToken,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("Error rendering template: %s", err), http.StatusInternalServerError)
	}
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

	// Get the currently playing song's ID, name, and artist
	songID, songName, artistName, err := getCurrentlyPlayingSong(accessToken)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting currently playing song: %s", err), http.StatusInternalServerError)
		return
	}

	// If no song is currently playing
	if songID == "" {
		http.Error(w, "No song is currently playing", http.StatusNotFound)
		return
	}

	// Display the currently playing song and artist
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

type SpotifyAccessToken struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}
