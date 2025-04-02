package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCallHttpMethod(t *testing.T) {
	mem := NewMemoryHandleRegistry()

	RegisterHttpPeopleServiceServer(mem, &HttpPeopleServiceImpl{})

	server := httptest.NewServer(NewHttpHandler(mem))

	host := server.URL
	httpClient := &http.Client{}

	caller := NewHttpClient(host, httpClient)

	client := CreateHttpPeopleServiceClient(caller)

	result, err := client.GetRandom(context.Background(), 10)
	assert.NoError(t, err)
	assert.Equal(t, &Person{
		Name:    "Ella",
		Age:     10,
		Emotion: Emotion_Excited,
	}, result)

	result, err = client.GetRandom(context.Background(), -1)
	assert.Error(t, err)
	assert.Equal(t, &Person{}, result)
}
