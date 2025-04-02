package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseError(t *testing.T) {
	payload := `{"error":{"code":0,"message":"something unknown happens","cause":"Post \"http://127.0.0.1:49799\": context deadline exceeded"}}`

	err := parseCallerResponse(strings.NewReader(payload))
	assert.Error(t, err)
}

func TestStreamWithAsync(t *testing.T) {
	mem := NewMemoryHandleRegistry()

	RegisterHttpSignalServiceServer(mem, NewHttpSignalServiceImpl(NewMemoryBus[string]()))

	server := httptest.NewServer(NewHttpHandler(mem))

	defer server.Close()

	host := server.URL
	caller := NewHttpClient(host, &http.Client{})

	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()
		client := CreateHttpSignalServiceClient(caller)

		ctx := context.Background()
		ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		err := client.Send(ctx, "inbox", "Hello")
		assert.NoError(t, err)
	}()

	go func() {
		defer wg.Done()

		client := CreateHttpSignalServiceClient(caller)

		ctx := context.Background()
		ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		msgs, errs := client.Recv(ctx, "inbox")

		select {
		case err := <-errs:
			assert.Fail(t, err.Error())
		case <-time.After(2 * time.Second):
			assert.Fail(t, "timeout")
		case msg := <-msgs:
			assert.Equal(t, "Hello", msg)
		}
	}()

	wg.Wait()
}
