package utils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

// Spotify API Endpoints
const (
	SpotifyAPIBaseURL = "https://api.spotify.com/v1"
	SpotifyAuthURL    = "https://accounts.spotify.com/authorize"
	SpotifyTokenURL   = "https://accounts.spotify.com/api/token"
)

func GetCurrentlyPlayingSong(accessToken string) (string, string, string, string, string, error) {
	req, err := http.NewRequest("GET", SpotifyAPIBaseURL+"/me/player/currently-playing", nil)
	if err != nil {
		return "", "", "", "", "", err
	}
	req.Header.Add("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", "", "", "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", "", "", "", "", err
	}

	var data map[string]interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return "", "", "", "", "", err
	}

	var songID, songName, artistName, playlistID, playlistName string

	if item, exists := data["item"].(map[string]interface{}); exists {
		if id, exists := item["id"].(string); exists {
			songID = id
		}
		if name, exists := item["name"].(string); exists {
			songName = name
		}

		if artists, exists := item["artists"].([]interface{}); exists && len(artists) > 0 {
			if firstArtist, ok := artists[0].(map[string]interface{}); ok {
				if artistNameValue, exists := firstArtist["name"].(string); exists {
					artistName = artistNameValue
				}
			}
		}
	}

	if context, exists := data["context"].(map[string]interface{}); exists {
		if uri, exists := context["uri"].(string); exists {
			if strings.HasPrefix(uri, "spotify:playlist:") {
				playlistID = strings.TrimPrefix(uri, "spotify:playlist:")
			}
		}
	}

	if playlistID != "" {
		playlistName, err = GetPlaylistName(accessToken, playlistID)
		if err != nil {
			// playlist name is only used for cosmetics, so don't hard error on failure to lookup name
			playlistName = "unknown"
		}
	}

	if songID == "" || songName == "" || artistName == "" {
		return "", "", "", "", "", fmt.Errorf("could not find the song ID, name, or artist")
	}

	return songID, songName, artistName, playlistID, playlistName, nil
}

func GetPlaylistName(accessToken, playlistID string) (string, error) {
	url := fmt.Sprintf("%s/playlists/%s", SpotifyAPIBaseURL, playlistID)

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
		return "", fmt.Errorf("failed to retrieve playlist name: %s", body)
	}

	var data struct {
		Name string `json:"name"`
	}

	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return "", err
	}

	return data.Name, nil
}

func RefreshSpotifyToken(refreshToken string, clientID string, clientSecret string) (*SpotifyAccessToken, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)

	req, err := http.NewRequest("POST", SpotifyTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(clientID, clientSecret)
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

func AddSongToPlaylist(accessToken, playlistID, songID string) error {
	url := fmt.Sprintf("%s/playlists/%s/tracks", SpotifyAPIBaseURL, playlistID)

	body := map[string]interface{}{
		"uris": []string{fmt.Sprintf("spotify:track:%s", songID)},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, strings.NewReader(string(jsonBody)))
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", "Bearer "+accessToken)
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("failed to add song to playlist: %s", body)
	}

	return nil
}

func RemoveSongFromPlaylist(accessToken, playlistID, songID string) error {
	url := fmt.Sprintf("%s/playlists/%s/tracks", SpotifyAPIBaseURL, playlistID)

	body := map[string]interface{}{
		"tracks": []map[string]string{
			{
				"uri": fmt.Sprintf("spotify:track:%s", songID),
			},
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("DELETE", url, strings.NewReader(string(jsonBody)))
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", "Bearer "+accessToken)
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("failed to remove song from playlist: %s", body)
	}

	return nil
}

func IsPlaylistOwnedByUser(accessToken, playlistID string) (bool, error) {
	url := fmt.Sprintf("%s/playlists/%s", SpotifyAPIBaseURL, playlistID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, err
	}
	req.Header.Add("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		return false, fmt.Errorf("failed to retrieve playlist details: %s", body)
	}

	var data struct {
		Owner struct {
			ID string `json:"id"`
		} `json:"owner"`
	}

	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return false, err
	}

	// Fetch the user's ID
	userID, err := GetSpotifyUserID(accessToken)
	if err != nil {
		return false, err
	}

	return data.Owner.ID == userID, nil
}

func IsSongInPlaylist(accessToken, playlistID, songID string) (bool, error) {
	url := fmt.Sprintf("%s/playlists/%s/tracks", SpotifyAPIBaseURL, playlistID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, err
	}
	req.Header.Add("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		return false, fmt.Errorf("failed to retrieve playlist tracks: %s", body)
	}

	var data struct {
		Items []struct {
			Track struct {
				ID string `json:"id"`
			} `json:"track"`
		} `json:"items"`
	}

	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return false, err
	}

	for _, item := range data.Items {
		if item.Track.ID == songID {
			return true, nil
		}
	}

	return false, nil
}
