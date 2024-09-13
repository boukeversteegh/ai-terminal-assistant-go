package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-yaml/yaml"
)

type Shell struct {
	Messages []Message `yaml:"messages"`
}

type Prompts struct {
	Bash       Shell `yaml:"bash"`
	Powershell Shell `yaml:"powershell"`
	Command    struct {
		Messages []Message `yaml:"messages"`
	} `yaml:"command"`
	Text struct {
		Messages []Message `yaml:"messages"`
	} `yaml:"text"`
}

type Mode int

const (
	CommandMode Mode = iota
	TextMode
)

func generateChatGPTMessages(userInput string, mode Mode) []Message {
	shell := getShellCached()
	shellVersion := getShellVersion(shell)
	systemInfo := getSystemInfo()
	workingDirectory := getWorkingDirectory()
	packageManagers := getPackageManagers()
	sudo := sudoAvailable()

	prompts := Prompts{}

	aiHome := getAiHome()
	promptsFilePath := filepath.Join(aiHome, "prompts.yaml")
	promptsData, err := ioutil.ReadFile(promptsFilePath)
	if err != nil {
		log.Printf("Error reading prompts file: %s", promptsFilePath)
		panic(err)
	}
	err = yaml.Unmarshal(promptsData, &prompts)
	if err != nil {
		panic(err)
	}

	shellMessages := prompts.Bash.Messages
	if shell == "powershell" {
		shellMessages = prompts.Powershell.Messages
	}

	var commonMessages []Message
	if mode == CommandMode {
		commonMessages = prompts.Command.Messages
	} else {
		commonMessages = prompts.Text.Messages
	}

	for i := range commonMessages {
		commonMessages[i].Content = strings.NewReplacer(
			"{shell}", shell,
			"{shell_version}", shellVersion,
			"{system_info}", systemInfo,
			"{working_directory}", workingDirectory,
			"{package_managers}", strings.Join(packageManagers, ", "),
			"{sudo}", func() string {
				if sudo {
					return "sudo"
				}
				return "no sudo"
			}(),
		).Replace(commonMessages[i].Content)
	}

	userMessage := Message{
		Role:    "user",
		Content: userInput,
	}

	var outputMessages []Message

	// add common messages
	outputMessages = append(outputMessages, commonMessages...)

	// add shell messages if in command mode
	if mode == CommandMode {
		outputMessages = append(outputMessages, shellMessages...)
	}

	// add user message
	outputMessages = append(outputMessages, userMessage)
	return outputMessages
}

func getAiHome() string {
	aiHome := os.Getenv("AI_HOME")
	if aiHome == "" || strings.Contains(aiHome, "go-build") {
		// Fallback: use the directory of the current file
		_, filename, _, ok := runtime.Caller(0)
		if !ok {
			panic(errors.New("Failed to get current file path"))
		}
		aiHome = filepath.Dir(filename)
	}

	return aiHome
}
