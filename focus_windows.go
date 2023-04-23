//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

var (
	getForegroundWin   = user32DLL.MustFindProc("GetForegroundWindow")
	getWindowThread    = user32DLL.MustFindProc("GetWindowThreadProcessId")
	getFocus           = user32DLL.MustFindProc("GetFocus")
	attachThreadInput  = user32DLL.MustFindProc("AttachThreadInput")
	getCurrentThreadId = kernel32DLL.MustFindProc("GetCurrentThreadId")
	getGUIThreadInfo   = user32DLL.MustFindProc("GetGUIThreadInfo")
)

// Add a struct for GUITHREADINFO
type GUITHREADINFO struct {
	CbSize        uint32
	Flags         uint32
	HwndActive    uintptr
	HwndFocus     uintptr
	HwndCapture   uintptr
	HwndMenuOwner uintptr
	HwndMoveSize  uintptr
	HwndCaret     uintptr
	RcCaret       RECT
}

type RECT struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

type FocusedControl struct {
	WindowHandle   uintptr
	WindowThreadId uintptr
	Control        uintptr
}

func (c FocusedControl) Is(other FocusedControl) bool {
	return c.WindowHandle == other.WindowHandle &&
		c.WindowThreadId == other.WindowThreadId &&
		c.Control == other.Control
}

func GetForegroundWindow() uintptr {
	handle, _, _ := getForegroundWin.Call()
	return handle
}

func GetCurrentThreadId() uintptr {
	currentThreadId, _, _ := getCurrentThreadId.Call()
	return currentThreadId
}
func AttachThreadInput(currentThreadId, targetThreadId uintptr, attach bool) bool {
	attachBool := 0
	if attach {
		attachBool = 1
	}

	ret, _, _ := attachThreadInput.Call(currentThreadId, targetThreadId, uintptr(attachBool))
	return ret != 0
}

func GetWindowThreadId(hwnd uintptr) uintptr {
	var processID uintptr
	_, _, _ = getWindowThread.Call(hwnd, uintptr(unsafe.Pointer(&processID)))
	return processID
}

func GetGUIThreadInfo(threadId uintptr, info *GUITHREADINFO) bool {
	ret, _, _ := syscall.Syscall(getGUIThreadInfo.Addr(), 2, uintptr(threadId), uintptr(unsafe.Pointer(info)), 0)
	return ret != 0
}

func GetFocusedControl() FocusedControl {
	hwnd := GetForegroundWindow()
	windowThreadId := GetWindowThreadId(hwnd)
	currentThreadId := GetCurrentThreadId()

	AttachThreadInput(currentThreadId, uintptr(windowThreadId), true)

	info := GUITHREADINFO{CbSize: uint32(unsafe.Sizeof(GUITHREADINFO{}))}
	GetGUIThreadInfo(windowThreadId, &info)
	focusedControlHandle := info.HwndFocus

	AttachThreadInput(currentThreadId, uintptr(windowThreadId), false)

	control := FocusedControl{
		WindowHandle:   hwnd,
		WindowThreadId: windowThreadId,
		Control:        focusedControlHandle,
	}
	//log.Println("Focused control:", control)
	return control
}
