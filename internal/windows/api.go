package windows

import "syscall"

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
	kernel32                     = syscall.NewLazyDLL("kernel32.dll")
	ProcCreateToolhelp32Snapshot = kernel32.NewProc("CreateToolhelp32Snapshot")
	ProcProcess32First           = kernel32.NewProc("Process32FirstW")
	ProcProcess32Next            = kernel32.NewProc("Process32NextW")
	ProcCloseHandle              = kernel32.NewProc("CloseHandle")
	procGetCurrentProcess        = kernel32.NewProc("GetCurrentProcess")
	procOpenProcessToken         = kernel32.NewProc("OpenProcessToken")
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
	SMTO_ABORTIFHUNG = 0x0002
	SMTO_BLOCK       = 0x0003
	BN_CLICKED       = 0

	INPUT_KEYBOARD        = 1
	KEYEVENTF_SCANCODE    = 0x0008
	KEYEVENTF_KEYUP       = 0x0002
	KEYEVENTF_EXTENDEDKEY = 0x0001

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
