//go:build windows

package windows

import (
	"strings"
	"syscall"
	"unsafe"
)

// ControlExtractor is a function that extracts text and items from a specific control type
type ControlExtractor func(hwnd uintptr) (text string, items []string)

// controlExtractors is a table-driven map of control-specific extraction logic
// This makes it easy to add new control types without modifying CollectChildInfos
var controlExtractors = map[string]ControlExtractor{
	"Edit": func(hwnd uintptr) (string, []string) {
		return GetEditText(hwnd), nil
	},
	"ListBox": func(hwnd uintptr) (string, []string) {
		items := GetListBoxItems(hwnd)
		// Join for text field for backward compatibility
		return strings.Join(items, "\n"), items
	},
}

// extractControlInfo extracts information from a control using the appropriate extractor
func extractControlInfo(hwnd uintptr, className string) ChildInfo {
	extractor, exists := controlExtractors[className]
	if !exists {
		// Default extractor for unknown control types
		return ChildInfo{
			Hwnd:      hwnd,
			ClassName: className,
			Text:      GetWindowText(hwnd),
		}
	}

	text, items := extractor(hwnd)
	return ChildInfo{
		Hwnd:      hwnd,
		ClassName: className,
		Text:      text,
		Items:     items,
	}
}

// CollectChildInfos returns a slice of childInfo for all child controls of hwnd
func CollectChildInfos(hwnd uintptr) []ChildInfo {
	infos := []ChildInfo{}

	cb := func(chWnd uintptr, lparam uintptr) uintptr {
		className := GetClassName(chWnd)
		info := extractControlInfo(chWnd, className)
		infos = append(infos, info)
		return 1
	}

	// EnumChildWindows: return value indicates success but errors aren't meaningful here
	// The callback approach makes individual error logging impractical
	_, _, _ = procEnumChildWindows.Call(hwnd, syscall.NewCallback(cb), 0)
	return infos
}

// GetListBoxItems retrieves all items from a ListBox control
func GetListBoxItems(hwnd uintptr) []string {
	// Get the count of items in the ListBox
	countResult, _, _ := procSendMessageW.Call(hwnd, LB_GETCOUNT, 0, 0)
	count := int(countResult)

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
		items = append(items, text)
	}

	return items
}

// GetEditText retrieves the text from an Edit control
func GetEditText(hwnd uintptr) string {
	// Get the length of the text using SendMessageW directly
	lengthResult, _, _ := procSendMessageW.Call(hwnd, WM_GETTEXTLENGTH, 0, 0)
	length := int(lengthResult)

	if length == 0 {
		return ""
	}

	// Allocate buffer (add extra space for safety)
	buf := make([]uint16, length+256)
	_, _, _ = procSendMessageW.Call(hwnd, WM_GETTEXT, uintptr(len(buf)), uintptr(unsafe.Pointer(&buf[0])))
	return syscall.UTF16ToString(buf)
}

// CollectChildTexts retrieves the text of all child windows
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
