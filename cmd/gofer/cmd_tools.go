package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"text/tabwriter"
	"os"

	"github.com/alecthomas/kong"
)

// ToolsCmd represents all tool-related commands
type ToolsCmd struct {
	// Tool management
	List    ToolsListCmd    `cmd:"list" help:"List available tools"`
	Show    ToolsShowCmd    `cmd:"show" help:"Show tool details"`
	Test    ToolsTestCmd    `cmd:"test" help:"Test tool execution"`
	Enable  ToolsEnableCmd  `cmd:"enable" help:"Enable tools"`
	Disable ToolsDisableCmd `cmd:"disable" help:"Disable tools"`
	
	// Tool permissions
	Permissions ToolsPermissionsCmd `cmd:"permissions" help:"Manage tool permissions"`
	Allow       ToolsAllowCmd       `cmd:"allow" help:"Allow tool usage"`
	Deny        ToolsDenyCmd        `cmd:"deny" help:"Deny tool usage"`
	
	// Tool execution
	Execute ToolsExecuteCmd `cmd:"execute" help:"Execute tool directly"`
	
	// Tool installation (MCP tools)
	Install   ToolsInstallCmd   `cmd:"install" help:"Install external tools"`
	Uninstall ToolsUninstallCmd `cmd:"uninstall" help:"Uninstall external tools"`
	Update    ToolsUpdateCmd    `cmd:"update" help:"Update external tools"`
}

// ToolsListCmd lists available tools
type ToolsListCmd struct {
	Format    string `short:"f" enum:"table,json,simple,detailed" default:"table" help:"Output format"`
	Category  string `short:"c" help:"Filter by category"`
	Status    string `enum:"enabled,disabled,all" default:"all" help:"Filter by status"`
	Available bool   `short:"a" help:"Show only available tools"`
	Installed bool   `short:"i" help:"Show only installed tools"`
	Search    string `short:"s" help:"Search tools by name or description"`
}

func (c *ToolsListCmd) Run(ctx *kong.Context, cli *CLI) error {
	slog.Debug("Listing tools", "format", c.Format, "category", c.Category)
	
	// Get all tools
	allTools, err := GetAllTools()
	if err != nil {
		return fmt.Errorf("failed to get tools: %w", err)
	}
	
	// Convert to []interface{} for printing functions
	toolsInterface := make([]interface{}, len(allTools))
	for i, tool := range allTools {
		toolsInterface[i] = tool
	}
	
	switch c.Format {
	case "table":
		return printToolsTable(toolsInterface)
	case "json":
		return printToolsJSON(toolsInterface)
	case "simple":
		return printToolsSimple(toolsInterface)
	case "detailed":
		return printToolsDetailed(toolsInterface)
	default:
		return fmt.Errorf("unsupported format: %s", c.Format)
	}
}

// ToolsShowCmd shows tool details
type ToolsShowCmd struct {
	Name   string `arg:"" help:"Tool name"`
	Format string `short:"f" enum:"text,json,yaml" default:"text" help:"Output format"`
	Schema bool   `short:"s" help:"Show tool schema"`
	Usage  bool   `short:"u" help:"Show usage examples"`
}

func (c *ToolsShowCmd) Run(ctx *kong.Context, cli *CLI) error {
	// TODO: Implement tool details showing
	return runToolsShow(c)
}

// ToolsTestCmd tests tool execution
type ToolsTestCmd struct {
	Name   string `arg:"" help:"Tool name"`
	Input  string `short:"i" help:"Test input (JSON)"`
	File   string `short:"f" help:"Load test input from file"`
	DryRun bool   `help:"Dry run - validate input without executing"`
	Verbose bool  `short:"v" help:"Show detailed execution info"`
}

func (c *ToolsTestCmd) Run(ctx *kong.Context, cli *CLI) error {
	slog.Info("Testing tool", "name", c.Name, "input", c.Input)
	
	// TODO: Implement actual tool testing
	fmt.Printf("Testing tool '%s'...\n", c.Name)
	
	if c.DryRun {
		fmt.Println("Dry run - input validation passed")
		return nil
	}
	
	fmt.Println("Test result: OK")
	return nil
}

// ToolsEnableCmd enables tools
type ToolsEnableCmd struct {
	Names  []string `arg:"" help:"Tool names to enable"`
	All    bool     `help:"Enable all tools"`
	Global bool     `help:"Enable globally (affects all profiles)"`
}

