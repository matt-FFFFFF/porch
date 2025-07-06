import * as fs from 'fs';
import * as path from 'path';
import * as vscode from 'vscode';
import { LanguageClient, LanguageClientOptions, ServerOptions, TransportKind } from 'vscode-languageclient/node';

let client: LanguageClient;

export function activate(context: vscode.ExtensionContext) {
  try {
    console.log('Porch HCL extension is being activated');
    console.log('Extension path:', context.extensionPath);
    console.log('Platform:', process.platform, 'Architecture:', process.arch);

    // Get the language server executable path
    const config = vscode.workspace.getConfiguration('porch-hcl');
    let serverPath = config.get<string>('server.path', '');

    // If no custom path is configured, use the bundled binary for current platform
    if (!serverPath) {
      try {
        serverPath = getBundledLanguageServerPath(context);
      } catch (error) {
        console.error('Error getting bundled language server path:', error);
        vscode.window.showErrorMessage(
          `Failed to detect language server binary: ${error instanceof Error ? error.message : error}. ` +
          `Platform: ${process.platform}-${process.arch}`
        );
        // Continue anyway to register commands
        registerCommands(context);
        console.log('Porch HCL extension activated (without language server)');
        return;
      }
    }

    console.log('Detected server path:', serverPath);

    const serverArgs = config.get<string[]>('server.args', []);

    // Verify the server path exists
    if (!fs.existsSync(serverPath)) {
      const errorMsg = `Language server not found at: ${serverPath}`;
      console.error(errorMsg);
      vscode.window.showErrorMessage(
        `Failed to start Porch HCL Language Server. ${errorMsg}. ` +
        `Please make sure the extension is properly installed.`
      );
      // Continue anyway to register commands
      registerCommands(context);
      console.log('Porch HCL extension activated (without language server)');
      return;
    }

    console.log(`Using language server at: ${serverPath}`);
    console.log('Server args:', serverArgs);

    // Define the server options
    const serverOptions: ServerOptions = {
      command: serverPath,
      args: serverArgs,
      transport: TransportKind.stdio
    };

    // Define the client options
    const clientOptions: LanguageClientOptions = {
      // Register the server for Porch HCL documents
      documentSelector: [
        {
          scheme: 'file',
          language: 'porch-hcl'
        }
      ],
      synchronize: {
        // Notify the server about file changes to '.porch.hcl' files
        fileEvents: vscode.workspace.createFileSystemWatcher('**/*.porch.hcl')
      },
      // Output channel for the language server
      outputChannelName: 'Porch HCL Language Server',
      // Trace communication between client and server
      traceOutputChannel: vscode.window.createOutputChannel('Porch HCL Language Server Trace'),
      // Completion settings to prioritize our suggestions
      initializationOptions: {
        completionSettings: {
          enabled: true,
          snippetSupport: true
        }
      }
    };

    // Create the language client
    client = new LanguageClient(
      'porch-hcl-lsp',
      'Porch HCL Language Server',
      serverOptions,
      clientOptions
    );

    console.log(`Starting language server at: ${serverPath}`);

    // Start the client. This will also launch the server
    client.start().then(() => {
      console.log('Language server started successfully');
      vscode.window.showInformationMessage('Porch HCL Language Server is running');
    }).catch((error) => {
      console.error('Failed to start language server:', error);
      vscode.window.showErrorMessage(
        `Failed to start Porch HCL Language Server: ${error.message}. ` +
        `Make sure the language server is built: cd tools && make build-lsp. ` +
        `Server path: ${serverPath}`
      );
    });

    // Add client to subscriptions for cleanup
    context.subscriptions.push({
      dispose: () => {
        if (client) {
          client.stop();
        }
      }
    });

    // Register commands
    registerCommands(context);

    console.log('Porch HCL extension activated successfully');
  } catch (error) {
    console.error('Failed to activate Porch HCL extension:', error);
    vscode.window.showErrorMessage(
      `Failed to activate Porch HCL extension: ${error instanceof Error ? error.message : error}`
    );
    // Still register commands so the extension can be used manually
    try {
      registerCommands(context);
    } catch (commandError) {
      console.error('Failed to register commands:', commandError);
    }
  }
}

export function deactivate(): Thenable<void> | undefined {
  if (!client) {
    return undefined;
  }
  return client.stop();
}

