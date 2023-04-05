package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	sendkeys "git.tcp.direct/kayos/sendkeys"
	"github.com/fatih/color"
	"github.com/go-yaml/yaml"
	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/process"
	"golang.org/x/crypto/ssh/terminal"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type Message struct {
	Role    string `yaml:"role" json:"role"`
	Content string `yaml:"content" json:"content"`
}

type Shell struct {
	Messages []Message `yaml:"messages"`
}

type Prompts struct {
	Bash       Shell `yaml:"bash"`
	Powershell Shell `yaml:"powershell"`
	Common     struct {
		Messages []Message `yaml:"messages"`
	} `yaml:"common"`
}

func getSystemInfo() string {
	osName := runtime.GOOS
	platformSystem := runtime.GOARCH

	return fmt.Sprintf("operating system: %s\nplatform: %s\n", osName, platformSystem)
}

func getShell() string {
	knownShells := []string{"bash", "sh", "zsh", "powershell", "cmd", "fish", "tcsh", "csh", "ksh", "dash"}

	// Find the first non-Go process in the process tree
	pid := os.Getppid()
	var parentProcess *process.Process
	for {
		ppid, err := process.NewProcess(int32(pid))
		if err != nil {
			log.Fatal(err)
		}

		parentProcessName, err := ppid.Name()
		parentProcessName = strings.TrimSuffix(parentProcessName, ".exe")

		if err != nil {
			log.Fatal(err)
		}

		if parentProcessName != "go" {
			parentProcess = ppid
			break
		}

		pidInt, err := ppid.Ppid()
		pid = int(pidInt)

		if err != nil {
			log.Fatal(err)
		}
	}

	// Check if the process is a known shell
	parentProcessName, err := parentProcess.Name()
	if err != nil {
		log.Fatal(err)
	}

	parentProcessName = strings.ToLower(strings.TrimSuffix(parentProcessName, ".exe"))

	for _, shell := range knownShells {
		if parentProcessName == shell {
			return shell
		}
	}

	return ""
}

func getShellVersion(shell string) string {
	versionCmd := exec.Command(shell, "--version")
	versionOutput, err := versionCmd.Output()
	if err != nil {
		log.Fatal(err)
	}
	return strings.TrimSpace(string(versionOutput))
}

func getWorkingDirectory() string {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	return wd
}

func getPackageManagers() []string {
	packageManagers := []string{
		"pip", "conda", "npm", "yarn", "gem", "apt", "dnf", "yum", "pacman", "zypper", "brew", "choco", "scoop",
	}
	installedPackageManagers := []string{}

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

func generateChatGPTMessages(userInput string) []Message {
	shell := getShell()
	shellVersion := getShellVersion(shell)
	systemInfo := getSystemInfo()
	workingDirectory := getWorkingDirectory()
	packageManagers := getPackageManagers()
	sudo := sudoAvailable()

	prompts := Prompts{}

	promptsData, err := ioutil.ReadFile("prompts.yaml")
	if err != nil {
		log.Fatal(err)
	}
	err = yaml.Unmarshal(promptsData, &prompts)
	if err != nil {
		log.Fatal(err)
	}

	shellMessages := prompts.Bash.Messages
	if shell == "powershell" {
		shellMessages = prompts.Powershell.Messages
	}

	commonMessages := prompts.Common.Messages

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

	return append(append(commonMessages, shellMessages...), userMessage)
}

func main() {
	userInput := ""
	if len(os.Args) > 1 {
		userInput = os.Args[1]
	} else {
		fmt.Println("Usage: ./ai \"<natural language command>\"")
		os.Exit(1)
	}

	if !isTerm(os.Stdin.Fd()) {
		stdinBytes, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			log.Fatal(err)
		}
		stdin := strings.TrimSpace(string(stdinBytes))
		if len(stdin) > 0 {
			userInput = fmt.Sprintf("%s. Use the following additional context to improve your suggestion:\n\n---\n\n%s\n", userInput, stdin)
		}
	}

	messages := generateChatGPTMessages(userInput)

	color.New(color.FgYellow).Printf("🤖 Thinking ...")
	color.Unset()

	// Dummy delay to simulate the AI thinking
	time.Sleep(3 * time.Second)

	fmt.Printf("\r%s\r", strings.Repeat(" ", 80))
	color.New(color.FgYellow).Print("🤖")
	color.Unset()
	fmt.Println()

	// Call the function to process messages and type the commands
	processMessagesAndTypeCommands(messages)

}

type ChatGPTResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func processMessagesAndTypeCommands(messages []Message) {
	bashCommand, err := getBashCommand(messages)
	if err != nil {
		log.Fatalln(err)
	}

	// Normalize the command by removing unnecessary characters
	normalizeCommand := func(command string) string {
		return strings.Trim(strings.Trim(strings.Trim(command, "&& \\"), ";"), " ")
	}

	getExecutableCommands := func(command string) []string {
		commands := []string{}
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

	executableCommands := getExecutableCommands(bashCommand)

	for _, line := range strings.Split(bashCommand, "\n") {
		if len(strings.Trim(line, " ")) == 0 {
			continue
		}

		if strings.HasPrefix(line, "#") {
			comment := line
			fmt.Println(comment)
		} else if len(line) > 0 && len(executableCommands) > 1 {
			fmt.Println(line)
		}
	}

	typeCommands(executableCommands)
}

type APIError struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Param   string `json:"param"`
		Code    string `json:"code"`
	} `json:"error"`
}

func getBashCommand(messages []Message) (string, error) {
	// Replace with your actual API key or use os.Getenv("OPENAI_API_KEY") if it's set as an environment variable
	apiKey := os.Getenv("OPENAI_API_KEY")

	client := &http.Client{}

	requestBody, err := json.Marshal(map[string]interface{}{
		"model":    "gpt-3.5-turbo",
		"messages": messages,
	})
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal request body")
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(requestBody))
	if err != nil {
		return "", errors.Wrap(err, "failed to create new request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "failed to do request")
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "failed to read response body")
	}

	var apiError APIError
	if err := json.Unmarshal(body, &apiError); err != nil {
		panic(err)
	}

	if apiError.Error.Type != "" {
		fmt.Println("Error in response body:", apiError.Error.Message)
		os.Exit(1)
	}

	var chatGPTResponse ChatGPTResponse
	err = json.Unmarshal(body, &chatGPTResponse)
	if err != nil {
		return "", errors.Wrap(err, "failed to unmarshal response")
	}
	command := chatGPTResponse.Choices[0].Message.Content

	return command, nil
}

func typeCommands(executableCommands []string) {
	shellName := getShell()

	k, err := sendkeys.NewKBWrapWithOptions(sendkeys.Noisy)
	if err != nil {
		log.Fatal(err)
	}

	if shellName == "powershell" {
		if len(executableCommands) == 1 {
			k.Type(executableCommands[0])
			return
		}

		k.Type("AiDo {\n")
		for _, command := range executableCommands {
			k.Type(command)
			k.Enter()
		}
		k.Type("}")
	} else {
		for commandIndex, command := range executableCommands {
			k.Type(command)
			if commandIndex < len(executableCommands)-1 && !strings.HasSuffix(command, "\\") {
				k.Type(" && \\")
				k.Enter()
			}
		}
	}
}

func isTerm(fd uintptr) bool {
	return terminal.IsTerminal(int(fd))
}