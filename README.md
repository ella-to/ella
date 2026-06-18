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
go install ella.to/ella@v0.3.1
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

# Start the language server (for editor integration, speaks LSP over stdio)
ella lsp

# Print version
ella ver
```

The output format is determined by the file extension of the output path:
- `.go` — Go structs, interfaces, JSON-RPC client/server code
- `_js.go` — Go WASM bindings (use `--allow-ext` flag)
- `.d.ts` — TypeScript declarations (types/interfaces for WASM usage)
- `.ts` — TypeScript runtime client (fetch JSON-RPC helper + `create<Service>` factories + models/enums)

## Schema Language

### Modules

Every `.ella` file **must** begin with a `module` declaration. It is the first
thing in the file — only comments may appear before it, never a `const`, `enum`,
`model`, `service`, or `error`.

```ella
module billing

const Currency = "USD"

model Invoice {
    Id: string
    Total: int64
}
```

A module is a **namespace**. Files that share the same module name are compiled
together as one namespace, so a model, enum, const, or service can be split
across several files. Declarations in *different* modules are fully isolated:
the same name may be reused across modules without conflict, and a type
reference resolves only within its own module.

```ella
# users.ella
module users

model User { Id: string }
```

```ella
# orders.ella
module orders

# This "User" does not collide with the one in module `users`.
model User { OrderId: string }
```

This is what keeps tooling (and the language server in particular) from
reporting false "duplicate declaration" errors when two unrelated projects in
the same workspace happen to use the same names — give them different module
names and they stay independent.

> The schema fragments in the sections below omit the `module` line for brevity,
> but a real file always starts with one.

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

#### Optional Arguments

Method arguments can be marked optional with `?`, using the same syntax as optional model fields. Optional arguments must come after all required arguments:

```ella
service UserService {
    List (query: string, limit?: int64, tags?: []string) => (users: []User)
}

service GroupService {
    List (limit?: int64) => (groups: []Group)
}
```

In **Go**, optional arguments become type-safe functional options. A single `With<Name>` function is generated per unique argument name and is shared by every method that declares an optional argument with that name — `WithLimit` below works for both `UserService.List` and `GroupService.List`:

```go
users, err := userClient.List(ctx, "active", WithLimit(50), WithTags([]string{"admin"}))
groups, err := groupClient.List(ctx, WithLimit(10))
```

Because of this sharing, optional arguments with the same name must use the same type everywhere; the compiler reports an error otherwise.

Server implementations receive the same variadic options and read them through the generated per-method options struct, where unset arguments are `nil`:

```go
func (s *server) List(ctx context.Context, query string, opts ...UserServiceListOption) ([]*User, error) {
    options := NewUserServiceListOptions(opts...)
    if options.Limit != nil {
        // limit was provided
    }
    ...
}
```

In **TypeScript** (both `.d.ts` and `.ts` outputs), optional arguments are grouped into an optional object parameter placed between the required arguments and the call options:

```ts
const users = await userService.list("active", { limit: 50, tags: ["admin"] })
const groups = await groupService.list({ limit: 10 })
```

On the wire, unset optional arguments are omitted from the JSON-RPC params entirely, so Go and TypeScript clients and servers are fully interoperable.

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

## Editor Support

### Language Server

Ella ships a built-in language server. Start it with:

```bash
ella lsp
```

It speaks the [Language Server Protocol](https://microsoft.github.io/language-server-protocol/)
over **stdio** and provides:

| Feature                  | LSP method                      |
| ------------------------ | ------------------------------- |
| Diagnostics (errors)     | `textDocument/publishDiagnostics` |
| Go-to-definition         | `textDocument/definition`       |
| Hover                    | `textDocument/hover`            |
| Document outline/symbols | `textDocument/documentSymbol`   |
| Completion               | `textDocument/completion`       |
| Find references          | `textDocument/references`       |
| Formatting               | `textDocument/formatting`       |

Diagnostics combine scanner/parser errors with full schema validation, and the
server is **workspace-aware**: it indexes every `.ella` file under the workspace
root, so go-to-definition and "unknown type" checks work across files exactly
like `ella gen` does. Resolution is **module-scoped** — symbols, references, and
duplicate-name checks stay within a module, so two projects that share names but
declare different modules never interfere with each other.

The server has no runtime configuration — point your editor's LSP client at the
`ella lsp` command for documents with language id `ella` (file extension
`.ella`). It writes only protocol frames to stdout; logs go to stderr (or to a
file with `ella lsp --log /path/to/ella-lsp.log`). `--stdio` is accepted for
client compatibility but stdio is the only transport.

### VS Code

The extension in `tools/syntax` provides syntax highlighting **and** wires up
the language server automatically.

```bash
cd tools/syntax
code --install-extension ella-syntax-0.1.0.vsix --force
```

Run `Developer: Reload Window` afterward. The extension launches `ella lsp` for
any `.ella` file; ensure `ella` is on your `PATH` or set `ella.server.path` in
your settings. See [`tools/syntax/README.md`](tools/syntax/README.md) for build
and configuration details.

### Neovim (built-in LSP)

```lua
vim.filetype.add({ extension = { ella = "ella" } })

vim.api.nvim_create_autocmd("FileType", {
  pattern = "ella",
  callback = function(args)
    vim.lsp.start({
      name = "ella-lsp",
      cmd = { "ella", "lsp" },
      root_dir = vim.fs.root(args.buf, { ".git", "go.mod" }) or vim.fn.getcwd(),
    })
  end,
})
```

### Helix

Add to `~/.config/helix/languages.toml`:

```toml
[language-server.ella-lsp]
command = "ella"
args = ["lsp"]

[[language]]
name = "ella"
scope = "source.ella"
file-types = ["ella"]
comment-token = "#"
language-servers = ["ella-lsp"]
roots = [".git", "go.mod"]
```

### Other editors

Any LSP-capable editor works: configure a server whose command is `ella lsp`,
associate it with the `.ella` extension / `ella` language id, and (optionally)
set the workspace root so cross-file features work. On `initialize` the server
reads `workspaceFolders` / `rootUri` and scans them for `.ella` files.

## License

MIT — see [LICENSE](LICENSE) for details.