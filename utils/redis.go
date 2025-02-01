package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gomodule/redigo/redis"
)

var redisPool *redis.Pool

func InitRedis() (*redis.Pool, error) {
	redisURL := os.Getenv("KV_URL")
	if redisURL == "" {
		log.Fatal("❌ KV_URL environment variable is not set")
	}

	redisPool = &redis.Pool{
		MaxIdle:   10,
		MaxActive: 100,
		Wait:      true,
		Dial: func() (redis.Conn, error) {
			c, err := redis.DialURL(redisURL)
			if err != nil {
				log.Fatalf("❌ Failed to connect to Redis: %v", err)
			}
			return c, nil
		},
	}

	return redisPool, nil
}

// Stores API key and token data
func StoreAPIKey(apiKey string, token *SpotifyAccessToken) error {
	conn := redisPool.Get()
	defer conn.Close()

	tokenData := TokenData{
		AccessToken: token.AccessToken,
		ExpiresAt:   time.Now().Add(time.Hour),
	}

	data, err := json.Marshal(tokenData)
	if err != nil {
		return fmt.Errorf("failed to marshal token data: %v", err)
	}

	_, err = conn.Do("SET", apiKey, data, "EX", 3600*24*30)
	return err
}

// Retrieves token data using API key
func GetTokenData(apiKey string) (*TokenData, error) {
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

// Maps user ID to API key
func MapUserIDToAPIKey(userID, apiKey string) error {
	conn := redisPool.Get()
	defer conn.Close()

	_, err := conn.Do("SET", fmt.Sprintf("user:%s", userID), apiKey)
	return err
}

// Retrieves API key using user ID
func GetAPIKeyByUserID(userID string) (string, error) {
	conn := redisPool.Get()
	defer conn.Close()

	apiKey, err := redis.String(conn.Do("GET", fmt.Sprintf("user:%s", userID)))
	if err == redis.ErrNil {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to retrieve API key by user ID: %v", err)
	}

	return apiKey, nil
}

// GetUserIDByAPIKey retrieves the user ID associated with an API key
func GetUserIDByAPIKey(apiKey string) (string, error) {
	conn := redisPool.Get()
	defer conn.Close()

	userID, err := redis.String(conn.Do("GET", fmt.Sprintf("apiKey:%s:user", apiKey)))
	if err == redis.ErrNil {
		return "", fmt.Errorf("API key not found")
	}
	if err != nil {
		return "", fmt.Errorf("failed to retrieve user ID: %v", err)
	}

	return userID, nil
}

// DeleteAPIKey removes the API key from Redis
func DeleteAPIKey(apiKey string) error {
	conn := redisPool.Get()
	defer conn.Close()

	_, err := conn.Do("DEL", fmt.Sprintf("apiKey:%s", apiKey))
	if err != nil {
		return fmt.Errorf("failed to delete API key: %v", err)
	}

	return nil
}

// DeleteUserIDMapping removes the user-to-API key mapping
func DeleteUserIDMapping(userID string) error {
	conn := redisPool.Get()
	defer conn.Close()

	_, err := conn.Do("DEL", fmt.Sprintf("user:%s", userID))
	if err != nil {
		return fmt.Errorf("failed to delete user mapping: %v", err)
	}

	return nil
}
