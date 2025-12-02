package windows

import (
	"syscall"
	"time"

	"github.com/Norgate-AV/smpc/internal/logger"
)

const (
	WM_GETTEXT       = 0x000D
	WM_GETTEXTLENGTH = 0x000E
	LB_GETCOUNT      = 0x018B
	LB_GETTEXT       = 0x0189
	LB_GETTEXTLEN    = 0x018A
)

var (
	shell32                      = syscall.NewLazyDLL("shell32.dll")
	procShellExecute             = shell32.NewProc("ShellExecuteW")
	procShellExecuteEx           = shell32.NewProc("ShellExecuteExW")
	kernel32                     = syscall.NewLazyDLL("kernel32.dll")
	ProcCreateToolhelp32Snapshot = kernel32.NewProc("CreateToolhelp32Snapshot")
	ProcProcess32First           = kernel32.NewProc("Process32FirstW")
	ProcProcess32Next            = kernel32.NewProc("Process32NextW")
	ProcCloseHandle              = kernel32.NewProc("CloseHandle")
	procGetCurrentProcess        = kernel32.NewProc("GetCurrentProcess")
	procGetProcessId             = kernel32.NewProc("GetProcessId")
	procOpenProcessToken         = kernel32.NewProc("OpenProcessToken")
	procOpenProcess              = kernel32.NewProc("OpenProcess")
	procTerminateProcess         = kernel32.NewProc("TerminateProcess")
	advapi32                     = syscall.NewLazyDLL("advapi32.dll")
	procGetTokenInformation      = advapi32.NewProc("GetTokenInformation")
	user32                       = syscall.NewLazyDLL("user32.dll")
	procEnumWindows              = user32.NewProc("EnumWindows")
	procGetWindowTextW           = user32.NewProc("GetWindowTextW")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procIsWindowVisible          = user32.NewProc("IsWindowVisible")
	ProcSendMessageTimeoutW      = user32.NewProc("SendMessageTimeoutW")
	procSendMessageW             = user32.NewProc("SendMessageW")
	procPostMessageW             = user32.NewProc("PostMessageW")
	procSetForegroundWindow      = user32.NewProc("SetForegroundWindow")
	procGetForegroundWindow      = user32.NewProc("GetForegroundWindow")
	procKeybd_event              = user32.NewProc("keybd_event")
	procSendInput                = user32.NewProc("SendInput")
	procShowWindow               = user32.NewProc("ShowWindow")
	procEnumChildWindows         = user32.NewProc("EnumChildWindows")
	procGetClassNameW            = user32.NewProc("GetClassNameW")
)

const (
	WM_NULL          = 0x0000
	WM_CLOSE         = 0x0010
	WM_COMMAND       = 0x0111
	WM_KEYDOWN       = 0x0100
	WM_KEYUP         = 0x0101
	WM_SYSKEYDOWN    = 0x0104
	WM_SYSKEYUP      = 0x0105
	SMTO_ABORTIFHUNG = 0x0002
	SMTO_BLOCK       = 0x0003
	BN_CLICKED       = 0

	INPUT_KEYBOARD        = 1
	KEYEVENTF_SCANCODE    = 0x0008
	KEYEVENTF_KEYUP       = 0x0002
	KEYEVENTF_EXTENDEDKEY = 0x0001

	VK_MENU   = 0x12 // Alt key
	VK_F12    = 0x7B
	VK_RETURN = 0x0D

	SC_F12     = 0x58
	SW_RESTORE = 9
	GW_CHILD   = 5

	TOKEN_QUERY    = 0x0008
	TokenElevation = 20
)

const (
	TH32CS_SNAPPROCESS = 0x00000002
	MAX_PATH           = 260
)

// WindowsAPI is a concrete implementation of all Windows-related interfaces
// It wraps a Client to provide the required functionality
type WindowsAPI struct {
	client *Client
}

// NewWindowsAPI creates a new WindowsAPI with the provided logger
func NewWindowsAPI(log logger.LoggerInterface) *WindowsAPI {
	return &WindowsAPI{
		client: NewClient(log),
	}
}

// WindowManager interface implementation
func (w *WindowsAPI) CloseWindow(hwnd uintptr, title string) {
	w.client.Window.CloseWindow(hwnd, title)
}
func (w *WindowsAPI) SetForeground(hwnd uintptr) bool { return w.client.Window.SetForeground(hwnd) }
func (w *WindowsAPI) VerifyForegroundWindow(expectedHwnd uintptr, expectedPid uint32) bool {
	return w.client.Window.VerifyForegroundWindow(expectedHwnd, expectedPid)
}
func (w *WindowsAPI) IsElevated() bool { return w.client.Window.IsElevated() }
func (w *WindowsAPI) CollectChildInfos(hwnd uintptr) []ChildInfo {
	return w.client.Window.CollectChildInfos(hwnd)
}

func (w *WindowsAPI) WaitOnMonitor(timeout time.Duration, matchers ...func(WindowEvent) bool) (WindowEvent, bool) {
	return w.client.Window.WaitOnMonitor(timeout, matchers...)
}

// KeyboardInjector interface implementation
func (w *WindowsAPI) SendF12()    { w.client.Keyboard.SendF12() }
func (w *WindowsAPI) SendAltF12() { w.client.Keyboard.SendAltF12() }
func (w *WindowsAPI) SendEnter()  { w.client.Keyboard.SendEnter() }
func (w *WindowsAPI) SendF12ToWindow(hwnd uintptr) bool {
	return w.client.Keyboard.SendF12ToWindow(hwnd)
}

func (w *WindowsAPI) SendAltF12ToWindow(hwnd uintptr) bool {
	return w.client.Keyboard.SendAltF12ToWindow(hwnd)
}

func (w *WindowsAPI) SendF12WithSendInput() bool {
	return w.client.Keyboard.SendF12WithSendInput()
}

func (w *WindowsAPI) SendAltF12WithSendInput() bool {
	return w.client.Keyboard.SendAltF12WithSendInput()
}

// ControlReader interface implementation
func (w *WindowsAPI) GetListBoxItems(hwnd uintptr) []string { return GetListBoxItems(hwnd) }
func (w *WindowsAPI) GetEditText(hwnd uintptr) string       { return GetEditText(hwnd) }
func (w *WindowsAPI) FindAndClickButton(parentHwnd uintptr, buttonText string) bool {
	return w.client.Window.FindAndClickButton(parentHwnd, buttonText)
}