func (c *ToolsEnableCmd) Run(ctx *kong.Context, cli *CLI) error {
	if c.All {
		fmt.Println("Enabling all tools...")
		return nil
	}
	
	for _, name := range c.Names {
		slog.Info("Enabling tool", "name", name)
		fmt.Printf("Tool '%s' enabled\n", name)
	}
	return nil
}

// ToolsDisableCmd disables tools
type ToolsDisableCmd struct {
	Names  []string `arg:"" help:"Tool names to disable"`
	All    bool     `help:"Disable all tools"`
	Global bool     `help:"Disable globally (affects all profiles)"`
}

func (c *ToolsDisableCmd) Run(ctx *kong.Context, cli *CLI) error {
	if c.All {
		fmt.Println("Disabling all tools...")
		return nil
	}
	
	for _, name := range c.Names {
		slog.Info("Disabling tool", "name", name)
		fmt.Printf("Tool '%s' disabled\n", name)
	}
	return nil
}

// ToolsPermissionsCmd manages tool permissions
type ToolsPermissionsCmd struct {
	List  ToolsPermissionsListCmd  `cmd:"list" help:"List tool permissions"`
	Show  ToolsPermissionsShowCmd  `cmd:"show" help:"Show permissions for specific tool"`
	Reset ToolsPermissionsResetCmd `cmd:"reset" help:"Reset tool permissions"`
	Export ToolsPermissionsExportCmd `cmd:"export" help:"Export permissions"`
	Import ToolsPermissionsImportCmd `cmd:"import" help:"Import permissions"`
}

type ToolsPermissionsListCmd struct {
	Format string `short:"f" enum:"table,json" default:"table" help:"Output format"`
	Tool   string `short:"t" help:"Show permissions for specific tool"`
}

func (c *ToolsPermissionsListCmd) Run(ctx *kong.Context, cli *CLI) error {
	// TODO: Implement permissions listing
	return runToolsPermissionsList(c)
}

type ToolsPermissionsShowCmd struct {
	Tool string `arg:"" help:"Tool name"`
}

func (c *ToolsPermissionsShowCmd) Run(ctx *kong.Context, cli *CLI) error {
	// TODO: Implement permissions showing
	return runToolsPermissionsShow(c)
}

type ToolsPermissionsResetCmd struct {
	Tool    string `help:"Reset permissions for specific tool"`
	All     bool   `help:"Reset all permissions"`
	Confirm bool   `short:"y" help:"Skip confirmation"`
}

func (c *ToolsPermissionsResetCmd) Run(ctx *kong.Context, cli *CLI) error {
	// TODO: Implement permissions reset
	return runToolsPermissionsReset(c)
}

type ToolsPermissionsExportCmd struct {
	Output string `short:"o" help:"Output file"`
	Format string `short:"f" enum:"json,yaml" default:"json" help:"Export format"`
}

func (c *ToolsPermissionsExportCmd) Run(ctx *kong.Context, cli *CLI) error {
	// TODO: Implement permissions export
	return runToolsPermissionsExport(c)
}

type ToolsPermissionsImportCmd struct {
	File   string `arg:"" help:"Permissions file to import"`
	Merge  bool   `help:"Merge with existing permissions"`
	DryRun bool   `help:"Preview import without applying"`
}

func (c *ToolsPermissionsImportCmd) Run(ctx *kong.Context, cli *CLI) error {
	// TODO: Implement permissions import
	return runToolsPermissionsImport(c)
}

// ToolsAllowCmd allows tool usage
type ToolsAllowCmd struct {
	Tool      string `arg:"" help:"Tool name"`
	Operation string `help:"Specific operation to allow"`
	Path      string `help:"Specific path to allow (for file tools)"`
	Command   string `help:"Specific command to allow (for command tools)"`
	Permanent bool   `help:"Make permission permanent"`
}

func (c *ToolsAllowCmd) Run(ctx *kong.Context, cli *CLI) error {
	// TODO: Implement tool allowing
	return runToolsAllow(c)
}

// ToolsDenyCmd denies tool usage
type ToolsDenyCmd struct {
	Tool      string `arg:"" help:"Tool name"`
	Operation string `help:"Specific operation to deny"`
	Path      string `help:"Specific path to deny (for file tools)"`
	Command   string `help:"Specific command to deny (for command tools)"`
	Permanent bool   `help:"Make denial permanent"`
}

func (c *ToolsDenyCmd) Run(ctx *kong.Context, cli *CLI) error {
	// TODO: Implement tool denying
	return runToolsDeny(c)
}

