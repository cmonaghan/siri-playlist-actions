package utils

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
	"time"
)

// SpotifyAccessToken represents the token structure
type SpotifyAccessToken struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

// UserAuthData represents stored token information
type UserAuthData struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	UserID       string    `json:"user_id"`
}

// Exchanges authorization code for access token
func ExchangeCodeForToken(code string) (*SpotifyAccessToken, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", os.Getenv("REDIRECT_URI"))

	req, err := http.NewRequest("POST", "https://accounts.spotify.com/api/token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(os.Getenv("SPOTIFY_CLIENT_ID"), os.Getenv("SPOTIFY_CLIENT_SECRET"))
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
		return nil, fmt.Errorf("failed to get access token: %s", body)
	}

	err = json.Unmarshal(body, &token)
	return &token, err
}

// Fetches Spotify user ID
func GetSpotifyUserID(accessToken string) (string, error) {
	url := "https://api.spotify.com/v1/me"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to fetch user ID: %s", body)
	}

	var data struct {
		ID string `json:"id"`
	}
	err = json.NewDecoder(resp.Body).Decode(&data)
	return data.ID, err
}

func GenerateAPIKey() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const length = 32
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	apiKey := make([]byte, length)
	for i := range apiKey {
		apiKey[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(apiKey)
}

func RefreshSpotifyToken(refreshToken string) (*SpotifyAccessToken, error) {
	// Prepare request data
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)

	req, err := http.NewRequest("POST", SpotifyTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(os.Getenv("SPOTIFY_CLIENT_ID"), os.Getenv("SPOTIFY_CLIENT_SECRET"))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Parse response
	var newToken SpotifyAccessToken
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to refresh token: %s", body)
	}

	err = json.Unmarshal(body, &newToken)
	if err != nil {
		return nil, err
	}

	log.Println("✅ Successfully refreshed Spotify token!")

	return &newToken, nil
}
