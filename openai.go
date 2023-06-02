package main

import (
	"context"
	"github.com/sashabaranov/go-openai"
)

type Message struct {
	Role    string `yaml:"role" json:"role"`
	Content string `yaml:"content" json:"content"`
}

func chatCompletionStream(messages []Message) (*openai.ChatCompletionStream, error) {
	var oaiMessages []openai.ChatCompletionMessage
	for _, msg := range messages {
		oaiMessages = append(oaiMessages, openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}
	ctx := context.Background()
	req := openai.ChatCompletionRequest{
		Model:     openai.GPT3Dot5Turbo,
		MaxTokens: 150,
		Messages:  oaiMessages,
		Stream:    true,
	}

	c := openai.NewClient(getAPIKey())
	chunkStream, err := c.CreateChatCompletionStream(ctx, req)
	return chunkStream, err
}
