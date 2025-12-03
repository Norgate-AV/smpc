//go:build windows

package windows

import (
	"log/slog"
	"time"
	"unsafe"

	"github.com/Norgate-AV/smpc/internal/logger"
)

// keyboardInjector implements the KeyboardInjector interface
type keyboardInjector struct {
	log logger.LoggerInterface
}

// newKeyboardInjector creates a new keyboard injector
func newKeyboardInjector(log logger.LoggerInterface) *keyboardInjector {
	return &keyboardInjector{log: log}
}

// SendF12 sends the F12 key
func (k *keyboardInjector) SendF12() {
	// VK_F12 = 0x7B
	vkCode := uintptr(0x7B)

	// keybd_event(vk, scan, flags, extraInfo)
	// Note: keybd_event has void return type, no error checking needed
	k.log.Debug("Sending F12 KEYDOWN")
	_, _, _ = procKeybd_event.Call(vkCode, 0, 0x1, 0) // KEYEVENTF_EXTENDEDKEY

	time.Sleep(50 * time.Millisecond)

	k.log.Debug("Sending F12 KEYUP")
	_, _, _ = procKeybd_event.Call(vkCode, 0, 0x1|0x2, 0) // KEYEVENTF_EXTENDEDKEY | KEYEVENTF_KEYUP
}

// SendAltF12 sends the Alt+F12 key combination
func (k *keyboardInjector) SendAltF12() {
	// VK_MENU (Alt) = 0x12
	// VK_F12 = 0x7B
	vkAlt := uintptr(0x12)
	vkF12 := uintptr(0x7B)

	// Note: keybd_event has void return type, no error checking needed
	k.log.Debug("Sending Alt KEYDOWN")
	_, _, _ = procKeybd_event.Call(vkAlt, 0, 0x1, 0) // KEYEVENTF_EXTENDEDKEY
	time.Sleep(50 * time.Millisecond)

	k.log.Debug("Sending F12 KEYDOWN")
	_, _, _ = procKeybd_event.Call(vkF12, 0, 0x1, 0) // KEYEVENTF_EXTENDEDKEY
	time.Sleep(50 * time.Millisecond)

	k.log.Debug("Sending F12 KEYUP")
	_, _, _ = procKeybd_event.Call(vkF12, 0, 0x1|0x2, 0) // KEYEVENTF_EXTENDEDKEY | KEYEVENTF_KEYUP
	time.Sleep(50 * time.Millisecond)

	k.log.Debug("Sending Alt KEYUP")
	_, _, _ = procKeybd_event.Call(vkAlt, 0, 0x1|0x2, 0) // KEYEVENTF_EXTENDEDKEY | KEYEVENTF_KEYUP
}

// SendEnter sends the Enter key
func (k *keyboardInjector) SendEnter() {
	// VK_RETURN = 0x0D
	vkCode := uintptr(0x0D)

	// Note: keybd_event has void return type, no error checking needed
	k.log.Debug("Sending Enter KEYDOWN")
	_, _, _ = procKeybd_event.Call(vkCode, 0, 0x1, 0)
	time.Sleep(50 * time.Millisecond)

	k.log.Debug("Sending Enter KEYUP")
	_, _, _ = procKeybd_event.Call(vkCode, 0, 0x1|0x2, 0)
}

// SendF12ToWindow sends F12 key directly to a specific window using SendMessage
func (k *keyboardInjector) SendF12ToWindow(hwnd uintptr) bool {
	k.log.Debug("Sending F12 to window via PostMessage", slog.Uint64("hwnd", uint64(hwnd)))

	// lParam construction for F12:
	// Bits 0-15: Repeat count (1)
	// Bits 16-23: Scan code for F12 (0x58)
	// Bit 24: Extended key flag (1 for F12)
	// Bits 25-28: Reserved (0)
	// Bit 29: Context code (0 for non-Alt)
	// Bit 30: Previous key state (0 for key down)
	// Bit 31: Transition state (0 for key down, 1 for key up)
	const scanCodeF12 = 0x58
	lParamDown := uintptr(1 | (scanCodeF12 << 16) | (1 << 24))                       // Extended key flag set
	lParamUp := uintptr(1 | (scanCodeF12 << 16) | (1 << 24) | (1 << 30) | (1 << 31)) // Previous state + transition

	// Try SendMessage first (synchronous)
	k.log.Debug("Trying SendMessage for F12")
	ret, _, _ := procSendMessageW.Call(hwnd, WM_KEYDOWN, VK_F12, lParamDown)
	k.log.Debug("SendMessage WM_KEYDOWN returned", slog.Uint64("ret", uint64(ret)))
	time.Sleep(50 * time.Millisecond)

	ret, _, _ = procSendMessageW.Call(hwnd, WM_KEYUP, VK_F12, lParamUp)
	k.log.Debug("SendMessage WM_KEYUP returned", slog.Uint64("ret", uint64(ret)))

	k.log.Debug("F12 sent via SendMessage (synchronous)")
	return true
}

