package upload

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUpload(t *testing.T) {
	mem := NewMemoryHandleRegistry()

	RegisterHttpStorageServiceServer(mem, &HttpStorageServiceImpl{})

	server := httptest.NewServer(NewHttpHandler(mem))

	defer server.Close()

	caller := NewHttpClient(server.URL, &http.Client{})

	client := CreateHttpStorageServiceClient(caller)

	files := make(chan *struct {
		FileName string
		Content  io.Reader
	}, 1)
	for i := 0; i < 1; i++ {
		files <- &struct {
			FileName string
			Content  io.Reader
		}{
			FileName: fmt.Sprintf("test%d.txt", i),
			Content:  strings.NewReader("Hello World"),
		}
	}
	close(files)

	results, err := client.UploadFiles(context.Background(), "test", func() (string, io.Reader, error) {
		file, ok := <-files
		if !ok {
			return "", nil, io.EOF
		}

		return file.FileName, file.Content, nil
	})
	assert.NoError(t, err)
	assert.Equal(t, []*File{
		{
			Name: "test0.txt",
			Size: 11,
		},
		// {
		// 	Name: "test1.txt",
		// 	Size: 11,
		// },
	}, results)
}
