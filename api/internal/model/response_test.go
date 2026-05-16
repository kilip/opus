package model

import (
	"encoding/json"
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestNewSuccessResponse(t *testing.T) {
	data := map[string]string{"foo": "bar"}
	resp := NewSuccessResponse(data)

	assert.True(t, resp.Success)
	assert.Equal(t, data, resp.Data)
	assert.Nil(t, resp.Error)

	bytes, err := json.Marshal(resp)
	assert.NoError(t, err)
	assert.Contains(t, string(bytes), `"success":true`)
	assert.Contains(t, string(bytes), `"data":{"foo":"bar"}`)
}

func TestNewErrorResponse(t *testing.T) {
	resp := NewErrorResponse("ERR_CODE", "Something went wrong")

	assert.False(t, resp.Success)
	assert.Nil(t, resp.Data)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, "ERR_CODE", resp.Error.Code)
	assert.Equal(t, "Something went wrong", resp.Error.Message)

	bytes, err := json.Marshal(resp)
	assert.NoError(t, err)
	assert.Contains(t, string(bytes), `"success":false`)
	assert.Contains(t, string(bytes), `"code":"ERR_CODE"`)
	assert.Contains(t, string(bytes), `"message":"Something went wrong"`)
}
