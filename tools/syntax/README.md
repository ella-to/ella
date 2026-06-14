# Ella for VS Code

Language support for `.ella` schema files:

- **Syntax highlighting** + the "Ella Vivid" color theme.
- **Language features** powered by the `ella lsp` language server:
  - Diagnostics (parse + validation errors, live as you type)
  - Go-to-definition (jump to a model/enum/service/error/const)
  - Hover (declaration preview, built-in type docs)
  - Document outline / symbols
  - Completion (keywords, built-in types, declared models & enums)
  - Find references
  - Formatting (`ella fmt` rules, via "Format Document")

## Requirements

The language features shell out to the `ella` binary, so it must be installed
and on your `PATH`:

```bash
go install ella.to/ella@latest
```

If `ella` is not on your `PATH`, set an absolute path in your settings:

```jsonc
// settings.json
{
  "ella.server.path": "/absolute/path/to/ella"
}
```

## Installation

Install the packaged extension:

```bash
code --install-extension ella-syntax-0.1.0.vsix --force
```

Then run `Developer: Reload Window` from the command palette (cmd/ctrl+shift+p).

## Settings

| Setting             | Default   | Description                                                        |
| ------------------- | --------- | ------------------------------------------------------------------ |
| `ella.server.path`  | `"ella"`  | Path to the `ella` executable that provides the language server.   |
| `ella.server.args`  | `["lsp"]` | Arguments used to start the server.                                |
| `ella.trace.server` | `"off"`   | `off` / `messages` / `verbose` — trace JSON-RPC traffic for debug. |

Command: **Ella: Restart Language Server** (palette) restarts the server, e.g.
after rebuilding the `ella` binary.

## Development

The extension is plain JavaScript bundled with esbuild. After changing
`src/extension.js`, the grammar, theme, or `language-configuration.json`:

```bash
npm install          # first time only
npm run build        # bundle src/extension.js -> dist/extension.js
npm run package      # build + produce ella-syntax-<version>.vsix
```

Then install the rebuilt `.vsix` as described above and reload the window.

To iterate quickly, open this folder in VS Code and press `F5` to launch an
Extension Development Host (run `npm run watch` alongside to rebuild on save).
