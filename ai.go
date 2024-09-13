package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/fatih/color"
	"github.com/go-yaml/yaml"
	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/process"
	"golang.org/x/crypto/ssh/terminal"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/debug"
	"strings"
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

func getSystemInfo() string {
	osName := runtime.GOOS
	platformSystem := runtime.GOARCH

	return fmt.Sprintf("operating system: %s\nplatform: %s\n", osName, platformSystem)
}

func getShell() string {
	knownShells := []string{"bash", "sh", "zsh", "powershell", "cmd", "fish", "tcsh", "csh", "ksh", "dash"}

	pid := os.Getppid()
	for {
		ppid, err := process.NewProcess(int32(pid))
		if err != nil {
			// If the process does not exist or there's another error, break the loop
			break
		}
		parentProcessName, err := ppid.Name()
		if err != nil {
			panic(err)
		}
		parentProcessName = strings.TrimSuffix(parentProcessName, ".exe")

		for _, shell := range knownShells {
			if parentProcessName == shell {
				return shell
			}
		}

		pidInt, err := ppid.Ppid()

		if err != nil {
			// If there's an error, break the loop
			break
		}
		pid = int(pidInt)
	}

	return ""
}

var shellCache *string = nil

func getShellCached() string {
	if shellCache == nil {
		shell := getShell()
		shellCache = &shell
	}
	return *shellCache
}

func getShellVersion(shell string) string {
	if shell == "" {
		return ""
	}

	var versionOutput *string = nil
	switch shell {
	case "powershell":
		// read: $PSVersionTable.PSVersion
		versionCmd := exec.Command(shell, "-Command", "$PSVersionTable.PSVersion")
		versionCmdOutput, err := versionCmd.Output()
		if err != nil {
			log.Printf("Error getting shell version: %s", shell)
			panic(err)
		}
		versionCmdOutputString := string(versionCmdOutput)
		versionOutput = &versionCmdOutputString
	default:
		versionCmd := exec.Command(shell, "--version")
		versionCmdOutput, err := versionCmd.Output()
		if err != nil {
			log.Printf("Error getting shell version: %s", shell)
			panic(err)
		}
		versionCmdOutputString := string(versionCmdOutput)
		versionOutput = &versionCmdOutputString
	}
	return strings.TrimSpace(*versionOutput)
}

func getWorkingDirectory() string {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return wd
}

func getPackageManagers() []string {
	packageManagers := []string{
		"pip", "conda", "npm", "yarn", "gem", "apt", "dnf", "yum", "pacman", "zypper", "brew", "choco", "scoop",
	}
	var installedPackageManagers []string

	for _, pm := range packageManagers {
		_, err := exec.LookPath(pm)
		if err == nil {
			installedPackageManagers = append(installedPackageManagers, pm)
		}
	}

	return installedPackageManagers
}

func sudoAvailable() bool {
	_, err := exec.LookPath("sudo")
	return err == nil
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

func generateChatGPTMessages(userInput string, mode Mode) []Message {
	shell := getShell()
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

type Mode int

const (
	CommandMode Mode = iota
	TextMode
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			// This block will execute when a panic occurs.
			// We can print a stack trace by calling debug.PrintStack.
			debug.PrintStack()
			fmt.Println("Panic:", r)
		}
	}()
	modelFlag := flag.String("model", "gpt-4-0613", "Model to use (e.g., gpt-4-0613 or gpt-3.5-turbo)")
	debugFlag := flag.Bool("debug", false, "Enable debug mode")
	executeFlag := flag.Bool("execute", false, "Execute the command instead of typing it out (dangerous!)")
	textFlag := flag.Bool("text", false, "Enable text mode")
	gpt3Flag := flag.Bool("3", false, "Shorthand for --model=gpt-3.5-turbo")
	initFlag := flag.Bool("init", false, "Initialize AI")

	// Add shorthands
	flag.StringVar(modelFlag, "m", "gpt-4", "Shorthand for model")
	flag.BoolVar(debugFlag, "d", false, "Shorthand for debug")
	flag.BoolVar(executeFlag, "x", false, "Shorthand for execute")

	flag.Parse()

	if initFlag != nil && *initFlag {
		initApiKey()
	}

	var mode = CommandMode
	if *gpt3Flag {
		*modelFlag = "gpt-3.5-turbo"
	}
	if *textFlag {
		mode = TextMode
	}

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
		fmt.Println("Model:", *modelFlag)
		fmt.Println("Debug:", *debugFlag)
		fmt.Println("User Input:", userInput)
	}

	isInteractive := isTerm(os.Stdin.Fd())
	withPipedInput := !isInteractive
	if withPipedInput {
		stdinBytes, err := ioutil.ReadAll(os.Stdin)
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
		for _, message := range messages {
			fmt.Println(message.Content)
		}
	}

	chunkStream, err := chatCompletionStream(messages)
	if err != nil {
		panic(err)
	}
	defer chunkStream.Close()

	var response = ""
	var firstResponse = true
	var functionName string
	var functionArgs string

	for {
		// Clear the 'thinking' message on first chunk
		if mode == CommandMode && firstResponse {
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

	if err != nil {
		log.Fatalln(err)
	}

	if mode == CommandMode {
		if functionName == "return_command" {
			var returnCommand ReturnCommandFunction
			err := json.Unmarshal([]byte(functionArgs), &returnCommand)
			if err != nil {
				log.Fatalln("Error parsing function arguments:", err)
			}

			// Check if required binaries are available
			missingBinaries := checkBinaries(returnCommand.Binaries)
			if len(missingBinaries) > 0 {
				color.Red("Missing required binaries: %s", strings.Join(missingBinaries, ", "))
				return
			}

			executableCommands := []string{returnCommand.Command}
			shell := getShellCached()

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
				typeCommands(executableCommands, keyboard)
			}
		} else {
			color.Yellow("No command returned. AI response:")
			fmt.Println(response)
		}
	} else {
		fmt.Println(response)
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

func getExecutableCommands(command string) []string {
	normalizeCommand := func(command string) string {
		return strings.Trim(command, " ")
	}
	var commands []string
	for _, command := range strings.Split(command, "\n") {
		if strings.HasPrefix(command, "#") {
			continue
		}
		normalizedCommand := normalizeCommand(command)
		if len(normalizedCommand) > 0 {
			commands = append(commands, normalizedCommand)
		}
	}
	return commands
}

func executeCommands(commands []string, shell string) {
	// if we're running bash, concatenate all commands into a single command with newlines, start with set -e
	// then pipe the whole thing into bash
	if shell == "bash" {
		command := fmt.Sprintf("set -e\n%s", strings.Join(commands, "\n"))
		err := executeCommand(command, shell)
		if err != nil {
			log.Fatalln(err)
		}
		return
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

func typeCommands(executableCommands []string, keyboard KeyboardInterface) {
	if len(executableCommands) == 0 {
		return
	}
	shellName := getShell()

	if shellName == "powershell" {
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
