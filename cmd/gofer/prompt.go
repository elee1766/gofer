package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/elee1766/gofer/src/agent"
	"github.com/elee1766/gofer/src/aisdk"
	"github.com/elee1766/gofer/src/app"
	"github.com/elee1766/gofer/src/goferagent"
	"github.com/elee1766/gofer/src/goferagent/tools"
	"github.com/elee1766/gofer/src/executor"
	"github.com/elee1766/gofer/src/shell"
	"github.com/elee1766/gofer/src/storage"
	"github.com/spf13/afero"
)

type RunPromptParams struct {
	APIKey       string
	Model        string
	Text         string
	SystemPrompt string
	Output       string
	Raw          bool
	EnableTools  bool
	Logger       *slog.Logger
	Resume       bool
	SessionID    string
	MaxTurns     int
	Verbose      bool
}

// RunPrompt executes a single prompt command using the new prompt package
func RunPrompt(ctx context.Context, a *app.App, params RunPromptParams) error {
	if params.Text == "" {
		return fmt.Errorf("prompt text is required")
	}

	// Determine the model to use
	model := params.Model
	if model == "" {
		return fmt.Errorf("model is required - no default model configured")
	}

	modelClient, err := a.ModelProvider.Model(ctx, model)
	if err != nil {
		return fmt.Errorf("failed to get model client: %w", err)
	}

	// Create single shell manager for tools that need it
	var singleShellManager *shell.SingleShellManager
	if params.EnableTools {
		var err error
		singleShellManager, err = shell.NewSingleShellManager(params.Logger)
		if err != nil {
			return fmt.Errorf("failed to create shell manager: %w", err)
		}
		defer singleShellManager.Close()
	}

	// Set up toolbox (will be created contextually later)
	var toolbox *agent.DefaultToolbox
	if params.EnableTools {
		toolbox, err = createToolbox(params.Logger, afero.NewOsFs(), singleShellManager)
		if err != nil {
			return fmt.Errorf("failed to create toolbox: %w", err)
		}
	}

	// Determine system prompt
	systemPrompt := params.SystemPrompt
	if systemPrompt == "" && params.EnableTools {
		systemPrompt = goferagent.GetDefaultSystemPrompt(toolbox)
	}

	// Create executor service
	service := executor.NewService(executor.ServiceConfig{
		Database:     a.Store.DB(),
		ProjectDir:   a.ProjectDir,
		SystemPrompt: systemPrompt,
		MaxTurns:     3,
		Logger:       params.Logger,
	})

	// Create event sink and processor
	processorConfig := executor.ConsoleProcessorConfig{
		ShowTimestamps:      false,
		ShowTurnNumbers:     false,
		ShowToolArguments:   true,
		ShowToolResults:     true,
		ShowIntermediateAI:  true,
		RawMode:            params.Raw,
		MaxResultPreview:    200,
	}
	
	consoleProcessor := executor.NewConsoleEventProcessor(processorConfig)
	eventSink := executor.NewChannelEventSink(100, consoleProcessor)
	defer eventSink.Close()
	
	// Legacy callbacks for compatibility (will be phased out)
	callbacks := &executor.Callbacks{}

	// Get or create session and conversation
	session, conversation, err := getOrCreateSessionAndConversation(ctx, service, params)
	if err != nil {
		return err
	}

	// Build conversation from existing messages
	aisdkConv, err := buildConversationFromDB(ctx, service, conversation, params.SystemPrompt)
	if err != nil {
		return err
	}

	// Create and save user message (will be wrapped with context later)
	originalUserText := params.Text

	// Save user message to database
	err = service.SaveUserMessage(ctx, conversation.ID, params.Text)
	if err != nil {
		return fmt.Errorf("failed to save user message: %w", err)
	}

	// Execute conversation with turn tracking
	maxTurns := params.MaxTurns
	if maxTurns <= 0 {
		maxTurns = 3
	}

	currentConv := aisdkConv
	turnsRemaining := maxTurns
	isFirstTurn := true
	justExecutedTools := false

	for turnsRemaining > 0 {
		// For single shell, we just use the same toolbox without conversation-specific context
		contextualToolbox := toolbox

		// Prepare the message for this step
		var messageToSend *aisdk.Message
		
		if isFirstTurn {
			// Wrap the initial user message with context
			wrappedContent := executor.WrapFirstMessage(originalUserText, turnsRemaining, params.EnableTools)
			messageToSend = &aisdk.Message{
				Role:    "user",
				Content: wrappedContent,
			}
		} else if justExecutedTools && turnsRemaining > 1 {
			// After tool execution, let LLM assess completion rather than pushing to continue
			messageToSend = nil
		} else {
			// No message needed - just tool results
			messageToSend = nil
		}

		// Step 1: Send message to LLM
		stepReq := &executor.StepRequest{
			Conversation:   currentConv,
			Message:        messageToSend,
			ModelClient:    modelClient,
			SessionID:      session.ID,
			ConversationID: conversation.ID,
			Toolbox:        contextualToolbox,
			Callbacks:      callbacks,
			EventSink:      eventSink,
			TurnNumber:     maxTurns - turnsRemaining + 1,
		}

		stepResult, err := service.Step(ctx, stepReq)
		if err != nil {
			return err
		}

		if stepResult.State == executor.StateError {
			return stepResult.Error
		}

		currentConv = stepResult.UpdatedConversation
		isFirstTurn = false
		justExecutedTools = false

		// Events are now handled by the event processor, no need for manual printing

		// Step 2: If no tool calls, we're done
		if stepResult.State == executor.StateTextResponse {
			turnsRemaining--
			break
		}

		// Step 3: Execute tool calls if needed
		if stepResult.State == executor.StateToolCallsNeeded {
			toolReq := &executor.ToolExecutionRequest{
				ToolCalls:      stepResult.ToolCalls,
				Toolbox:        contextualToolbox,
				SessionID:      session.ID,
				ConversationID: conversation.ID,
				Model:          modelClient.GetModelInfo().ID,
				Callbacks:      callbacks,
				EventSink:      eventSink,
				TurnNumber:     maxTurns - turnsRemaining + 1,
			}

			toolResult, err := service.ExecuteToolCalls(ctx, toolReq)
			if err != nil {
				return err
			}

			if toolResult.State == executor.StateError {
				return toolResult.Error
			}

			// Add tool results to conversation
			for _, toolMsg := range toolResult.ToolResults {
				currentConv.Messages = append(currentConv.Messages, toolMsg)
			}

			// Mark that we just executed tools and continue to next iteration
			justExecutedTools = true
			turnsRemaining--
			continue
		}

		// Fallback - shouldn't reach here normally
		turnsRemaining--
		break
	}

	// Output is now handled by the console processor via events

	// Emit conversation complete event
	if eventSink != nil {
		emitter := executor.NewEventEmitter(eventSink, conversation.ID, maxTurns - turnsRemaining)
		if turnsRemaining <= 0 {
			emitter.EmitConversationComplete("max_turns", maxTurns - turnsRemaining, turnsRemaining)
			params.Logger.Warn("Max turns reached", "turns", params.MaxTurns)
		} else {
			emitter.EmitConversationComplete("task_complete", maxTurns - turnsRemaining, turnsRemaining)
		}
	}

	return nil
}

