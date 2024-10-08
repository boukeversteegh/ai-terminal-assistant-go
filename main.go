package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/fatih/color"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh/terminal"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"runtime/debug"
	"sort"
	"strings"
)

type Model string

func (m *Model) String() string {
	return string(*m)
}

func (m *Model) Set(value string) error {
	*m = Model(value)
	return nil
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			// This block will execute when a panic occurs.
			// We can print a stack trace by calling debug.PrintStack.
			debug.PrintStack()
			fmt.Println("Panic:", r)
		}
	}()
	var modelFlag Model = "gpt-4-0613"
	flag.Var(&modelFlag, "model", "Model to use (e.g., gpt-4-0613 or gpt-3.5-turbo)")
	debugFlag := flag.Bool("debug", false, "Enable debug mode")
	executeFlag := flag.Bool("execute", false, "Execute the command instead of typing it out (dangerous!)")
	textFlag := flag.Bool("text", false, "Enable text mode")
	gpt3Flag := flag.Bool("3", false, "Shorthand for --model=gpt-3.5-turbo")
	initFlag := flag.Bool("init", false, "Initialize AI")
	listModelsFlag := flag.Bool("list-models", false, "List available models")

	// Add shorthands
	flag.Var(&modelFlag, "m", "Shorthand for model")
	flag.BoolVar(debugFlag, "d", false, "Shorthand for debug")
	flag.BoolVar(executeFlag, "x", false, "Shorthand for execute")

	flag.Parse()

	if initFlag != nil && *initFlag {
		initApiKey()
	}

	aiClient := NewAIClient(getAPIKey(), modelFlag.String())

	if *listModelsFlag {
		listModels(aiClient)
		os.Exit(0)
	}

	var mode = CommandMode
	if *gpt3Flag {
		modelFlag = "gpt-3.5-turbo"
	}
	if *textFlag {
		mode = TextMode
		if *debugFlag {
			fmt.Println("Debug: Text mode enabled")
		}
	}
	if *debugFlag {
		fmt.Printf("Debug: Mode is %v\n", mode)
	}

	modelString := modelFlag.String()

	userInput := ""
	args := flag.Args()
	if len(args) > 0 {
		userInput = strings.Join(args, " ")
	} else {
		fmt.Println("Usage: ai [options] <natural language command>")
		flag.PrintDefaults()
		os.Exit(1)
	}

	if *debugFlag {
		fmt.Println("Model:", modelFlag.String())
		fmt.Println("Debug:", *debugFlag)
		fmt.Println("User Input:", userInput)
	}

	functionCalled := false

	isInteractive := isTerm(os.Stdin.Fd())
	withPipedInput := !isInteractive
	if withPipedInput {
		stdinBytes, err := io.ReadAll(os.Stdin)
		if err != nil {
			panic(err)
		}
		stdin := strings.TrimSpace(string(stdinBytes))
		if len(stdin) > 0 {
			userInput = fmt.Sprintf("%s\n\nUse the following additional context to improve your response:\n\n---\n\n%s\n", userInput, stdin)
		}
	}

	if mode == CommandMode {
		fmt.Printf("%s\r", color.YellowString("🤖 Thinking ..."))
	}

	var keyboard KeyboardInterface

	if mode == CommandMode && !*executeFlag {
		keyboard = NewKeyboard()
	}

	messages := generateChatGPTMessages(userInput, mode)
	if *debugFlag {
		fmt.Println("Debug: Messages generated")
		for i, msg := range messages {
			fmt.Printf("Debug: Message %d - Role: %s, Content: %.50s...\n", i, msg.Role, msg.Content)
		}
		for _, message := range messages {
			fmt.Println(message.Content)
		}
	}

	if mode == TextMode {
		response, err := aiClient.ChatCompletion(messages)
		if err != nil {
			log.Fatalln(err)
		}
		if *debugFlag {
			fmt.Printf("AI response (using model %s):\n", modelString)
		}
		fmt.Println(response)
	} else {
		chunkStream, err := aiClient.ChatCompletionStream(messages)
		if err != nil {
			panic(err)
		}
		defer chunkStream.Close()
		if *debugFlag {
			fmt.Println("Debug: Chat completion stream created")
		}

		var response = ""
		var firstResponse = true
		var functionName string
		var functionArgs string

		for {
			// Clear the 'thinking' message on first chunk
			if firstResponse {
				firstResponse = false
				color.Yellow("%s\r🤖", strings.Repeat(" ", 80))
			}

			chunkResponse, err := chunkStream.Recv()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				fmt.Printf("\nStream error: %v\n", err)
				return
			}

			if chunkResponse.Choices[0].Delta.FunctionCall != nil {
				functionCalled = true
				if chunkResponse.Choices[0].Delta.FunctionCall.Name != "" {
					functionName = chunkResponse.Choices[0].Delta.FunctionCall.Name
				}
				if chunkResponse.Choices[0].Delta.FunctionCall.Arguments != "" {
					functionArgs += chunkResponse.Choices[0].Delta.FunctionCall.Arguments
				}
			} else {
				chunk := chunkResponse.Choices[0].Delta.Content
				response += chunk
				printChunk(chunk, isInteractive)
			}
		}

		if *debugFlag {
			fmt.Printf("Function called: %v\n", functionCalled)
			if functionCalled {
				fmt.Printf("Function name: %s\n", functionName)
				fmt.Printf("Function arguments: %s\n", functionArgs)
			}
		}

		if functionName == "return_command" {
			var returnCommand ReturnCommandFunction
			err := json.Unmarshal([]byte(functionArgs), &returnCommand)
			if err != nil {
				log.Fatalln("Error parsing function arguments:", err)
			}

			if returnCommand.Command == "" {
				color.Yellow("No command returned. AI response:")
				fmt.Println(response)
				return
			}

			// Print the command in blue
			color.Blue(returnCommand.Command)

			// Check if required binaries are available
			missingBinaries := checkBinaries(returnCommand.Binaries)
			shell := getShellCached()
			if len(missingBinaries) > 0 {
				color.Yellow("Missing required binaries: %s", strings.Join(missingBinaries, ", "))

				// Inform the AI about missing binaries and ask for an alternative
				alternativeInput := fmt.Sprintf("The following binaries are missing: %s. Please provide a command to install these binaries, or if that's not possible, provide an alternative command that doesn't require these binaries. If installation instructions are complex, provide a brief explanation or a link to installation instructions.", strings.Join(missingBinaries, ", "))
				alternativeMessages := append(messages, Message{Role: "user", Content: alternativeInput})

				alternativeResponse, alternativeCommand := getAlternativeResponse(aiClient, alternativeMessages)

				if alternativeCommand != nil && alternativeCommand.Command != "" {
					fmt.Println("\nAI's alternative command:")
					fmt.Println(alternativeCommand.Command)

					// Check if required binaries for the alternative command are available
					missingBinaries := checkBinaries(alternativeCommand.Binaries)
					if len(missingBinaries) > 0 {
						color.Yellow("The alternative command also requires missing binaries: %s", strings.Join(missingBinaries, ", "))
						fmt.Println("\nAI's explanation:")
						fmt.Println(alternativeResponse)
					} else {
						if *executeFlag {
							executeCommands([]string{alternativeCommand.Command}, shell)
						} else {
							typeCommands([]string{alternativeCommand.Command}, keyboard, shell)
						}
					}
				} else {
					fmt.Println("\nAI's alternative response:")
					fmt.Println(alternativeResponse)
				}
				return
			}

			executableCommands := []string{returnCommand.Command}
			if *executeFlag {
				executeCommands(executableCommands, shell)
			} else {
				if !keyboard.IsFocusTheSame() {
					color.New(color.Faint).Println("Window focus changed during command generation.")
					color.Unset()

					if !withPipedInput {
						fmt.Println("Press enter to continue")
						fmt.Scanln()
					}
				}
				typeCommands(executableCommands, keyboard, shell)
			}
		} else {
			color.Yellow("No command returned. AI response:")
			fmt.Println(response)
		}
	}
}

