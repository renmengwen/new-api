package xunfei

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"runtime/pprof"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/dto"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

func TestXunfeiMakeRequestDoesNotLeakCancelWatcherForBackgroundContext(t *testing.T) {
	before := countXunfeiCancelWatchers(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		_, _, err = conn.ReadMessage()
		require.NoError(t, err)

		response := XunfeiChatResponse{}
		response.Payload.Choices.Status = 2
		response.Payload.Choices.Text = []XunfeiChatResponseTextItem{
			{Role: "assistant", Content: "ok"},
		}
		require.NoError(t, conn.WriteJSON(response))
	}))
	defer server.Close()

	authURL := "ws" + strings.TrimPrefix(server.URL, "http")
	dataChan, stopChan, err := xunfeiMakeRequest(context.Background(), dto.GeneralOpenAIRequest{
		Model: "spark-test",
		Messages: []dto.Message{
			{Role: "user", Content: "ping"},
		},
	}, "general", authURL, "app-id")
	require.NoError(t, err)

	select {
	case <-dataChan:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for xunfei response")
	}

	select {
	case <-stopChan:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for xunfei stop signal")
	}

	require.Eventually(t, func() bool {
		return countXunfeiCancelWatchers(t) <= before
	}, time.Second, 10*time.Millisecond)
}

func countXunfeiCancelWatchers(t *testing.T) int {
	t.Helper()

	var buf bytes.Buffer
	require.NoError(t, pprof.Lookup("goroutine").WriteTo(&buf, 2))
	return strings.Count(buf.String(), "xunfeiMakeRequest.func1.2")
}