// RunPromptWithApp executes a single prompt command using the shared app instance
func RunPromptWithApp(ctx context.Context, a *app.App, params RunPromptParams) error {
	return RunPrompt(ctx, a, params)
}

// createToolbox creates a toolbox with all the default tools using the provided filesystem and shell manager
func createToolbox(logger *slog.Logger, fs afero.Fs, singleShellManager *shell.SingleShellManager) (*agent.DefaultToolbox, error) {
	toolbox := agent.NewToolbox[agent.Tool]()

	// List of filesystem-based tool creation functions
	fsToolCreators := []struct {
		name    string
		creator func(afero.Fs) agent.Tool
	}{
	}

	// Register all filesystem-based tools
	for _, tc := range fsToolCreators {
		tool := tc.creator(fs)
		if err := toolbox.RegisterTool(tool); err != nil {
			return nil, fmt.Errorf("failed to register %s tool: %w", tc.name, err)
		}
		if logger != nil {
			logger.Debug("Registered tool", "tool", tc.name)
		}
	}

	// List of non-filesystem tool creation functions
	nonFsToolCreators := []struct {
		name    string
		creator func() agent.Tool
	}{
	}

	// Register all non-filesystem tools
	for _, tc := range nonFsToolCreators {
		tool := tc.creator()
		if err := toolbox.RegisterTool(tool); err != nil {
			return nil, fmt.Errorf("failed to register %s tool: %w", tc.name, err)
		}
		if logger != nil {
			logger.Debug("Registered tool", "tool", tc.name)
		}
	}

	// Register tools that can return errors (like GenericTools)
	patchTool, err := tools.PatchTool()
	if err != nil {
		return nil, fmt.Errorf("failed to create patch tool: %w", err)
	}
	if err := toolbox.RegisterTool(patchTool); err != nil {
		return nil, fmt.Errorf("failed to register patch tool: %w", err)
	}
	if logger != nil {
		logger.Debug("Registered tool", "tool", tools.PatchName)
	}

	// Register ReadFileTool (now returns error)
	readFileTool, err := tools.ReadFileTool(fs)
	if err != nil {
		return nil, fmt.Errorf("failed to create read file tool: %w", err)
	}
	if err := toolbox.RegisterTool(readFileTool); err != nil {
		return nil, fmt.Errorf("failed to register read file tool: %w", err)
	}
	if logger != nil {
		logger.Debug("Registered tool", "tool", tools.ReadFileName)
	}

	// Register WriteFileTool (now returns error)
	writeFileTool, err := tools.WriteFileTool(fs)
	if err != nil {
		return nil, fmt.Errorf("failed to create write file tool: %w", err)
	}
	if err := toolbox.RegisterTool(writeFileTool); err != nil {
		return nil, fmt.Errorf("failed to register write file tool: %w", err)
	}
	if logger != nil {
		logger.Debug("Registered tool", "tool", tools.WriteFileName)
	}

	// Register ListDirectoryTool (now returns error)
	listDirTool, err := tools.ListDirectoryTool(fs)
	if err != nil {
		return nil, fmt.Errorf("failed to create list directory tool: %w", err)
	}
	if err := toolbox.RegisterTool(listDirTool); err != nil {
		return nil, fmt.Errorf("failed to register list directory tool: %w", err)
	}
	if logger != nil {
		logger.Debug("Registered tool", "tool", tools.ListDirectoryName)
	}

	// Register CreateDirectoryTool (now returns error)
	createDirTool, err := tools.CreateDirectoryTool(fs)
	if err != nil {
		return nil, fmt.Errorf("failed to create create directory tool: %w", err)
	}
	if err := toolbox.RegisterTool(createDirTool); err != nil {
		return nil, fmt.Errorf("failed to register create directory tool: %w", err)
	}
	if logger != nil {
		logger.Debug("Registered tool", "tool", tools.CreateDirectoryName)
	}

	// Register EditFileTool (now returns error)
	editFileTool, err := tools.EditFileTool(fs)
	if err != nil {
		return nil, fmt.Errorf("failed to create edit file tool: %w", err)
	}
	if err := toolbox.RegisterTool(editFileTool); err != nil {
		return nil, fmt.Errorf("failed to register edit file tool: %w", err)
	}
	if logger != nil {
		logger.Debug("Registered tool", "tool", tools.EditFileName)
	}

	// Register DeleteFileTool (now returns error)
	deleteFileTool, err := tools.DeleteFileTool(fs)
	if err != nil {
		return nil, fmt.Errorf("failed to create delete file tool: %w", err)
	}
	if err := toolbox.RegisterTool(deleteFileTool); err != nil {
		return nil, fmt.Errorf("failed to register delete file tool: %w", err)
	}
	if logger != nil {
		logger.Debug("Registered tool", "tool", tools.DeleteFileName)
	}

	// Register MoveFileTool (now returns error)
	moveFileTool, err := tools.MoveFileTool(fs)
	if err != nil {
		return nil, fmt.Errorf("failed to create move file tool: %w", err)
	}
	if err := toolbox.RegisterTool(moveFileTool); err != nil {
		return nil, fmt.Errorf("failed to register move file tool: %w", err)
	}
	if logger != nil {
		logger.Debug("Registered tool", "tool", tools.MoveFileName)
	}

	// Register CopyFileTool (now returns error)
	copyFileTool, err := tools.CopyFileTool(fs)
	if err != nil {
		return nil, fmt.Errorf("failed to create copy file tool: %w", err)
	}
	if err := toolbox.RegisterTool(copyFileTool); err != nil {
		return nil, fmt.Errorf("failed to register copy file tool: %w", err)
	}
	if logger != nil {
		logger.Debug("Registered tool", "tool", tools.CopyFileName)
	}

	// Register GetFileInfoTool (now returns error)
	getFileInfoTool, err := tools.GetFileInfoTool(fs)
	if err != nil {
		return nil, fmt.Errorf("failed to create get file info tool: %w", err)
	}
	if err := toolbox.RegisterTool(getFileInfoTool); err != nil {
		return nil, fmt.Errorf("failed to register get file info tool: %w", err)
	}
	if logger != nil {
		logger.Debug("Registered tool", "tool", tools.GetFileInfoName)
	}

	// Register SearchFilesTool (now returns error)
	searchFilesTool, err := tools.SearchFilesTool(fs)
	if err != nil {
		return nil, fmt.Errorf("failed to create search files tool: %w", err)
	}
	if err := toolbox.RegisterTool(searchFilesTool); err != nil {
		return nil, fmt.Errorf("failed to register search files tool: %w", err)
	}
	if logger != nil {
		logger.Debug("Registered tool", "tool", tools.SearchFilesName)
	}

	// Register GrepFilesTool (now returns error)
	grepFilesTool, err := tools.GrepFilesTool(fs)
	if err != nil {
		return nil, fmt.Errorf("failed to create grep files tool: %w", err)
	}
	if err := toolbox.RegisterTool(grepFilesTool); err != nil {
		return nil, fmt.Errorf("failed to register grep files tool: %w", err)
	}
	if logger != nil {
		logger.Debug("Registered tool", "tool", tools.GrepFilesName)
	}

	// Register WebFetchTool (now returns error)
	webFetchTool, err := tools.WebFetchTool()
	if err != nil {
		return nil, fmt.Errorf("failed to create web fetch tool: %w", err)
	}
	if err := toolbox.RegisterTool(webFetchTool); err != nil {
		return nil, fmt.Errorf("failed to register web fetch tool: %w", err)
	}
	if logger != nil {
		logger.Debug("Registered tool", "tool", tools.WebFetchName)
	}

	// Register RunCommandTool (requires single shell manager)
	if singleShellManager != nil {
		runCommandTool := tools.RunCommandToolSingle(singleShellManager)
		if err := toolbox.RegisterTool(runCommandTool); err != nil {
			return nil, fmt.Errorf("failed to register run command tool: %w", err)
		}
		if logger != nil {
			logger.Debug("Registered tool", "tool", tools.RunCommandName)
		}
	}

	return toolbox, nil
}


// getOrCreateSessionAndConversation handles session and conversation setup
func getOrCreateSessionAndConversation(ctx context.Context, service *executor.Service, params RunPromptParams) (session *storage.Session, conversation *storage.Conversation, err error) {
	// Get or create session
	session, err = service.GetOrCreateSession(ctx, params.SessionID, params.Resume)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get or create session: %w", err)
	}

	// Get or create conversation
	conversation, err = service.GetOrCreateConversation(ctx, session)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get or create conversation: %w", err)
	}

	return session, conversation, nil
}

// buildConversationFromDB builds an aisdk.Conversation from database messages
func buildConversationFromDB(ctx context.Context, service *executor.Service, conversation *storage.Conversation, systemPrompt string) (*aisdk.Conversation, error) {
	return service.BuildConversationFromDB(ctx, conversation, systemPrompt)
}