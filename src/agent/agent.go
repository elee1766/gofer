package agent

import (
	"context"
	"log/slog"

	"github.com/elee1766/gofer/src/aisdk"
)

type Agent struct {
	SystemPrompt string
	Model        aisdk.ModelClient
	Toolbox      *DefaultToolbox
	Logger       *slog.Logger
}

// TODO: this probably should have a parameters struct
func (a *Agent) SendMessageStream(ctx context.Context, conversation *aisdk.Conversation, message *aisdk.Message) (aisdk.StreamInterface, error) {
	messages := conversation.Messages
	// TODO: if the existing messages dont exist, need to initialize a system prompt
	if message != nil {
		messages = append(messages, message)
	}
	// Convert tools to ChatTool format
	var chatTools []*aisdk.ChatTool
	if a.Toolbox != nil {
		chatTools = ToChatTools(a.Toolbox.Tools())
	}
	
	ccr := &aisdk.ChatCompletionRequest{
		Messages: messages,
		Tools:    chatTools,
	}
	stream, err := a.Model.CreateChatCompletionStream(ctx, ccr)
	if err != nil {
		return nil, err
	}
	return stream, nil
}
