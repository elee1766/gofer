package tools

// This file provides barrel-style re-exports for all tools, making them accessible
// from the main tools package for backward compatibility and convenience.
// Similar to JavaScript barrel exports.

import (
	"github.com/elee1766/gofer/src/agent"
	"github.com/elee1766/gofer/src/shell"
	tool_copyfile "github.com/elee1766/gofer/src/goferagent/tools/tool_copyfile"
	tool_createdir "github.com/elee1766/gofer/src/goferagent/tools/tool_createdir"
	tool_deletefile "github.com/elee1766/gofer/src/goferagent/tools/tool_deletefile"
	tool_editfile "github.com/elee1766/gofer/src/goferagent/tools/tool_editfile"
	tool_getfileinfo "github.com/elee1766/gofer/src/goferagent/tools/tool_getfileinfo"
	tool_grepfiles "github.com/elee1766/gofer/src/goferagent/tools/tool_grepfiles"
	tool_listdir "github.com/elee1766/gofer/src/goferagent/tools/tool_listdir"
	tool_movefile "github.com/elee1766/gofer/src/goferagent/tools/tool_movefile"
	tool_patchfile "github.com/elee1766/gofer/src/goferagent/tools/tool_patchfile"
	tool_readfile "github.com/elee1766/gofer/src/goferagent/tools/tool_readfile"
	tool_runcommand "github.com/elee1766/gofer/src/goferagent/tools/tool_runcommand"
	tool_searchfiles "github.com/elee1766/gofer/src/goferagent/tools/tool_searchfiles"
	tool_webfetch "github.com/elee1766/gofer/src/goferagent/tools/tool_webfetch"
	tool_writefile "github.com/elee1766/gofer/src/goferagent/tools/tool_writefile"
	"github.com/spf13/afero"
)

// Tool name constants - re-exported from individual packages
const (
	ReadFileName        = tool_readfile.Name
	WriteFileName       = tool_writefile.Name
	CopyFileName        = tool_copyfile.Name
	MoveFileName        = tool_movefile.Name
	DeleteFileName      = tool_deletefile.Name
	EditFileName        = tool_editfile.Name
	CreateDirectoryName = tool_createdir.Name
	ListDirectoryName   = tool_listdir.Name
	GetFileInfoName     = tool_getfileinfo.Name
	PatchName           = tool_patchfile.Name
	RunCommandName      = tool_runcommand.Name
	SearchFilesName     = tool_searchfiles.Name
	GrepFilesName       = tool_grepfiles.Name
	WebFetchName        = tool_webfetch.Name
)

// Filesystem-based tool constructors (require afero.Fs parameter) - re-exported as values
var (
)

// Non-filesystem tools (no fs parameter required) - re-exported as values
var (
)

// Tools that can return errors (kept as function wrappers)
func PatchTool() (agent.Tool, error) { return tool_patchfile.Tool() }
func ReadFileTool(fs afero.Fs) (agent.Tool, error) { return tool_readfile.ToolMultimodal(fs) }
func WriteFileTool(fs afero.Fs) (agent.Tool, error) { return tool_writefile.Tool(fs) }
func ListDirectoryTool(fs afero.Fs) (agent.Tool, error) { return tool_listdir.Tool(fs) }
func CreateDirectoryTool(fs afero.Fs) (agent.Tool, error) { return tool_createdir.Tool(fs) }
func EditFileTool(fs afero.Fs) (agent.Tool, error) { return tool_editfile.Tool(fs) }
func DeleteFileTool(fs afero.Fs) (agent.Tool, error) { return tool_deletefile.Tool(fs) }
func MoveFileTool(fs afero.Fs) (agent.Tool, error) { return tool_movefile.Tool(fs) }
func CopyFileTool(fs afero.Fs) (agent.Tool, error) { return tool_copyfile.Tool(fs) }
func GetFileInfoTool(fs afero.Fs) (agent.Tool, error) { return tool_getfileinfo.Tool(fs) }
func SearchFilesTool(fs afero.Fs) (agent.Tool, error) { return tool_searchfiles.Tool(fs) }
func GrepFilesTool(fs afero.Fs) (agent.Tool, error) { return tool_grepfiles.Tool(fs) }
func WebFetchTool() (agent.Tool, error) { return tool_webfetch.Tool() }

// Tools that require a shell manager
func RunCommandTool(shellManager *shell.ShellManager) agent.Tool { return tool_runcommand.Tool(shellManager) }
func RunCommandToolSingle(shellManager *shell.SingleShellManager) agent.Tool { return tool_runcommand.ToolWithSingleShell(shellManager) }