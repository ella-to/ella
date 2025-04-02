package upload

import (
	"bytes"
	"context"
	"errors"
	"io"
)

type HttpStorageServiceImpl struct{}

var _ HttpStorageService = (*HttpStorageServiceImpl)(nil)

func (s *HttpStorageServiceImpl) UploadFiles(ctx context.Context, id string, files func() (string, io.Reader, error)) (results []*File, err error) {
	results = make([]*File, 0)

	for {
		filename, content, err := files()
		if errors.Is(err, io.EOF) {
			return results, nil
		} else if err != nil {
			return nil, err
		}

		var buffer bytes.Buffer

		size, _ := io.Copy(io.Discard, io.TeeReader(content, &buffer))

		results = append(results, &File{
			Name: filename,
			Size: size,
		})
	}
}
