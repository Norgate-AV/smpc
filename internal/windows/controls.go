package windows

import (
	"fmt"
	"strings"
	"syscall"
	"unsafe"
)

// collectChildInfos returns a slice of childInfo for all child controls of hwnd
func CollectChildInfos(hwnd uintptr) []ChildInfo {
	infos := []ChildInfo{}
	var cb func(hwnd uintptr, lparam uintptr) uintptr

	cb = func(chWnd uintptr, lparam uintptr) uintptr {
		c := GetClassName(chWnd)

		var t string

		switch c {
		case "Edit":
			t = GetEditText(chWnd)
		case "ListBox":
			// For ListBox, get all items and store them directly
			items := GetListBoxItems(chWnd)
			t = strings.Join(items, "\n") // Still join for text field for backward compatibility
			infos = append(infos, ChildInfo{Hwnd: chWnd, ClassName: c, Text: t, Items: items})
			return 1
		default:
			t = GetWindowText(chWnd)
		}

		infos = append(infos, ChildInfo{Hwnd: chWnd, ClassName: c, Text: t})
		return 1
	}

	procEnumChildWindows.Call(hwnd, syscall.NewCallback(cb), 0)
	return infos
}

func GetListBoxItems(hwnd uintptr) []string {
	// Get the count of items in the ListBox
	countResult, _, _ := procSendMessageW.Call(hwnd, LB_GETCOUNT, 0, 0)
	count := int(countResult)
	fmt.Printf("[DEBUG] getListBoxItems: hwnd=%d, count=%d\n", hwnd, count)

	if count <= 0 {
		return nil
	}

	items := make([]string, 0, count)
	for i := range count {
		// Get the length of this item
		lenResult, _, _ := procSendMessageW.Call(hwnd, LB_GETTEXTLEN, uintptr(i), 0)
		itemLen := int(lenResult)

		if itemLen <= 0 {
			continue
		}

		// Allocate buffer and get the text
		buf := make([]uint16, itemLen+1)
		procSendMessageW.Call(hwnd, LB_GETTEXT, uintptr(i), uintptr(unsafe.Pointer(&buf[0])))
		text := syscall.UTF16ToString(buf)
		fmt.Printf("[DEBUG] getListBoxItems: item[%d]=%q\n", i, text)
		items = append(items, text)
	}

	return items
}

func GetEditText(hwnd uintptr) string {
	// Get the length of the text using SendMessageW directly
	lengthResult, _, _ := procSendMessageW.Call(hwnd, WM_GETTEXTLENGTH, 0, 0)
	length := int(lengthResult)
	fmt.Printf("[DEBUG] getEditText: hwnd=%d, length=%d\n", hwnd, length)
	if length == 0 {
		return ""
	}
	// Allocate buffer (add extra space for safety)
	buf := make([]uint16, length+256)
	result, _, _ := procSendMessageW.Call(hwnd, WM_GETTEXT, uintptr(len(buf)), uintptr(unsafe.Pointer(&buf[0])))
	fmt.Printf("[DEBUG] getEditText: SendMessage returned %d\n", result)
	text := syscall.UTF16ToString(buf)
	fmt.Printf("[DEBUG] getEditText: extracted text length=%d, text=%q\n", len(text), text)
	return text
}

// findAndClickButton finds a button child control with the specified text and clicks it
// Returns true if the button was found and clicked, false otherwise
func FindAndClickButton(parentHwnd uintptr, buttonText string) bool {
	childInfos := CollectChildInfos(parentHwnd)

	for _, ci := range childInfos {
		if ci.ClassName == "Button" && strings.EqualFold(ci.Text, buttonText) {
			fmt.Printf("[DEBUG] Found button %q with hwnd=%d, sending click\n", buttonText, ci.Hwnd)
			// Send BN_CLICKED notification to parent
			// WM_COMMAND: wParam = MAKEWPARAM(controlID, BN_CLICKED), lParam = hwnd
			procSendMessageW.Call(parentHwnd, WM_COMMAND, uintptr(BN_CLICKED), ci.Hwnd)
			return true
		}
	}

	fmt.Printf("[DEBUG] Button %q not found\n", buttonText)
	return false
}

func CollectChildTexts(hwnd uintptr) []string {
	texts := []string{}

	// inner callback captures texts
	var cb func(hwnd uintptr, lparam uintptr) uintptr

	cb = func(chWnd uintptr, lparam uintptr) uintptr {
		t := GetWindowText(chWnd)
		if t != "" {
			texts = append(texts, t)
		}

		// continue enumeration
		return 1
	}

	procEnumChildWindows.Call(hwnd, syscall.NewCallback(cb), 0)
	return texts
}
