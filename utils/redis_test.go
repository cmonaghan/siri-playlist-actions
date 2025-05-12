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
	if commandName == "DEL" {
		key := fmt.Sprintf("%v", args[0])
		delete(m.data, key)
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

func TestGetUserIDToAPIKey_Success(t *testing.T) {
	mock := &mockConn{data: map[string][]byte{"user:user-123": []byte("api-key-abc")}}
	apiKey, err := GetUserIDToAPIKey("user-123", mock)
	assert.NoError(t, err)
	assert.Equal(t, "api-key-abc", apiKey)
}

func TestGetUserIDToAPIKey_NotFound(t *testing.T) {
	mock := &mockConn{data: map[string][]byte{}}
	apiKey, err := GetUserIDToAPIKey("user-unknown", mock)
	assert.NoError(t, err)
	assert.Equal(t, "", apiKey)
}

func TestGetUserIDToAPIKey_RedisError(t *testing.T) {
	mock := &mockConn{data: map[string][]byte{}}
	// Simulate error by wrapping mockConn.Do
	errConn := &errorConn{mockConn: mock}
	_, err := GetUserIDToAPIKey("user-err", errConn)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to retrieve API key by user ID")
}

type errorConn struct {
	*mockConn
}

func (e *errorConn) Do(commandName string, args ...interface{}) (reply interface{}, err error) {
	return nil, fmt.Errorf("redis error")
}

func TestDeleteAPIKey_Success(t *testing.T) {
	mock := &mockConn{data: map[string][]byte{"apiKey:test-key": []byte("value")}}
	err := DeleteAPIKey("test-key", mock)
	assert.NoError(t, err)
	_, exists := mock.data["apiKey:test-key"]
	assert.False(t, exists)
}

func TestDeleteAPIKey_RedisError(t *testing.T) {
	mock := &mockConn{data: map[string][]byte{"apiKey:test-key": []byte("value")}}
	errConn := &errorConn{mockConn: mock}
	err := DeleteAPIKey("test-key", errConn)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete API key")
}

func TestSetAPIKeyToUserAuthData_Success(t *testing.T) {
	mock := &mockConn{data: map[string][]byte{}}
	token := &SpotifyAccessToken{AccessToken: "token", RefreshToken: "refresh"}
	err := SetAPIKeyToUserAuthData("api-key", token, "user-1", mock)
	assert.NoError(t, err)
	stored, ok := mock.data["apiKey:api-key"]
	assert.True(t, ok)
	var auth UserAuthData
	err = json.Unmarshal(stored, &auth)
	assert.NoError(t, err)
	assert.Equal(t, "token", auth.AccessToken)
	assert.Equal(t, "refresh", auth.RefreshToken)
	assert.Equal(t, "user-1", auth.UserID)
}

func TestSetAPIKeyToUserAuthData_Error(t *testing.T) {
	errConn := &errorConn{mockConn: &mockConn{data: map[string][]byte{}}}
	token := &SpotifyAccessToken{AccessToken: "token", RefreshToken: "refresh"}
	err := SetAPIKeyToUserAuthData("api-key", token, "user-1", errConn)
	assert.Error(t, err)
}

func TestSetUserIDToAPIKey_Success(t *testing.T) {
	mock := &mockConn{data: map[string][]byte{}}
	err := SetUserIDToAPIKey("user-1", "api-key", mock)
	assert.NoError(t, err)
	val, ok := mock.data["user:user-1"]
	assert.True(t, ok)
	assert.Equal(t, []byte("api-key"), val)
}

func TestSetUserIDToAPIKey_Error(t *testing.T) {
	errConn := &errorConn{mockConn: &mockConn{data: map[string][]byte{}}}
	err := SetUserIDToAPIKey("user-1", "api-key", errConn)
	assert.Error(t, err)
}

func TestDeleteUserID_Success(t *testing.T) {
	mock := &mockConn{data: map[string][]byte{"user:user-1": []byte("api-key")}}
	err := DeleteUserID("user-1", mock)
	assert.NoError(t, err)
	_, ok := mock.data["user:user-1"]
	assert.False(t, ok)
}

func TestDeleteUserID_Error(t *testing.T) {
	errConn := &errorConn{mockConn: &mockConn{data: map[string][]byte{"user:user-1": []byte("api-key")}}}
	err := DeleteUserID("user-1", errConn)
	assert.Error(t, err)
}
