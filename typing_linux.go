//go:build linux

package main

type LinuxKeyboard struct{}

func NewKeyboard() *LinuxKeyboard {
	return &LinuxKeyboard{}
}

func (k *LinuxKeyboard) IsFocusTheSame() bool {
	return true
}

func (k *LinuxKeyboard) SendString(key string) {
	panic("Typing is not implemented on Linux")
}

func (k *LinuxKeyboard) SendNewLine() {
	panic("Typing is not implemented on Linux")
}
