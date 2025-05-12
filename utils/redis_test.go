package utils

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockConn struct {
	data  map[string][]byte
	calls []string
}

func (m *mockConn) Do(commandName string, args ...interface{}) (reply interface{}, err error) {
	m.calls = append(m.calls, commandName)
	if commandName == "GET" {
		key := fmt.Sprintf("%v", args[0])
		if val, ok := m.data[key]; ok {
			return val, nil
		}
		return nil, redis.ErrNil
	}
	if commandName == "SET" {
		key := fmt.Sprintf("%v", args[0])
		if len(args) > 1 {
			if b, ok := args[1].([]byte); ok {
				m.data[key] = b
			} else if s, ok := args[1].(string); ok {
				m.data[key] = []byte(s)
			}
		}
		return nil, nil
	}
	return nil, nil
}
func (m *mockConn) Close() error                      { return nil }
func (m *mockConn) Err() error                        { return nil }
func (m *mockConn) Send(string, ...interface{}) error { return nil }
func (m *mockConn) Flush() error                      { return nil }
func (m *mockConn) Receive() (interface{}, error)     { return nil, nil }

type mockPool struct{ conn *mockConn }

func (p *mockPool) Get() redis.Conn { return p.conn }

func TestGetAPIKeyToUserAuthData_ExpiredTokenRefresh(t *testing.T) {
	// Setup initial expired token
	oldToken := &SpotifyAccessToken{
		AccessToken:  "expired-token",
		RefreshToken: "refresh-token",
	}
	expiredAuth := &UserAuthData{
		AccessToken:  oldToken.AccessToken,
		RefreshToken: oldToken.RefreshToken,
		ExpiresAt:    time.Now().Add(-time.Hour),
		UserID:       "user-123",
	}
	b, _ := json.Marshal(expiredAuth)

	// Setup mock Redis
	mock := &mockConn{data: map[string][]byte{"apiKey:test-api-key": b}}

	// Inline mock for RefreshSpotifyToken
	mockRefresh := func(refreshToken string) (*SpotifyAccessToken, error) {
		return &SpotifyAccessToken{
			AccessToken:  "new-token",
			RefreshToken: refreshToken,
		}, nil
	}

	// Inline mock for SetAPIKeyToUserAuthData
	mockSet := func(apiKey string, token *SpotifyAccessToken, userID string) error {
		updated := &UserAuthData{
			AccessToken:  token.AccessToken,
			RefreshToken: token.RefreshToken,
			ExpiresAt:    time.Now().Add(time.Hour),
			UserID:       userID,
		}
		b, _ := json.Marshal(updated)
		mock.data["apiKey:"+apiKey] = b
		return nil
	}

	// Call function under test
	result, err := GetAPIKeyToUserAuthData("test-api-key", mock, mockRefresh, mockSet)
	require.NoError(t, err)
	assert.Equal(t, "new-token", result.AccessToken)
	assert.Equal(t, "refresh-token", result.RefreshToken)
	assert.Equal(t, "user-123", result.UserID)
	assert.True(t, result.ExpiresAt.After(time.Now()))
}
