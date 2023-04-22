package main

import (
	"log"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

type KEYBDINPUT struct {
	Vk        uint16
	Scan      uint16
	Flags     uint32
	Time      uint32
	ExtraInfo uintptr
}

type INPUT struct {
	Type    uint32
	Ki      KEYBDINPUT
	Padding [8]byte
}

const (
	INPUT_KEYBOARD        = 1
	KEYEVENTF_EXTENDEDKEY = 0x0001
	KEYEVENTF_KEYUP       = 0x0002
	KEYEVENTF_SCANCODE    = 0x0008
	KEYEVENTF_UNICODE     = 0x0004
	MAX_KEYBOARD_LAYOUT   = 10
	Enter                 = 0x0D // VK_RETURN
)

func NewWindowsKeyboard() *WindowsKeyboard {
	return &WindowsKeyboard{}
}

func (k *WindowsKeyboard) utf16FromString(s string) []uint16 {
	runes := utf16.Encode([]rune(s))
	return append(runes, uint16(0))
}

func (k *WindowsKeyboard) sendInputs(inputs []INPUT) {
	user32 := syscall.NewLazyDLL("user32.dll")
	sendInput := user32.NewProc("SendInput")

	count := uintptr(len(inputs))
	size := uintptr(unsafe.Sizeof(inputs[0]))

	ret, _, err := sendInput.Call(count, uintptr(unsafe.Pointer(&inputs[0])), size)
	if int(ret) == 0 {
		log.Println("Sending inputs failed.")
		log.Fatal("Error:", err)
	}
}

func (k *WindowsKeyboard) sendChar(key uint16) {
	inputs := []INPUT{
		{Type: INPUT_KEYBOARD, Ki: KEYBDINPUT{Flags: KEYEVENTF_UNICODE, Scan: key}},
		{Type: INPUT_KEYBOARD, Ki: KEYBDINPUT{Flags: KEYEVENTF_UNICODE | KEYEVENTF_KEYUP, Scan: key}},
	}

	k.sendInputs(inputs)
}

func (k *WindowsKeyboard) SendNewLine() {
	inputs := []INPUT{
		{Type: INPUT_KEYBOARD, Ki: KEYBDINPUT{Vk: Enter}},
		{Type: INPUT_KEYBOARD, Ki: KEYBDINPUT{Vk: Enter, Flags: KEYEVENTF_KEYUP}},
	}

	k.sendInputs(inputs)
}

func (k *WindowsKeyboard) SendString(s string) {
	for _, r := range s {
		if r == 10 {
			k.SendNewLine()
			continue
		}
		k.sendChar(uint16(r))
	}
}
