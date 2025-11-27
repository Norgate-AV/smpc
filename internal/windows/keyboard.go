package windows

import (
	"fmt"
	"time"
)

func SendF12() bool {
	fmt.Println("[DEBUG] Trying keybd_event approach...")

	// VK_F12 = 0x7B
	vkCode := uintptr(0x7B)

	// keybd_event(vk, scan, flags, extraInfo)
	// Key down
	fmt.Println("[DEBUG] Sending keybd_event KEYDOWN")
	procKeybd_event.Call(vkCode, 0, 0x1, 0) // KEYEVENTF_EXTENDEDKEY

	time.Sleep(50 * time.Millisecond)

	// Key up
	fmt.Println("[DEBUG] Sending keybd_event KEYUP")
	procKeybd_event.Call(vkCode, 0, 0x1|0x2, 0) // KEYEVENTF_EXTENDEDKEY | KEYEVENTF_KEYUP

	fmt.Println("[DEBUG] keybd_event succeeded")
	return true
}

func SendAltF12() bool {
	fmt.Println("[DEBUG] Sending Alt+F12 via keybd_event...")

	// VK_MENU (Alt) = 0x12
	// VK_F12 = 0x7B
	vkAlt := uintptr(0x12)
	vkF12 := uintptr(0x7B)

	// Press Alt down
	fmt.Println("[DEBUG] Sending Alt KEYDOWN")
	procKeybd_event.Call(vkAlt, 0, 0x1, 0) // KEYEVENTF_EXTENDEDKEY
	time.Sleep(50 * time.Millisecond)

	// Press F12 down
	fmt.Println("[DEBUG] Sending F12 KEYDOWN")
	procKeybd_event.Call(vkF12, 0, 0x1, 0) // KEYEVENTF_EXTENDEDKEY
	time.Sleep(50 * time.Millisecond)

	// Release F12
	fmt.Println("[DEBUG] Sending F12 KEYUP")
	procKeybd_event.Call(vkF12, 0, 0x1|0x2, 0) // KEYEVENTF_EXTENDEDKEY | KEYEVENTF_KEYUP
	time.Sleep(50 * time.Millisecond)

	// Release Alt
	fmt.Println("[DEBUG] Sending Alt KEYUP")
	procKeybd_event.Call(vkAlt, 0, 0x1|0x2, 0) // KEYEVENTF_EXTENDEDKEY | KEYEVENTF_KEYUP

	fmt.Println("[DEBUG] Alt+F12 keybd_event succeeded")
	return true
}

func SendEnter() bool {
	// VK_RETURN = 0x0D
	vkCode := uintptr(0x0D)
	fmt.Println("[DEBUG] Sending Enter via keybd_event")
	procKeybd_event.Call(vkCode, 0, 0x1, 0)
	time.Sleep(50 * time.Millisecond)
	procKeybd_event.Call(vkCode, 0, 0x1|0x2, 0)
	return true
}
