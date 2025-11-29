package windows

import (
	"log/slog"
	"strings"
	"syscall"
	"unsafe"
)

// collectChildInfos returns a slice of childInfo for all child controls of hwnd
func CollectChildInfos(hwnd uintptr) []ChildInfo {
	infos := []ChildInfo{}
	cb := func(chWnd uintptr, lparam uintptr) uintptr {
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

	_, _, _ = procEnumChildWindows.Call(hwnd, syscall.NewCallback(cb), 0)
	return infos
}

func GetListBoxItems(hwnd uintptr) []string {
	// Get the count of items in the ListBox
	countResult, _, _ := procSendMessageW.Call(hwnd, LB_GETCOUNT, 0, 0)
	count := int(countResult)
	slog.Debug("getListBoxItems", "hwnd", hwnd, "count", count)

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
		var buf [256]uint16
		_, _, _ = procSendMessageW.Call(hwnd, LB_GETTEXT, uintptr(i), uintptr(unsafe.Pointer(&buf[0])))
		text := syscall.UTF16ToString(buf[:])
		slog.Debug("getListBoxItems item", "index", i, "text", text)
		items = append(items, text)
	}

	return items
}

func GetEditText(hwnd uintptr) string {
	// Get the length of the text using SendMessageW directly
	lengthResult, _, _ := procSendMessageW.Call(hwnd, WM_GETTEXTLENGTH, 0, 0)
	length := int(lengthResult)
	slog.Debug("getEditText", "hwnd", hwnd, "length", length)
	if length == 0 {
		return ""
	}
	// Allocate buffer (add extra space for safety)
	buf := make([]uint16, length+256)
	result, _, _ := procSendMessageW.Call(hwnd, WM_GETTEXT, uintptr(len(buf)), uintptr(unsafe.Pointer(&buf[0])))
	slog.Debug("getEditText SendMessage result", "result", result)
	text := syscall.UTF16ToString(buf)
	slog.Debug("getEditText extracted", "length", len(text), "text", text)
	return text
}

// findAndClickButton finds a button child control with the specified text and clicks it
// Returns true if the button was found and clicked, false otherwise
func FindAndClickButton(parentHwnd uintptr, buttonText string) bool {
	childInfos := CollectChildInfos(parentHwnd)

	for _, ci := range childInfos {
		if ci.ClassName == "Button" && strings.EqualFold(ci.Text, buttonText) {
			slog.Debug("Found button, sending click", "text", buttonText, "hwnd", ci.Hwnd)
			// Send BN_CLICKED notification to parent
			// WM_COMMAND: wParam = MAKEWPARAM(controlID, BN_CLICKED), lParam = hwnd
			_, _, _ = procSendMessageW.Call(parentHwnd, WM_COMMAND, uintptr(BN_CLICKED), ci.Hwnd)
			return true
		}
	}

	slog.Debug("Button not found", "text", buttonText)
	return false
}

func CollectChildTexts(hwnd uintptr) []string {
	texts := []string{}

	// inner callback captures texts
	cb := func(chWnd uintptr, lparam uintptr) uintptr {
		t := GetWindowText(chWnd)
		if t != "" {
			texts = append(texts, t)
		}

		// continue enumeration
		return 1
	}

	_, _, _ = procEnumChildWindows.Call(hwnd, syscall.NewCallback(cb), 0)
	return texts
}
