// VS Code language client for the Ella language server.
//
// This activates whenever an .ella file is opened and launches the `ella lsp`
// server over stdio. The path to the `ella` executable and its arguments are
// configurable so users can point at a custom build.

const { workspace, window, commands } = require("vscode");
const { LanguageClient, TransportKind } = require("vscode-languageclient/node");

/** @type {import("vscode-languageclient/node").LanguageClient | undefined} */
let client;

function startClient() {
  const config = workspace.getConfiguration("ella");
  const command = config.get("server.path", "ella");
  const args = config.get("server.args", ["lsp"]);

  /** @type {import("vscode-languageclient/node").ServerOptions} */
  const serverOptions = {
    run: { command, args, transport: TransportKind.stdio },
    debug: { command, args, transport: TransportKind.stdio },
  };

  /** @type {import("vscode-languageclient/node").LanguageClientOptions} */
  const clientOptions = {
    documentSelector: [{ scheme: "file", language: "ella" }],
    synchronize: {
      // Notify the server when any .ella file changes on disk so cross-file
      // analysis stays current.
      fileEvents: workspace.createFileSystemWatcher("**/*.ella"),
    },
  };

  // The client id "ella" makes the client honor the `ella.trace.server`
  // setting automatically.
  client = new LanguageClient(
    "ella",
    "Ella Language Server",
    serverOptions,
    clientOptions
  );

  client.start().catch((err) => {
    window.showErrorMessage(
      `Ella language server failed to start (command: "${command} ${args.join(
        " "
      )}"). Is the ella binary on your PATH? ${err}`
    );
  });
}

function activate(context) {
  startClient();

  context.subscriptions.push(
    commands.registerCommand("ella.restartServer", async () => {
      if (client) {
        await client.stop();
      }
      startClient();
      window.showInformationMessage("Ella language server restarted.");
    })
  );
}

async function deactivate() {
  if (client) {
    await client.stop();
    client = undefined;
  }
}

module.exports = { activate, deactivate };