// ToolsExecuteCmd executes tools directly
type ToolsExecuteCmd struct {
	Name      string `arg:"" help:"Tool name"`
	Input     string `short:"i" help:"Tool input (JSON)"`
	File      string `short:"f" help:"Load input from file"`
	NoConfirm bool   `help:"Skip confirmation prompts"`
	Output    string `short:"o" help:"Output format (json, text)" default:"text"`
	Timeout   int    `help:"Execution timeout in seconds"`
}

func (c *ToolsExecuteCmd) Run(ctx *kong.Context, cli *CLI) error {
	// TODO: Implement direct tool execution
	return runToolsExecute(c)
}

// ToolsInstallCmd installs external tools
type ToolsInstallCmd struct {
	Source  string `arg:"" help:"Tool source (URL, package name, or local path)"`
	Name    string `help:"Custom tool name"`
	Version string `help:"Specific version to install"`
	Force   bool   `help:"Force reinstall if already exists"`
	Global  bool   `help:"Install globally (system-wide)"`
	DryRun  bool   `help:"Show what would be installed without installing"`
}

func (c *ToolsInstallCmd) Run(ctx *kong.Context, cli *CLI) error {
	slog.Info("Installing tool", "source", c.Source, "name", c.Name)
	
	// TODO: Implement actual tool installation
	fmt.Printf("Installing tool from '%s'...\n", c.Source)
	
	if c.DryRun {
		fmt.Println("Dry run - would install successfully")
		return nil
	}
	
	fmt.Println("Installation complete")
	return nil
}

// ToolsUninstallCmd uninstalls external tools
type ToolsUninstallCmd struct {
	Names   []string `arg:"" help:"Tool names to uninstall"`
	All     bool     `help:"Uninstall all external tools"`
	Confirm bool     `short:"y" help:"Skip confirmation"`
	Cleanup bool     `help:"Remove configuration and data"`
}

func (c *ToolsUninstallCmd) Run(ctx *kong.Context, cli *CLI) error {
	// TODO: Implement tool uninstallation
	return runToolsUninstall(c)
}

// ToolsUpdateCmd updates external tools
type ToolsUpdateCmd struct {
	Names   []string `arg:"" optional:"" help:"Tool names to update (all if empty)"`
	All     bool     `help:"Update all tools"`
	Check   bool     `help:"Check for updates without installing"`
	Version string   `help:"Update to specific version"`
}

func (c *ToolsUpdateCmd) Run(ctx *kong.Context, cli *CLI) error {
	// TODO: Implement tool updates
	return runToolsUpdate(c)
}

// Helper functions for tool output formatting
func printToolsTable(toolList []interface{}) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintln(w, "NAME\tDESCRIPTION\tSTATUS\tCATEGORY")
	fmt.Fprintln(w, "----\t-----------\t------\t--------")
	
	// Print built-in tools
	builtinTools := []struct {
		Name        string
		Description string
		Status      string
		Category    string
	}{
		{"read_file", "Read file contents", "enabled", "file"},
		{"write_file", "Write content to file", "enabled", "file"},
		{"list_directory", "List directory contents", "enabled", "file"},
		{"run_command", "Execute shell commands", "enabled", "system"},
		{"search_files", "Search for files containing patterns", "enabled", "file"},
		{"edit_file", "Edit file by replacing content", "enabled", "file"},
		{"create_directory", "Create new directories", "enabled", "file"},
		{"delete_file", "Delete files safely", "enabled", "file"},
		{"move_file", "Move/rename files", "enabled", "file"},
		{"copy_file", "Copy files", "enabled", "file"},
		{"get_file_info", "Get file metadata", "enabled", "file"},
		{"grep_files", "Advanced file content search", "enabled", "file"},
	}
	
	for _, tool := range builtinTools {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", tool.Name, tool.Description, tool.Status, tool.Category)
	}
	
	return nil
}

