install:
	go install ella.to/ella

regenrate: install
	ella gen http ./e2e/http/http.gen.go ./e2e/http/http.ella
	ella gen stream ./e2e/stream/stream.gen.go ./e2e/stream/stream.ella
	ella gen upload ./e2e/upload/upload.gen.go ./e2e/upload/upload.ella
	ella gen rpc ./e2e/rpc/rpc.gen.go ./e2e/rpc/rpc.ella
	ella gen http ./e2e/http_async_stream/http_async_stream.gen.go ./e2e/http_async_stream/http_async_stream.ella
	ella gen download ./e2e/download/download.gen.go ./e2e/download/download.ella

	ella gen-models models ./e2e/split/models/model.gen.go ./e2e/split/*.ella
	ella gen-services services ./e2e/split/services/services.gen.go models "ella.to/ella/e2e/split/models" ./e2e/split/*.ella	

run-e2e: regenrate
	go mod tidy
	go test ./e2e/http/... -v
	go test ./e2e/stream/... -v
	go test ./e2e/upload/... -v
	go test ./e2e/rpc/... -v
	go test ./e2e/http_async_stream/... -v
	go test ./e2e/download/... -v