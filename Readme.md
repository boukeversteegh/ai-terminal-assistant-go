# 🤖 AI Terminal Assistant

*Your Personal Shell Expert*

AI Terminal Assistant is an intelligent and user-friendly command-line tool designed to help you generate and execute shell commands with natural language inputs effortlessly. Say goodbye to memorizing countless command syntaxes and enjoy a seamless and secure user experience that adapts to your terminal shell, operating system, and package managers.

## 🌟 Main Features

1. **AI-Powered Assistance:** Generates shell commands for various terminal environments using the OpenAI API, boosting your productivity and making complex tasks easy.
2. **Shell Command Execution:** Executes the AI-generated shell commands directly or simulates typing the command into the terminal, keeping your system secure.
3. **Dynamic Shell and System Detection:** Automatically detects your active shell environment, system information, and package managers, ensuring accurate AI responses.
4. **Cross Platform:** Works on Windows, macOS, and Linux (beta).

## Usage

Unleash the power of AI Assistant to simplify your command-line tasks. Just pass a natural language command in quotes as an argument:

```bash
$ ai how to create a new directory called myfolder
# 🤖 To create a new directory called myfolder, type the following command:
mkdir myfolder
$ mkdir myfolder█
```

Use it in powershell:

```powershell
PS C:\Users\jdoe> ai how much free disk space in mb?
# Show free disk space for all drives in megabytes (MB).
PS C:\Users\jdoe> Get-PSDrive -PSProvider FileSystem | Select-Object Name, @{Name="FreeSpaceMB"; Expression={$_.Free / 1MB -as [int]}}
```
After pressing enter, the command will be executed in the terminal:

```
Name FreeSpaceMB
---- -----------
C           9375
```

Embrace the future with AI Assistant, and enhance your command-line experience with the power of AI!

## Installation

- Download the [latest release](https://github.com/boukeversteegh/ai-terminal-assistant-go/releases/latest) from the releases page.
- Extract the archive.
- Optionally, add the executable to your PATH.
- Run the executable.
- Enter your OpenAI API key when prompted.
- Enjoy!

### MacOS

- You may need to allow the app to run in System Preferences > Security & Privacy > General.
  - Alternatively, open the `ai` executable in finder by right-clicking and selecting "Open". Then, click "Open" in the dialog that appears. After this, you should be able to run the app normally.
- To allow the app to enter text in the terminal, you need to give it permissions in System Preferences > Security & Privacy > Privacy > Accessibility. Click the lock icon in the bottom left, enter your password, and then add the app to the list.