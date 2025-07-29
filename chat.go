package main

import (
	"context"
	"fmt"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/memory"
)

type Chat interface {
	SetRole(ctx context.Context, role string) error
	GenerateContentForUser(ctx context.Context, message string) (string, error)
}

func NewChat(llm llms.Model) Chat {
	return &chat{
		llm:    llm,
		memory: memory.NewConversationBuffer(),
	}
}

type chat struct {
	llm    llms.Model
	memory *memory.ConversationBuffer
}

func (c *chat) SetRole(ctx context.Context, role string) error {
	if err := c.memory.ChatHistory.Clear(ctx); err != nil {
		return fmt.Errorf("failed to clear chat history: %w", err)
	}

	return c.memory.ChatHistory.AddMessage(ctx, generateSystemMessage(role))
}

func generateSystemMessage(role string) llms.SystemChatMessage {
	return llms.SystemChatMessage{
		Content: "Отныне ты - " + role + ". Ты должен отвечать на вопросы и помогать пользователю в соответствии с этой ролью.",
	}
}

func (c *chat) GenerateContentForUser(ctx context.Context, message string) (string, error) {
	if err := c.memory.ChatHistory.AddUserMessage(ctx, message); err != nil {
		return "", fmt.Errorf("failed to add user message: %w", err)
	}

	history, err := c.memory.ChatHistory.Messages(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get chat history: %w", err)
	}

	fullHistory := make([]llms.MessageContent, 0, len(history))
	for _, m := range history {
		fullHistory = append(fullHistory, llms.TextParts(m.GetType(), m.GetContent()))
	}

	response, err := c.llm.GenerateContent(ctx, fullHistory)
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %w", err)
	}

	aiResponse := response.Choices[0].Content

	if err := c.memory.ChatHistory.AddAIMessage(ctx, aiResponse); err != nil {
		return "", fmt.Errorf("failed to add AI message: %w", err)
	}

	return aiResponse, nil
}