function registerCommands(context: vscode.ExtensionContext) {
  // Command to restart the language server
  const restartServerCommand = vscode.commands.registerCommand('porch-hcl.restartServer', () => {
    if (client) {
      client.stop().then(() => {
        client.start();
        vscode.window.showInformationMessage('Porch HCL Language Server restarted');
      });
    }
  });

  // Command to manually trigger completion for debugging
  const triggerCompletionCommand = vscode.commands.registerCommand('porch-hcl.triggerCompletion', () => {
    vscode.commands.executeCommand('editor.action.triggerSuggest');
  });

  // Command to show server status
  const showServerStatusCommand = vscode.commands.registerCommand('porch-hcl.showServerStatus', () => {
    if (client) {
      const state = client.state;
      let status: string;
      switch (state) {
        case 1: // Starting
          status = 'Starting';
          break;
        case 2: // Running
          status = 'Running';
          break;
        case 3: // Stopped
          status = 'Stopped';
          break;
        default:
          status = 'Unknown';
      }
      vscode.window.showInformationMessage(`Porch HCL Language Server Status: ${status}`);
    } else {
      vscode.window.showWarningMessage('Porch HCL Language Server is not available');
    }
  });

  // Command to create a new Porch HCL file
  const createPorchFileCommand = vscode.commands.registerCommand('porch-hcl.createFile', async () => {
    const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
    if (!workspaceFolder) {
      vscode.window.showErrorMessage('No workspace folder is open');
      return;
    }

    const fileName = await vscode.window.showInputBox({
      prompt: 'Enter the name for the new Porch HCL file',
      value: 'workflow.porch.hcl',
      validateInput: (value: string) => {
        if (!value.endsWith('.porch.hcl')) {
          return 'File name must end with .porch.hcl';
        }
        return null;
      }
    });

    if (fileName) {
      const filePath = path.join(workspaceFolder.uri.fsPath, fileName);
      const fileUri = vscode.Uri.file(filePath);

      const template = `# Porch HCL Configuration
# This file defines workflows for the Porch orchestration tool

workflow "example" {
  name        = "Example Workflow"
  description = "An example workflow to get you started"

  command {
    type = "shell"
    name = "Hello World"
    command_line = "echo 'Hello from Porch!'"
  }
}
`;

      try {
        await vscode.workspace.fs.writeFile(fileUri, Buffer.from(template, 'utf8'));
        const document = await vscode.workspace.openTextDocument(fileUri);
        await vscode.window.showTextDocument(document);
      } catch (error) {
        vscode.window.showErrorMessage(`Failed to create file: ${error}`);
      }
    }
  });

  context.subscriptions.push(restartServerCommand, triggerCompletionCommand, showServerStatusCommand, createPorchFileCommand);
}

// Status bar item to show language server status
let statusBarItem: vscode.StatusBarItem;

function createStatusBarItem() {
  statusBarItem = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Right, 100);
  statusBarItem.command = 'porch-hcl.showServerStatus';
  statusBarItem.text = '$(server-process) Porch LSP';
  statusBarItem.tooltip = 'Click to show Porch HCL Language Server status';

  // Show status bar item only when a Porch HCL file is open
  vscode.window.onDidChangeActiveTextEditor((editor: vscode.TextEditor | undefined) => {
    if (editor && editor.document.languageId === 'porch-hcl') {
      statusBarItem.show();
    } else {
      statusBarItem.hide();
    }
  });

  return statusBarItem;
}

// Function to get the correct bundled language server binary for the current platform
function getBundledLanguageServerPath(context: vscode.ExtensionContext): string {
  const platform = process.platform;
  const arch = process.arch;

  let binaryName: string;

  // Determine the correct binary name based on platform and architecture
  if (platform === 'darwin') {
    if (arch === 'arm64') {
      binaryName = 'porch-lsp-darwin-arm64';
    } else {
      binaryName = 'porch-lsp-darwin-amd64';
    }
  } else if (platform === 'linux') {
    if (arch === 'arm64') {
      binaryName = 'porch-lsp-linux-arm64';
    } else {
      binaryName = 'porch-lsp-linux-amd64';
    }
  } else if (platform === 'win32') {
    if (arch === 'arm64') {
      binaryName = 'porch-lsp-windows-arm64.exe';
    } else {
      binaryName = 'porch-lsp-windows-amd64.exe';
    }
  } else {
    throw new Error(`Unsupported platform: ${platform}-${arch}`);
  }

  return path.join(context.extensionPath, 'bin', binaryName);
}
