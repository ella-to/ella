```
███████╗██╗░░░░░██╗░░░░░░█████╗░
██╔════╝██║░░░░░██║░░░░░██╔══██╗
█████╗░░██║░░░░░██║░░░░░███████║
██╔══╝░░██║░░░░░██║░░░░░██╔══██║
███████╗███████╗███████╗██║░░██║
╚══════╝╚══════╝╚══════╝╚═╝░░╚═╝ v0.1.0
```

Ella, is yet another compiler to produce Go and Typescript code based on simple and easy-to-read schema IDL. There are many tools like gRPC, Twirp or event WebRPC to generate codes, but this little compiler is designed based on my views of 12+ years of developing backends and APIs. I wanted to simplify the tooling and produce almost perfect optimized, handcrafted code that can be read and understood.

Ella's schema went through several iterations to make it easier for extension and backward compatibility in future releases.

> **NOTE:**
>
> Ella's code generated has been used in couple of production projects, it has some designs that might not fit your needs, but I think it might solve a large number of projects. Also Ella's only emit `Go`, as a server and client, and `Typescript` as only client. This is intentinal as it serves my needs. However it can be easily extetended to produce other languages code by traversing the generated AST. For getting some examples about that, please refer to `generate-golang.go` and `generate-typescript.go`

# Installation

to install Ella's compiler, simply use the go install command

```bash
go install ella.to/ella@latest
```

# Usage

Simplicity applies to the CLI command as well, it looks for all files that need to be compiled and outputs the result to the designated file. The extension of the output file tells the compiler whether you want to produce the typescript or golang code. That's pretty much of it.

For example, the following command, will generate `api.gen.go` in `/api` folder with the package name `api` and will read all the ella files inside `./schema` folder.

```bash
ella gen api /api/api.gen.go ./schema/*.ella
```

Also, we can format the schema as well to have a consistent look by running the following command

```bash
ella fmt ./schema/*.ella
```

The full CLI documentation can be accessed by running Ella command without any arguments

```
███████╗██╗░░░░░██╗░░░░░░█████╗░
██╔════╝██║░░░░░██║░░░░░██╔══██╗
█████╗░░██║░░░░░██║░░░░░███████║
██╔══╝░░██║░░░░░██║░░░░░██╔══██║
███████╗███████╗███████╗██║░░██║
╚══════╝╚══════╝╚══════╝╚═╝░░╚═╝ v0.1.0

Usage: ella [command]

Commands:
  - fmt Format one or many files in place using glob pattern
        ella fmt <glob path>

  - gen Generate code from a folder to a file and currently
        supports .go and .ts extensions
        ella gen <pkg> <output path to file> <search glob paths...>

  - ver Print the version of ella

example:
  ella fmt ./path/to/*.ella
  ella gen rpc ./path/to/output.go ./path/to/*.ella
  ella gen rpc ./path/to/output.ts ./path/to/*.ella ./path/to/other/*.ella
```

> NOTE
> There is an experimental feature which splits const, enum and models from services. The api of cli will change and it is not recommended to use. It only supports golang and has no effect on typescript

```
ella gen-models model /model/model.gen.go ./schema/*.ella
ella gen-services api /api/api.gen.go model "example.com/pkg/model" ./schema/*.ella
```

# Schema

## Comment

comment can be created using `#`

for example

```
# this is a comment
```

## Constant

```
const <identifier> = <identifier> | <value>
```

for example

```
const A = 1
const B = 1_000_000
const C = 1.23
const D = "hello world"
const E = 'hello world'
const F = `hello
world
`
const FileSize = 10gb
const Timeout = 2s

const RefFileSize = FileSize
```

## Enum

```
enum <identifier> {
    <identifier> = <integer number>
    <identifier>
}
```

for example

```
enum UserRole {
    # _ skip the generation but keeps the order
    _ = 1
    Root
    Normal
}
```

## Model

```
model <identifer> {
    # for extending the model
    ...<model's identifer>
    <identifier>: <type> {
        <identifier> = <value> | <const identifer>
    }
}
```

## Service

```
service <identifer> {
    rpc | http <identifier> (<identifer>: <type>) => (<identifer>: <type>) {
        <identifider> = <value> | <const identifer>
    }
}
```

## Identifier

there are 2 types of identifiers, camelCase and PascalCase. Basically all the args and returns names must be camelCase (first char must be lowercase) and all other identifer must be PascalCase (first char must be uppercase)

## Custom Error

defining a custom error that can be safely used over the network. Code and HttpStatus is optional. Code has to be unique and HttpStatus should be valid http status code. If Code is not defined, the compiler will assign a unique Id. If HttpStatus is not defined, it will be default to 500

```
error <identifer> { Code = <Integer> HttpStatus = <Valid Http Status Code> Msg = "" }
```

## Type

type can be either the following list or refer to Model's identifer

```
int8, int16, int32, int64
uint8, uint16, uint32, uint64
float32, float64
string
bool
timestamp
any
file
[]<type>
map<type, type>
```

## Value

value is a const value for example

```
//
// Integer and float values
//

1
1.2

//
// String values
//

"hello world"
'hello world'
`hello world`

//
// boolean values
//

true
false

//
// duration values
//

1ns
1us
1ms
1s
1m
1h

//
// byte size values
//

1b
1kb
1mb
1gb
1tb
1pb
1eb

//
// null value
//

null
```
