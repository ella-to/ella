package main

import (
	"context"
	"io"
	"net/http/httptest"
)

type httpUserService struct {
}

var _ HttpUserService = (*httpUserService)(nil)

func (s *httpUserService) GetById(ctx context.Context, userId string) (result *User, err error) {
	return
}

func (s *httpUserService) Upload(ctx context.Context, id string, files func() (filename string, content io.Reader, err error)) (err error) {
	return
}

func NewHttpUserService() *httpUserService {
	return &httpUserService{}
}

func main() {
	registry := CreateRegistry()

	RegisterHttpUserServiceServer(registry, NewHttpUserService())

	handler := NewHttpReceiver(registry)

	server := httptest.NewServer(handler)
	defer server.Close()
}
