package main

import (
	"os"
	"os/exec"
	"strings"

	"github.com/shirou/gopsutil/process"
)

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
			return "Error getting PowerShell version"
		}
		versionCmdOutputString := string(versionCmdOutput)
		versionOutput = &versionCmdOutputString
	default:
		versionCmd := exec.Command(shell, "--version")
		versionCmdOutput, err := versionCmd.Output()
		if err != nil {
			return "Error getting shell version"
		}
		versionCmdOutputString := string(versionCmdOutput)
		versionOutput = &versionCmdOutputString
	}
	return strings.TrimSpace(*versionOutput)
}
