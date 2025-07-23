# Configuration System

The gofer configuration system provides a flexible, hierarchical configuration management with fine-grained permissions and security controls.

## Configuration Hierarchy

Configuration files are loaded in the following order (later overrides earlier):

1. **System**: `/etc/gofer/config.json`
2. **User**: `~/.config/gofer/config.json`
3. **Project**: `.gofer/config.json` (searched up the directory tree)
4. **Local**: `.gofer/config.local.json` (not committed to version control)
5. **Environment**: Environment variables with `GOCODECLI_` prefix
6. **CLI**: Command-line arguments

## Configuration Structure

### API Configuration
```json
{
  "api": {
    "provider": "openrouter",
    "base_url": "https://openrouter.ai/api/v1",
    "api_key": "sk-...",
    "api_key_env_var": "OPENROUTER_API_KEY",
    "headers": {
      "HTTP-Referer": "https://github.com/user/project"
    },
    "timeout": 30,
    "retry": {
      "max_retries": 3,
      "initial_delay": 1,
      "max_delay": 10
    }
  }
}
```

### Permissions

The permission system supports three modes:
- `allow`: Operations are allowed by default
- `deny`: Operations are denied by default
- `prompt`: User confirmation required (default)

#### Tool Permissions
```json
{
  "permissions": {
    "default_mode": "prompt",
    "tools": {
      "allow": ["read_file", "write_file", "execute_command"],
      "deny": ["system_*", "admin_*"],
      "require_confirmation": ["execute_command", "delete_*"],
      "custom_rules": [
        {
          "name": "npm_scripts",
          "pattern": "execute_command(npm run *)",
          "action": "allow"
        }
      ]
    }
  }
}
```

#### File System Permissions
```json
{
  "permissions": {
    "filesystem": {
      "read_paths": [".", "~/documents"],
      "write_paths": ["."],
      "deny_paths": ["/etc", "/sys", "/proc"],
      "max_file_size": 10485760,
      "denied_extensions": [".exe", ".dll"],
      "sandbox_mode": false
    }
  }
}
```

#### Command Permissions
```json
{
  "permissions": {
    "commands": {
      "allowed_commands": ["git", "npm", "go"],
      "denied_commands": ["rm -rf /", "shutdown"],
      "denied_patterns": [".*\\brm\\s+-rf\\s+/.*"],
      "max_timeout": 300,
      "filter_env_vars": ["AWS_SECRET_ACCESS_KEY"]
    }
  }
}
```

### Security Configuration
```json
{
  "security": {
    "require_confirmation": true,
    "show_preview": true,
    "log_operations": true,
    "session_timeout": 1800,
    "audit_log": {
      "enabled": true,
      "path": "~/.local/share/gofer/audit.log",
      "max_size": 104857600,
      "format": "json"
    }
  }
}
```

## Usage Examples

### Creating a Default Configuration

```go
import "github.com/elee1766/gofer/src/config"

// Create default config for OpenRouter
cfg := config.DefaultOpenRouterConfig()

// Save to user config directory
discovery := config.NewDiscovery()
err := discovery.CreateDefaultConfig("user", "openrouter")
```

### Loading Configuration

```go
// Create a configuration manager
manager, err := config.NewManager()
if err != nil {
    log.Fatal(err)
}

// Get the loaded configuration
cfg := manager.GetConfig()

// Get specific configurations
apiConfig := manager.GetAPIConfig()
agentConfig := manager.GetAgentConfig()
```

### Checking Permissions

```go
// Check if a tool operation is allowed
result, err := manager.CheckToolPermission("execute_command", map[string]interface{}{
    "command": "npm test",
})

if result.RequiresConfirmation {
    // Show confirmation dialog
    fmt.Println(result.ConfirmationMessage)
}

if !result.Allowed {
    fmt.Printf("Operation denied: %s\n", result.Reason)
    return
}

// Check file permissions
result, err = manager.CheckFileWritePermission("/path/to/file.txt")
```

### Updating Configuration

```go
// Update specific values
err := manager.Update(map[string]interface{}{
    "agent": map[string]interface{}{
        "temperature": 0.5,
        "max_tokens": 8192,
    },
})

// Save changes
err = manager.Save()
```

### Configuration Discovery

```go
discovery := config.NewDiscovery()

// Find all config files
locations, err := discovery.FindConfigs()
for _, loc := range locations {
    fmt.Printf("Found config: %s (source: %s)\n", loc.Path, loc.Source)
}

// Find project config
projectConfig, err := discovery.FindProjectConfig(".")
```

## Environment Variables

Configuration values can be overridden using environment variables:

- `GOCODECLI_API_KEY`: Override API key
- `GOCODECLI_MODEL`: Override model name
- `GOCODECLI_PROVIDER`: Override provider
- `GOCODECLI_BASE_URL`: Override API base URL
- `GOCODECLI_PERMISSION_MODE`: Override default permission mode
- `GOCODECLI_SANDBOX`: Enable sandbox mode (true/false)

## Security Best Practices

1. **Never commit API keys**: Use environment variables or `.local.json` files
2. **Use sandbox mode**: Enable for untrusted environments
3. **Restrict permissions**: Start with restrictive permissions and expand as needed
4. **Enable audit logging**: Track all operations for security review
5. **Regular reviews**: Periodically review permissions and audit logs

## Examples

- `openrouter.json`: Standard OpenRouter configuration
- `restricted.json`: Highly restricted read-only configuration
- `development.json`: Permissive configuration for development

## Migration from Gofer

The new configuration system is backward compatible with existing Gofer configurations. The migration path:

1. Existing `gofer.json` files will be automatically detected
2. Configuration values are mapped to the new structure
3. New permission and security features use sensible defaults

To migrate manually:
```bash
gofer config migrate --from gofer.json --to config.json
```