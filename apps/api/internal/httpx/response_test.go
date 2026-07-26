package httpx_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/artumont/dotslashstream/internal/httpx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteJSON(t *testing.T) {
	recorder := httptest.NewRecorder()

	httpx.WriteJSON(recorder, http.StatusCreated, map[string]string{"status": "created"})

	assert.Equal(t, http.StatusCreated, recorder.Code)
	assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
	assert.JSONEq(t, `{"status":"created"}`, recorder.Body.String())
}

func TestWriteError(t *testing.T) {
	recorder := httptest.NewRecorder()

	httpx.WriteError(recorder, http.StatusBadRequest, "invalid request")

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	var response map[string]string
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	assert.Equal(t, "invalid request", response["error"])
}
