# Porch HCL Language Support Testing Guide

This guide walks you through testing the Porch HCL Language Server and VSCode extension.

## Prerequisites

1. **Build the tools:**
   ```bash
   cd porch/tools
   make build-lsp
   make build-extension
   ```

2. **Install VSCode extension:**
   ```bash
   cd vscode-extension
   code --install-extension porch-hcl-0.1.0.vsix
   ```
   Or install in development mode by opening the `vscode-extension` folder in VSCode and pressing `F5`.

## Testing the Language Server Standalone

1. **Test basic functionality:**
   ```bash
   cd /Volumes/code/porch/tools/language-server
   echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"processId":null,"rootPath":"/Volumes/code/porch/tools","rootUri":"file:///Volumes/code/porch/tools","capabilities":{}}}' | ./porch-lsp
   ```

2. **Test with a real HCL file:**
   The language server can process Porch HCL files and provide:
   - Syntax validation and diagnostics
   - Completion suggestions for blocks and HCL functions
   - Hover information for functions and blocks

## Testing the Updated Language Server Features

### 1. Syntax Highlighting Test

Open `sample.porch.hcl` or `comprehensive-test.porch.hcl` in VSCode. You should see:
- ✅ Keywords highlighted (`variable`, `locals`, `workflow`, `command`)
- ✅ Strings highlighted in green/yellow
- ✅ Comments highlighted in gray/green
- ✅ Block names and identifiers properly colored

### 2. Enhanced Snippets Test

In a `.porch.hcl` file, type:
- `var` + Tab → Should create a variable block (now with higher priority)
- `workflow` + Tab → Should create a workflow block (now with higher priority)
- `command` + Tab → Should create a command block (now with higher priority)
- `locals` + Tab → Should create a locals block (now with higher priority)

**Note**: Snippets now have higher priority in completion suggestions!

### 3. Enhanced Completion Test

1. **HCL Function Completion** - Type in an attribute value context:
   ```hcl
   locals {
     test = time  # <- Type 'time' and press Cmd+Space
   }
   ```
   You should see HCL functions like `timestamp()`, `timeadd()` with documentation.

2. **Attribute Completion** - Inside a block, press Cmd+Space to see relevant attributes:
   ```hcl
   variable "test" {
     # <- Press Cmd+Space here to see attributes like 'description', 'type', 'default'
   }
   ```

3. **Context-Aware Completion** - The language server now provides different completions based on context:
   - At top level: Block types (variable, workflow, etc.)
   - Inside blocks: Attributes (name, description, etc.)
   - In expressions: HCL functions

### **🎯 CRITICAL TEST - Attribute Completion Priority**

**Expected Behavior**: When you press Cmd+Space inside a workflow block (where you can add attributes), `description` should be one of the TOP suggestions, NOT buried under generic text completions like `abc`, `168h`, etc.

**Test Steps**:
1. Open `/Volumes/code/porch/tools/language-server-test.porch.hcl`
2. Position cursor after `name = "Display Name"` in the workflow block
3. Press Enter to create a new line with proper indentation
4. Press `Cmd+Space` (or `Ctrl+Space` on Windows/Linux)

**✅ EXPECTED**: You should see attributes like:
- `description` (TOP priority)
- `working_directory`
- `command_line`
- `type`
- etc.

**❌ UNEXPECTED**: Generic text completions (`abc`, `168h`, `24h`) should NOT be at the top

---

### 4. Enhanced Hover Test

Hover over different elements to see rich documentation:

1. **HCL Functions** - Hover over functions like `timestamp()`, `base64encode()`, `upper()` to see descriptions
2. **Porch Keywords** - Hover over keywords like `workflow`, `command`, `variable` to see explanations
3. **Attributes** - Hover over attributes like `name`, `description`, `working_directory` for context

### 5. Advanced Features Test

1. **Trigger Characters** - Completion should trigger automatically when typing:
   - `=` (for attribute values)
   - `(` (for function calls)
   - ` ` (space in certain contexts)

2. **Rich Hover Information** - Hover content now uses Markdown formatting for better readability

3. **Function Documentation** - All major HCL functions now have proper documentation:
   - String functions: `upper`, `lower`, `join`, `split`, etc.
   - Encoding: `base64encode`, `jsonencode`, etc.
   - Time: `timestamp`, `formatdate`, `timeadd`
   - Crypto: `md5`, `sha256`, `uuid`
   - Collections: `length`, `keys`, `merge`, etc.
   - Math: `max`, `min`, `abs`, `sqrt`
   - Type: `type`, `can`, `try`, `tonumber`

### 4. Complete Integration Test

1. **Create a new Porch HCL file:**
   ```bash
   touch test.porch.hcl
   code test.porch.hcl
   ```

2. **Test the complete workflow:**
   ```hcl
   # Start typing this and test features at each step
   variable "test" {
     description = "Test variable"
     type        = string
     default     = "hello"
   }

   locals {
     # Test HCL function completion - type "time" and see if you get suggestions
     current_time = timestamp()
     # Test more functions from hclfuncs
     encoded = base64encode("hello world")
     json_data = jsonencode({
       name = "test"
       time = timestamp()
     })
   }

   workflow "test_workflow" {
     name = "Test Workflow"

     command {
       type = "shell"
       name = "Test Command"
       # Test string interpolation highlighting
       command_line = "echo ${local.current_time}"
     }
   }
   ```

3. **Verify each feature works:**
   - ✅ Syntax highlighting for all elements
   - ✅ Snippets expand correctly
   - ✅ No syntax errors reported for valid HCL
   - ✅ Completion suggestions for HCL functions
   - ✅ Hover information for functions
   - ✅ Language server starts automatically

## Troubleshooting

### Language Server Not Starting
- Check VSCode output panel → "Porch HCL Language Server"
- Verify the language server binary exists and is executable
- Check file associations in VSCode settings

### No Syntax Highlighting
- Verify the extension is installed and enabled
- Check that file has `.porch.hcl` extension
- Restart VSCode if needed

### No Completion/Hover
- Check if language server is running (VSCode output panel)
- Verify file is recognized as Porch HCL (bottom-right of VSCode)
- Check for errors in language server logs

### Extension Installation Issues
- Build the extension: `make build-extension`
- Package it: `make package-extension`
- Install manually from the generated `.vsix` file

## Available HCL Functions

The language server provides completion and hover for all functions from `github.com/lonegunmanb/hclfuncs`:

**String Functions:** `upper`, `lower`, `title`, `trim`, `trimspace`, `split`, `join`, `replace`, `regex_replace`, `substr`, `format`, `formatlist`

**Encoding Functions:** `base64encode`, `base64decode`, `jsonencode`, `jsondecode`, `urlencode`, `urldecode`

**Time Functions:** `timestamp`, `formatdate`, `timeadd`

**Crypto Functions:** `md5`, `sha1`, `sha256`, `sha512`, `uuid`, `uuidv5`

**Collection Functions:** `length`, `keys`, `values`, `lookup`, `element`, `index`, `contains`, `distinct`, `sort`, `reverse`, `merge`, `concat`, `flatten`, `range`, `zipmap`

**Math Functions:** `abs`, `ceil`, `floor`, `log`, `max`, `min`, `pow`, `signum`, `sqrt`

**Type Functions:** `type`, `can`, `try`, `tonumber`, `tostring`, `tobool`, `tolist`, `toset`, `tomap`

**File Functions:** `file`, `fileexists`, `filebase64`, `dirname`, `basename`, `pathexpand`

**Network Functions:** `cidrhost`, `cidrnetmask`, `cidrsubnet`

Try using any of these in your Porch HCL files to test completion and hover functionality!
