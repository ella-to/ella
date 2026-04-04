# Ella Syntax For VS Code

## Installation

run the following to install the syntax highlighting 

```bash
code --install-extension ella-syntax-0.0.1.vsix --force
```

then using cmd+shift+p and select `Developer: Reload Window`

## Development and configuration

if you plan to update the logic, color and theme, after any changes, run the following

```bash
bunx --bun @vscode/vsce package
```

and then install it as describe above.
