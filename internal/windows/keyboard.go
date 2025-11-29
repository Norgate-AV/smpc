package windows

import (
	"log/slog"
	"time"
)

func SendF12() bool {
	slog.Debug("Trying keybd_event approach")

	// VK_F12 = 0x7B
	vkCode := uintptr(0x7B)

	// keybd_event(vk, scan, flags, extraInfo)
	// Key down
	slog.Debug("Sending keybd_event KEYDOWN")
	_, _, _ = procKeybd_event.Call(vkCode, 0, 0x1, 0) // KEYEVENTF_EXTENDEDKEY

	time.Sleep(50 * time.Millisecond)

	// Key up
	slog.Debug("Sending keybd_event KEYUP")
	_, _, _ = procKeybd_event.Call(vkCode, 0, 0x1|0x2, 0) // KEYEVENTF_EXTENDEDKEY | KEYEVENTF_KEYUP

	slog.Debug("keybd_event succeeded")
	return true
}

func SendAltF12() bool {
	slog.Debug("Sending Alt+F12 via keybd_event")

	// VK_MENU (Alt) = 0x12
	// VK_F12 = 0x7B
	vkAlt := uintptr(0x12)
	vkF12 := uintptr(0x7B)

	// Press Alt down
	slog.Debug("Sending Alt KEYDOWN")
	_, _, _ = procKeybd_event.Call(vkAlt, 0, 0x1, 0) // KEYEVENTF_EXTENDEDKEY
	time.Sleep(50 * time.Millisecond)

	// Press F12 down
	slog.Debug("Sending F12 KEYDOWN")
	_, _, _ = procKeybd_event.Call(vkF12, 0, 0x1, 0) // KEYEVENTF_EXTENDEDKEY
	time.Sleep(50 * time.Millisecond)

	// Release F12
	slog.Debug("Sending F12 KEYUP")
	_, _, _ = procKeybd_event.Call(vkF12, 0, 0x1|0x2, 0) // KEYEVENTF_EXTENDEDKEY | KEYEVENTF_KEYUP
	time.Sleep(50 * time.Millisecond)

	// Release Alt
	slog.Debug("Sending Alt KEYUP")
	_, _, _ = procKeybd_event.Call(vkAlt, 0, 0x1|0x2, 0) // KEYEVENTF_EXTENDEDKEY | KEYEVENTF_KEYUP

	slog.Debug("Alt+F12 keybd_event succeeded")
	return true
}

func SendEnter() bool {
	// VK_RETURN = 0x0D
	vkCode := uintptr(0x0D)
	slog.Debug("Sending Enter via keybd_event")
	_, _, _ = procKeybd_event.Call(vkCode, 0, 0x1, 0)
	time.Sleep(50 * time.Millisecond)
	_, _, _ = procKeybd_event.Call(vkCode, 0, 0x1|0x2, 0)
	return true
}
