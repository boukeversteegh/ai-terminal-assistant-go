package main

import (
	"context"
	"encoding/json"
	"github.com/sashabaranov/go-openai"
)

type Message struct {
	Role    string `yaml:"role" json:"role"`
	Content string `yaml:"content" json:"content"`
}

type ReturnCommandFunction struct {
	Command  string   `json:"command"`
	Binaries []string `json:"binaries"`
}

func chatCompletionStream(messages []Message) (*openai.ChatCompletionStream, error) {
	var oaiMessages []openai.ChatCompletionMessage
	for _, msg := range messages {
		oaiMessages = append(oaiMessages, openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	returnCommandFunction := openai.FunctionDefinition{
		Name: "return_command",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"command": {
					"type": "string",
					"description": "The full command to be executed"
				},
				"binaries": {
					"type": "array",
					"items": {
						"type": "string"
					},
					"description": "List of required binaries for the command"
				}
			},
			"required": ["command"]
		}`),
		Description: "Return a command to be executed along with any required binaries",
	}

	ctx := context.Background()
	req := openai.ChatCompletionRequest{
		Model:        openai.GPT40613,
		Messages:     oaiMessages,
		Stream:       true,
		Functions:    []openai.FunctionDefinition{returnCommandFunction},
		FunctionCall: openai.FunctionCall{Name: "return_command"},
	}

	c := openai.NewClient(getAPIKey())
	chunkStream, err := c.CreateChatCompletionStream(ctx, req)
	return chunkStream, err
}