func listModels(aiClient *AIClient) {
	models, err := aiClient.GetAvailableModels()
	if err != nil {
		fmt.Printf("Error fetching models: %v\n", err)
		return
	}

	fmt.Println("Available models:")
	sort.Strings(models)
	for _, model := range models {
		fmt.Println(model)
	}
}

func printChunk(content string, isInteractive bool) {
	if !isInteractive {
		fmt.Print(content)
		return
	}
	// Before lines that start with a hash, i.e. '\n#' or '^#', make the color green
	commentRegex := regexp.MustCompile(`(?m)((\n|^)#)`)
	var formattedContent = commentRegex.ReplaceAllString(content, fmt.Sprintf("%1s[%dm$1", "\x1b", color.FgGreen))

	// Insert a color reset before each newline
	var newlineRegex = regexp.MustCompile(`(?m)(\n)`)
	formattedContent = newlineRegex.ReplaceAllString(formattedContent, fmt.Sprintf("%1s[%dm$1", "\x1b", color.Reset))
	fmt.Print(formattedContent)
}

func executeCommands(commands []string, shell string) {
	switch shell {
	case "bash":
		command := fmt.Sprintf("set -e\n%s", strings.Join(commands, "\n"))
		err := executeCommand(command, shell)
		if err != nil {
			log.Fatalln(err)
		}
	case "powershell":
		for _, command := range commands {
			err := executeCommand(command, shell)
			if err != nil {
				log.Fatalln(err)
			}
		}
	default:
		log.Fatalf("Unsupported shell: %s", shell)
	}
}