// SendAltF12ToWindow sends Alt+F12 key directly to a specific window using SendMessage
func (k *keyboardInjector) SendAltF12ToWindow(hwnd uintptr) bool {
	k.log.Debug("Sending Alt+F12 to window via SendMessage", slog.Uint64("hwnd", uint64(hwnd)))

	// lParam construction
	const scanCodeAlt = 0x38
	const scanCodeF12 = 0x58

	// Alt key down (context code bit 29 = 1 for Alt)
	lParamAltDown := uintptr(1 | (scanCodeAlt << 16) | (1 << 24) | (1 << 29))

	// F12 down while Alt is held (context code bit 29 = 1)
	lParamF12Down := uintptr(1 | (scanCodeF12 << 16) | (1 << 24) | (1 << 29))

	// F12 up (transition bit 31 = 1, previous state bit 30 = 1, context code bit 29 = 1)
	lParamF12Up := uintptr(1 | (scanCodeF12 << 16) | (1 << 24) | (1 << 29) | (1 << 30) | (1 << 31))

	// Alt up (transition bit 31 = 1, previous state bit 30 = 1, context code bit 29 = 1)
	lParamAltUp := uintptr(1 | (scanCodeAlt << 16) | (1 << 24) | (1 << 29) | (1 << 30) | (1 << 31))

	// Send Alt down
	k.log.Debug("Sending WM_SYSKEYDOWN (Alt)")
	_, _, _ = procSendMessageW.Call(hwnd, WM_SYSKEYDOWN, VK_MENU, lParamAltDown)
	time.Sleep(50 * time.Millisecond)

	// Send F12 down
	k.log.Debug("Sending WM_SYSKEYDOWN (F12)")
	_, _, _ = procSendMessageW.Call(hwnd, WM_SYSKEYDOWN, VK_F12, lParamF12Down)
	time.Sleep(50 * time.Millisecond)

	// Send F12 up
	k.log.Debug("Sending WM_SYSKEYUP (F12)")
	_, _, _ = procSendMessageW.Call(hwnd, WM_SYSKEYUP, VK_F12, lParamF12Up)
	time.Sleep(50 * time.Millisecond)

	// Send Alt up
	k.log.Debug("Sending WM_SYSKEYUP (Alt)")
	_, _, _ = procSendMessageW.Call(hwnd, WM_SYSKEYUP, VK_MENU, lParamAltUp)

	k.log.Debug("Alt+F12 sent via SendMessage (synchronous)")
	return true
}

// SendF12WithSendInput sends F12 key using SendInput API (more modern than keybd_event)
func (k *keyboardInjector) SendF12WithSendInput() bool {
	k.log.Debug("Sending F12 via SendInput")

	// Create INPUT structure for keydown
	inputs := make([]INPUT, 2)

	// F12 KEYDOWN
	inputs[0].Type = INPUT_KEYBOARD
	kb := (*KEYBDINPUT)(unsafe.Pointer(&inputs[0].Data[0]))
	kb.WVk = VK_F12
	kb.DwFlags = KEYEVENTF_EXTENDEDKEY

	// F12 KEYUP
	inputs[1].Type = INPUT_KEYBOARD
	kb2 := (*KEYBDINPUT)(unsafe.Pointer(&inputs[1].Data[0]))
	kb2.WVk = VK_F12
	kb2.DwFlags = KEYEVENTF_EXTENDEDKEY | KEYEVENTF_KEYUP

	// Send the input
	ret, _, _ := procSendInput.Call(
		uintptr(len(inputs)),
		uintptr(unsafe.Pointer(&inputs[0])),
		uintptr(unsafe.Sizeof(INPUT{})),
	)

	if ret != uintptr(len(inputs)) {
		k.log.Warn("SendInput failed", slog.Uint64("expected", uint64(len(inputs))), slog.Uint64("sent", uint64(ret)))
		return false
	}

	k.log.Debug("F12 sent via SendInput successfully")
	return true
}

// SendAltF12WithSendInput sends Alt+F12 key using SendInput API
func (k *keyboardInjector) SendAltF12WithSendInput() bool {
	k.log.Debug("Sending Alt+F12 via SendInput")

	// Create INPUT structures for Alt down, F12 down, F12 up, Alt up
	inputs := make([]INPUT, 4)

	// Alt KEYDOWN
	inputs[0].Type = INPUT_KEYBOARD
	kb0 := (*KEYBDINPUT)(unsafe.Pointer(&inputs[0].Data[0]))
	kb0.WVk = VK_MENU
	kb0.DwFlags = KEYEVENTF_EXTENDEDKEY

	// F12 KEYDOWN
	inputs[1].Type = INPUT_KEYBOARD
	kb1 := (*KEYBDINPUT)(unsafe.Pointer(&inputs[1].Data[0]))
	kb1.WVk = VK_F12
	kb1.DwFlags = KEYEVENTF_EXTENDEDKEY

	// F12 KEYUP
	inputs[2].Type = INPUT_KEYBOARD
	kb2 := (*KEYBDINPUT)(unsafe.Pointer(&inputs[2].Data[0]))
	kb2.WVk = VK_F12
	kb2.DwFlags = KEYEVENTF_EXTENDEDKEY | KEYEVENTF_KEYUP

	// Alt KEYUP
	inputs[3].Type = INPUT_KEYBOARD
	kb3 := (*KEYBDINPUT)(unsafe.Pointer(&inputs[3].Data[0]))
	kb3.WVk = VK_MENU
	kb3.DwFlags = KEYEVENTF_EXTENDEDKEY | KEYEVENTF_KEYUP

	// Send all inputs
	ret, _, _ := procSendInput.Call(
		uintptr(len(inputs)),
		uintptr(unsafe.Pointer(&inputs[0])),
		uintptr(unsafe.Sizeof(INPUT{})),
	)

	if ret != uintptr(len(inputs)) {
		k.log.Warn("SendInput failed", slog.Uint64("expected", uint64(len(inputs))), slog.Uint64("sent", uint64(ret)))
		return false
	}

	k.log.Debug("Alt+F12 sent via SendInput successfully")
	return true
}
