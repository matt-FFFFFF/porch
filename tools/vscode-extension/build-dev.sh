#!/bin/bash

# Development build script for Porch HCL VSCode Extension

set -e

echo "🔨 Building Porch HCL Language Server..."
cd ../
make cross-compile-lsp

echo "📦 Building VSCode Extension (development mode)..."
cd vscode-extension
npm run compile

echo "✅ Development build complete!"
echo "💡 To package for production, run: npm run package"
echo "📋 To install locally, run: code --install-extension porch-hcl-0.1.0.vsix --force"
