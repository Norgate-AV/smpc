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

func SendEnter() bool {
	// VK_RETURN = 0x0D
	vkCode := uintptr(0x0D)
	fmt.Println("[DEBUG] Sending Enter via keybd_event")
	procKeybd_event.Call(vkCode, 0, 0x1, 0)
	time.Sleep(50 * time.Millisecond)
	procKeybd_event.Call(vkCode, 0, 0x1|0x2, 0)
	return true
}
