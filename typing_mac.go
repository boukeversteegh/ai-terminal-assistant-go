//go:build darwin

package main

import (
	"fmt"
	"os/exec"
	"strings"
)

type MacKeyboard struct{}

func NewKeyboard() *MacKeyboard {
	return &MacKeyboard{}
}

func (k *MacKeyboard) IsFocusTheSame() bool {
	return true
}

func (k *MacKeyboard) SendString(key string) {
	escapedText := strings.ReplaceAll(key, `"`, `\"`)
	script := fmt.Sprintf(`tell application "System Events" to keystroke "%s"`, escapedText)

	cmd := exec.Command("osascript", "-e", script)
	err := cmd.Run()
	if err != nil {
		fmt.Printf("Error executing osascript: %v\n", err)
	}
}

func (k *MacKeyboard) SendNewLine() {
	cmd := exec.Command("osascript", "-e", `tell application "System Events" to keystroke return`)
	err := cmd.Run()
	if err != nil {
		fmt.Printf("Error executing osascript: %v\n", err)
	}
}
