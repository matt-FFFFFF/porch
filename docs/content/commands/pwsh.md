+++
title = "PowerShell Command"
weight = 2
+++

The `pwsh` command executes PowerShell scripts with full environment control and configurable exit code handling. It works on Windows, Linux, and macOS.

## Attributes

### Required

- **`type: "pwsh"`**: Identifies this as a PowerShell command
- **`name`**: Descriptive name for the command
- **`script`** OR **`script_file`**: PowerShell script content or path to script file (mutually exclusive)

### Optional

- **`working_directory`**: Directory to execute the command in
- **`env`**: Environment variables as key-value pairs
- **`runs_on_condition`**: When to run (`success`, `error`, `always`, `exit-codes`)
- **`runs_on_exit_codes`**: Specific exit codes that trigger execution
- **`success_exit_codes`**: Exit codes indicating success (defaults to `[0]`)
- **`skip_exit_codes`**: Exit codes that skip remaining commands

## Inline Script Example

```yaml
name: "PowerShell Inline"
commands:
  - type: "pwsh"
    name: "Run PowerShell Script"
    script: |
      Write-Host "Starting PowerShell script..."
      # Your PowerShell commands here
      Write-Host "PowerShell script completed."
```

## Script File Example

```yaml
name: "PowerShell File"
commands:
  - type: "pwsh"
    name: "Run PowerShell File"
    script_file: "./scripts/deploy.ps1"
    working_directory: "/app"
```

## Complete Example

```yaml
name: "Advanced PowerShell Command"
commands:
  - type: "pwsh"
    name: "Build and Deploy"
    script: |
      Write-Host "Building application..."
      dotnet build -c Release

      Write-Host "Running tests..."
      dotnet test

      if ($LASTEXITCODE -ne 0) {
        Write-Error "Tests failed"
        exit 1
      }

      Write-Host "Deployment successful"
    working_directory: "/path/to/project"
    env:
      DEPLOY_ENV: "production"
      API_KEY: "secret-key"
    success_exit_codes: [0]
    runs_on_condition: "success"
```

## Script vs Script File

Use `script` for inline PowerShell code:

```yaml
- type: "pwsh"
  name: "Inline Script"
  script: |
    Write-Host "Hello from PowerShell"
    Get-ChildItem
```

Use `script_file` to reference an external file:

```yaml
- type: "pwsh"
  name: "External Script"
  script_file: "./scripts/build.ps1"
```

**Note**: `script` and `script_file` are mutually exclusive. Use one or the other.

## Exit Code Handling

PowerShell exit codes work the same as shell commands:

```yaml
- type: "pwsh"
  name: "Flexible PowerShell"
  script: |
    # Your PowerShell code
    if ($someCondition) {
      exit 0  # Success
    } else {
      exit 1  # Failure
    }
  success_exit_codes: [0]
  skip_exit_codes: [99]
```

## Best Practices

1. **Use Write-Host for output**: Makes messages visible in Porch results
2. **Check $LASTEXITCODE**: Verify exit codes of called commands
3. **Set working directory**: Use `working_directory` attribute instead of Set-Location
4. **Use environment variables**: Access via `env` attribute, read with `$env:VAR_NAME`
5. **Handle errors**: Use `exit` to return appropriate codes

## Related

- [Shell Command](shell/) - Execute shell commands
- [Flow Control](../basics/flow-control/) - Conditional execution and error handling
