package model

import (
	"encoding/json"
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestLoginRequest_Serialization(t *testing.T) {
	req := LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}

	data, err := json.Marshal(req)
	assert.NoError(t, err)
	assert.Contains(t, string(data), `"email":"test@example.com"`)
	assert.Contains(t, string(data), `"password":"password123"`)

	var decoded LoginRequest
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, req.Email, decoded.Email)
	assert.Equal(t, req.Password, decoded.Password)
}

func TestRefreshRequest_Serialization(t *testing.T) {
	req := RefreshRequest{
		RefreshToken: "some-token",
	}

	data, err := json.Marshal(req)
	assert.NoError(t, err)
	assert.Contains(t, string(data), `"refresh_token":"some-token"`)

	var decoded RefreshRequest
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, req.RefreshToken, decoded.RefreshToken)
}

func TestAuthResponse_Serialization(t *testing.T) {
	resp := AuthResponse{
		AccessToken:  "access",
		RefreshToken: "refresh",
		User: &User{
			ID:    "123",
			Email: "test@example.com",
		},
	}

	data, err := json.Marshal(resp)
	assert.NoError(t, err)
	assert.Contains(t, string(data), `"accessToken":"access"`)
	assert.Contains(t, string(data), `"refreshToken":"refresh"`)
	assert.Contains(t, string(data), `"user":{`)

	var decoded AuthResponse
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, resp.AccessToken, decoded.AccessToken)
	assert.Equal(t, resp.RefreshToken, decoded.RefreshToken)
	assert.Equal(t, resp.User.ID, decoded.User.ID)
}
