package main

import (
	"context"
	"encoding/json"
	"github.com/sashabaranov/go-openai"
)

type AIClient struct {
	client *openai.Client
	model  string
}

func NewAIClient(apiKey, model string) *AIClient {
	return &AIClient{
		client: openai.NewClient(apiKey),
		model:  model,
	}
}

func (ai *AIClient) ChatCompletion(messages []Message) (string, error) {
	var openaiMessages []openai.ChatCompletionMessage
	for _, msg := range messages {
		openaiMessages = append(openaiMessages, openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	resp, err := ai.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:    ai.model,
			Messages: openaiMessages,
		},
	)

	if err != nil {
		return "", err
	}

	return resp.Choices[0].Message.Content, nil
}

func (ai *AIClient) ChatCompletionStream(messages []Message) (*openai.ChatCompletionStream, error) {
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
		Model:        ai.model,
		Messages:     oaiMessages,
		Stream:       true,
		Functions:    []openai.FunctionDefinition{returnCommandFunction},
		FunctionCall: openai.FunctionCall{Name: "return_command"},
	}

	return ai.client.CreateChatCompletionStream(ctx, req)
}

func (ai *AIClient) GetAvailableModels() ([]string, error) {
	modelList, err := ai.client.ListModels(context.Background())
	if err != nil {
		return nil, err
	}

	var models []string
	for _, model := range modelList.Models {
		models = append(models, model.ID)
	}
	return models, nil
}
