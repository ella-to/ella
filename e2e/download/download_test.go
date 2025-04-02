package download

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCallHttpMethod(t *testing.T) {
	mem := NewMemoryHandleRegistry()

	RegisterHttpDownloadServiceServer(mem, &HttpDownloadServiceImpl{})

	server := httptest.NewServer(NewHttpHandler(mem))

	host := server.URL
	httpClient := &http.Client{}

	caller := NewHttpClient(host, httpClient)

	client := CreateHttpDownloadServiceClient(caller)

	r, filename, contentType, err := client.Get(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, "text/plain", contentType)
	assert.Equal(t, "hello.txt", filename)

	data, err := io.ReadAll(r)
	assert.NoError(t, err)
	assert.Equal(t, "Hello, World!", string(data))
}
