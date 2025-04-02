package stream

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHttpStream(t *testing.T) {
	mem := NewMemoryHandleRegistry()

	RegisterHttpEventServiceServer(mem, &HttpEventServiceImpl{})

	server := httptest.NewServer(NewHttpHandler(mem))

	host := server.URL
	httpClient := &http.Client{}

	caller := NewHttpClient(host, httpClient)

	client := CreateHttpEventServiceClient(caller)

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	results, errs := client.GetRandomValues(ctx)

	for {
		select {
		case err, ok := <-errs:
			if !ok {
				return
			}
			t.Fatal(err)
		case result, ok := <-results:
			if !ok {
				return
			}
			println(result)
		}
	}
}
