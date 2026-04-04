```
███████╗██╗░░░░░██╗░░░░░░█████╗░
██╔════╝██║░░░░░██║░░░░░██╔══██╗
█████╗░░██║░░░░░██║░░░░░███████║
██╔══╝░░██║░░░░░██║░░░░░██╔══██║
███████╗███████╗███████╗██║░░██║
╚══════╝╚══════╝╚══════╝╚═╝░░╚═╝
```
<div align="center">

[![Go Reference](https://pkg.go.dev/badge/ella.to/ella.svg)](https://pkg.go.dev/ella.to/ella)
[![Go Report Card](https://goreportcard.com/badge/ella.to/ella)](https://goreportcard.com/report/ella.to/ella)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**ella** is a schema compiler that generates Go, TypeScript, and WebAssembly code from a single, human-readable definition language.

</div>

## What Is It

Ella takes `.ella` schema files and generates type-safe client/server code for Go, TypeScript clients and type definitions, and WASM bindings. Think of it like gRPC or Protocol Buffers, but with a much simpler syntax that reads like pseudocode.

You define your models, enums, services, and errors in one place, and ella generates everything you need to call those services from Go backends, TypeScript frontends, or browser WASM modules.

## Installation

```bash
go install ella.to/ella@latest
```

## Commands

```bash
# Format .ella files in place
ella fmt "./schema/src/*.ella"

# Generate Go code
ella gen schema "./schema/output.gen.go" "./schema/src/*.ella"

# Generate Go code with WASM extensions (for browser clients)
ella gen schema --allow-ext "./schema/output.gen_js.go" "./schema/src/*.ella"

# Generate TypeScript type definitions (.d.ts)
ella gen schema "./web/src/schema.d.ts" "./schema/src/*.ella"

# Generate TypeScript runtime client (.ts)
ella gen schema "./web/src/schema.ts" "./schema/src/*.ella"

# Print AST for debugging
ella gen schema --debug "./schema/output.gen.go" "./schema/src/*.ella"

# Print version
ella ver
```

The output format is determined by the file extension of the output path:
- `.go` — Go structs, interfaces, JSON-RPC client/server code
- `_js.go` — Go WASM bindings (use `--allow-ext` flag)
- `.d.ts` — TypeScript declarations (types/interfaces for WASM usage)
- `.ts` — TypeScript runtime client (fetch JSON-RPC helper + `create<Service>` factories + models/enums)

## Schema Language

### Constants

Constants define fixed values. They're useful for event topic names, configuration thresholds, or anything you want shared across generated code.

```ella
const TopicUserCreated = "app.user.created"
const TopicUserDeleted = "app.user.deleted"
const MaxUploadSize = 100mb
const RequestTimeout = 30s
```

Size units: `kb`, `mb`, `gb`, `tb`, `eb`
Time units: `ms`, `s`, `m`, `h`

### Enums

Enums default to integer values starting at 0. You can also give them explicit string values.

```ella
# Integer enum (values: 0, 1, 2)
enum UserStatus {
    Pending
    Active
    Disabled
}

# String enum
enum DeviceStatus {
    Init = "init"
    Online = "online"
    Offline = "offline"
}
```

### Models

Models define data structures. Fields have a name and a type, separated by a colon.

```ella
model User {
    Id: string
    Email: string
    Name: string
    Status: UserStatus
    Created: timestamp
    Attributes: map<string, any>
}
```

Models can extend other models to reuse fields:

```ella
model Device {
    ...User
    MachineId: string
    DeviceStatus: DeviceStatus
}
```

### Types

| Type | Description |
|------|-------------|
| `string` | Text |
| `bool` | Boolean |
| `byte` | Single byte |
| `int8`, `int16`, `int32`, `int64` | Signed integers |
| `uint8`, `uint16`, `uint32`, `uint64` | Unsigned integers |
| `float32`, `float64` | Floating point |
| `timestamp` | Unix timestamp |
| `any` | Untyped (maps to `interface{}` / `any`) |
| `[]Type` | Array of Type |
| `map<K, V>` | Map with key type K and value type V |

### Template Strings

String constants with `{{ }}` placeholders generate functions instead of plain values:

```ella
const TopicUserStatus = "app.user.{{userId}}.status"
```

This generates a function that takes `userId` as a parameter and returns the interpolated string.

### Services

Services define RPC methods. Each method lists its request parameters and response fields.

```ella
service UserService {
    Create (email: string, name: string) => (user: User)
    GetById (id: string) => (user: User)
    UpdateStatus (id: string, status: UserStatus) => (user: User)
    Delete (id: string)
    List () => (users: []User)
}
```

Methods without a return clause produce no response body.

### Errors

Named errors with optional HTTP status codes:

```ella
error ErrUserNotFound { Msg = "user not found" }
error ErrEmailConflict { Code = 409 Msg = "email already exists" }
```

These generate typed error values in Go that work with `errors.Is()`.

## Generated Code

### Go

The Go output includes:
- Struct types with `json:"camelCase"` tags for all models
- Enum types with `String()`, `MarshalJSON()`, and `UnmarshalJSON()` methods
- A service interface (e.g. `UserServiceHandler`) with `context.Context` on every method
- A server constructor that wires up JSON-RPC method routing
- A client constructor that implements the same interface via JSON-RPC calls
- Typed error variables

### TypeScript

For `.d.ts` output:
- Interface definitions for all models
- Enum types as string union types
- Service interfaces with `Promise<T>` return types
- Support for `AbortSignal`, caching, and timeout options

For `.ts` output:
- `createFetchJsonRpc(host, options)` helper compatible with `ella.to/jsonrpc` request/response format
- `create<Service>(conn)` factory functions that return async service clients
- Runtime constants and enum values
- `EllaRPCError` plus typed error guards for schema-defined errors

Example runtime client usage:

```ts
import {
    createFetchJsonRpc,
    createUserService,
    isErrUserNotFound,
} from "./schema"

const conn = createFetchJsonRpc("https://api.example.com/rpc")
const users = createUserService(conn)

try {
    const user = await users.getById("123")
    console.log(user)
} catch (err) {
    if (isErrUserNotFound(err)) {
        console.error("user not found")
    } else {
        throw err
    }
}
```

### WASM

The WASM output (with `--allow-ext`) generates Go code that:
- Creates a JavaScript-callable API object
- Wraps each service method as an async function
- Handles request/response serialization through the WASM bridge
- Supports client-side caching with configurable TTL

## Formatting

`ella fmt` normalizes your schema files by sorting declarations in a consistent order: constants, then enums, then models, then services, then errors. This keeps things tidy across a team.

```bash
ella fmt "./schema/src/*.ella"
```

## Syntax Highlighting

Ella includes a VS Code syntax extension in `tools/syntax`.

Install from this repository:

```bash
cd tools/syntax
code --install-extension ella-syntax-0.0.1.vsix --force
```

After installation, run `Developer: Reload Window` from the VS Code command palette.

## License

MIT — see [LICENSE](LICENSE) for details.