# Example

```
service RpcMathService {
  Sub(a: int64, b: int64) => (int64)
}

service HttpMathService {
  Events(id: string) => (stream Event) {
    PingInterval = 10s
  }

  Download(id: string) => (stream []byte) {


  }

  Upload(id: string, files: stream []byte) => ([]string)
}
```

## Simple Request/Response

- Request

params must be named argument

```json
{
  "method": "RpcMathService.Sub",
  "params": { "a": 10, "b": 2 },
  "id": "req_1234"
}
```

- Response (Success)

```json
{ "results": [8], "id": "req_1234" }
```

- Response (Failure)

```json
{
  "error": { "code": -32700, "message": "Parse error", "data": {} },
  "id": "req_1234"
}
```

## Stream (SSE)

- Request

```json
{ "method": "Notification.Events", "type": "stream:text", "params": {} }
```

- Response (Success)

```json
id: 1
type: data
data: "[]"


: ping


id: 2
type: data
data: "[]"


id: 3
type: error
data: "{ \"code\": -32700, \"message\": \"Parse error\", \"data\": {} }"


id: 4
type: done
data:
```

## Stream Binary

- Request

```json
{
  "method": "Assets.Download",
  "type": "stream:binary",
  "params": { "id": "asset_1234" }
}
```

- Response

```
binary data
```

## File Upload

- Request

```
Form.data("method", "Assets.Upload")
Form.data("params", "{\"filename\":\"avatar.jpg\"}")
Form.data("id", "req_1234")
Form.data("files", blob)
```

-- Response

```json
{ "results": [8], "id": "req_1234" }
```