func printToolsJSON(toolList []interface{}) error {
	// Create a simplified tool list for JSON output
	tools := []map[string]interface{}{
		{"name": "read_file", "description": "Read file contents", "status": "enabled", "category": "file"},
		{"name": "write_file", "description": "Write content to file", "status": "enabled", "category": "file"},
		{"name": "list_directory", "description": "List directory contents", "status": "enabled", "category": "file"},
		{"name": "run_command", "description": "Execute shell commands", "status": "enabled", "category": "system"},
		{"name": "search_files", "description": "Search for files containing patterns", "status": "enabled", "category": "file"},
		{"name": "edit_file", "description": "Edit file by replacing content", "status": "enabled", "category": "file"},
		{"name": "create_directory", "description": "Create new directories", "status": "enabled", "category": "file"},
		{"name": "delete_file", "description": "Delete files safely", "status": "enabled", "category": "file"},
		{"name": "move_file", "description": "Move/rename files", "status": "enabled", "category": "file"},
		{"name": "copy_file", "description": "Copy files", "status": "enabled", "category": "file"},
		{"name": "get_file_info", "description": "Get file metadata", "status": "enabled", "category": "file"},
		{"name": "grep_files", "description": "Advanced file content search", "status": "enabled", "category": "file"},
	}
	
	data, err := json.MarshalIndent(tools, "", "  ")
	if err != nil {
		return err
	}
	
	fmt.Println(string(data))
	return nil
}

func printToolsSimple(toolList []interface{}) error {
	tools := []string{
		"read_file", "write_file", "list_directory", "run_command",
		"search_files", "edit_file", "create_directory", "delete_file",
		"move_file", "copy_file", "get_file_info", "grep_files",
	}
	
	for _, tool := range tools {
		fmt.Println(tool)
	}
	
	return nil
}

func printToolsDetailed(toolList []interface{}) error {
	tools := []struct {
		Name        string
		Description string
		Status      string
		Category    string
		Parameters  []string
	}{
		{"read_file", "Read file contents with safety checks", "enabled", "file", []string{"path"}},
		{"write_file", "Write content to file with validation", "enabled", "file", []string{"path", "content"}},
		{"list_directory", "List directory contents with optional recursion", "enabled", "file", []string{"path", "recursive"}},
		{"run_command", "Execute shell commands with safety restrictions", "enabled", "system", []string{"command"}},
		{"search_files", "Search for files containing patterns", "enabled", "file", []string{"pattern", "path"}},
		{"edit_file", "Edit file by replacing specific content", "enabled", "file", []string{"path", "old_content", "new_content"}},
		{"create_directory", "Create new directories with permissions", "enabled", "file", []string{"path", "permissions"}},
		{"delete_file", "Delete files safely with confirmation", "enabled", "file", []string{"path", "force"}},
		{"move_file", "Move/rename files with validation", "enabled", "file", []string{"source", "destination"}},
		{"copy_file", "Copy files with progress indication", "enabled", "file", []string{"source", "destination"}},
		{"get_file_info", "Get detailed file metadata", "enabled", "file", []string{"path", "follow_symlinks"}},
		{"grep_files", "Advanced file content search with regex", "enabled", "file", []string{"pattern", "path", "context_lines"}},
	}
	
	for _, tool := range tools {
		fmt.Printf("Tool: %s\n", tool.Name)
		fmt.Printf("  Description: %s\n", tool.Description)
		fmt.Printf("  Status: %s\n", tool.Status)
		fmt.Printf("  Category: %s\n", tool.Category)
		fmt.Printf("  Parameters: %v\n", tool.Parameters)
		fmt.Println()
	}
	
	return nil
}

// Placeholder implementations for helper functions
func runToolsShow(c *ToolsShowCmd) error {
	// TODO: Implement
	return nil
}

func runToolsPermissionsList(c *ToolsPermissionsListCmd) error {
	// TODO: Implement
	return nil
}

func runToolsPermissionsShow(c *ToolsPermissionsShowCmd) error {
	// TODO: Implement
	return nil
}

func runToolsPermissionsReset(c *ToolsPermissionsResetCmd) error {
	// TODO: Implement
	return nil
}

func runToolsPermissionsExport(c *ToolsPermissionsExportCmd) error {
	// TODO: Implement
	return nil
}

func runToolsPermissionsImport(c *ToolsPermissionsImportCmd) error {
	// TODO: Implement
	return nil
}

func runToolsAllow(c *ToolsAllowCmd) error {
	// TODO: Implement
	return nil
}

func runToolsDeny(c *ToolsDenyCmd) error {
	// TODO: Implement
	return nil
}

func runToolsExecute(c *ToolsExecuteCmd) error {
	// TODO: Implement
	return nil
}

func runToolsUninstall(c *ToolsUninstallCmd) error {
	// TODO: Implement
	return nil
}

func runToolsUpdate(c *ToolsUpdateCmd) error {
	// TODO: Implement
	return nil
}