func executeCommand(command string, shell string) error {
	var cmd *exec.Cmd
	switch shell {
	case "bash":
		cmd = exec.Command("bash")
		cmd.Stdin = strings.NewReader(command)
	case "powershell":
		cmd = exec.Command("powershell", "-Command", command)
	default:
		return fmt.Errorf("unsupported shell: %s", shell)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

type KeyboardInterface interface {
	SendString(string)
	SendNewLine()
	IsFocusTheSame() bool
}

func typeCommands(executableCommands []string, keyboard KeyboardInterface, shell string) {
	if len(executableCommands) == 0 {
		return
	}

	if shell == "powershell" {
		if len(executableCommands) == 1 {
			keyboard.SendString(executableCommands[0])
			return
		}

		keyboard.SendString("AiDo {\n")
		for _, command := range executableCommands {
			keyboard.SendString(command)
			keyboard.SendNewLine()
		}
		keyboard.SendString("}")
	} else {
		if len(executableCommands) == 1 {
			keyboard.SendString(executableCommands[0])
			return
		}
		keyboard.SendString("(")
		keyboard.SendNewLine()
		for _, command := range executableCommands {
			keyboard.SendString(command)
			keyboard.SendNewLine()
		}
		keyboard.SendString(")")
	}
}

func isTerm(fd uintptr) bool {
	return terminal.IsTerminal(int(fd))
}

func checkBinaries(binaries []string) []string {
	var missingBinaries []string
	for _, binary := range binaries {
		_, err := exec.LookPath(binary)
		if err != nil {
			missingBinaries = append(missingBinaries, binary)
		}
	}
	return missingBinaries
}

func getAlternativeResponse(aiClient *AIClient, messages []Message) (string, *ReturnCommandFunction) {
	chunkStream, err := aiClient.ChatCompletionStream(messages)
	if err != nil {
		panic(err)
	}
	defer chunkStream.Close()

	var response string
	var functionName string
	var functionArgs string
	var returnCommand *ReturnCommandFunction

	for {
		chunkResponse, err := chunkStream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			fmt.Printf("\nStream error: %v\n", err)
			return "", nil
		}

		if chunkResponse.Choices[0].Delta.FunctionCall != nil {
			if chunkResponse.Choices[0].Delta.FunctionCall.Name != "" {
				functionName = chunkResponse.Choices[0].Delta.FunctionCall.Name
			}
			if chunkResponse.Choices[0].Delta.FunctionCall.Arguments != "" {
				functionArgs += chunkResponse.Choices[0].Delta.FunctionCall.Arguments
			}
		} else if chunkResponse.Choices[0].Delta.Content != "" {
			response += chunkResponse.Choices[0].Delta.Content
		}
	}

	if functionName == "return_command" {
		returnCommand = &ReturnCommandFunction{}
		err := json.Unmarshal([]byte(functionArgs), returnCommand)
		if err != nil {
			log.Println("Error parsing function arguments:", err)
			return response, nil
		}
	}

	return response, returnCommand
}
