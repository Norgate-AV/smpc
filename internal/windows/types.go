package windows

type TOKEN_ELEVATION struct {
	TokenIsElevated uint32
}

// childInfo and collectChildInfos moved from collect_child_infos.go for single-file build
type ChildInfo struct {
	Hwnd      uintptr
	ClassName string
	Text      string
	Items     []string // For ListBox controls, stores items directly
}

// Structures for SendInput
type KEYBDINPUT struct {
	WVk         uint16
	WScan       uint16
	DwFlags     uint32
	Time        uint32
	DwExtraInfo uintptr
}

type MOUSEINPUT struct {
	Dx, Dy      int32
	MouseData   uint32
	DwFlags     uint32
	Time        uint32
	DwExtraInfo uintptr
}

type HARDWAREINPUT struct {
	UMsg    uint32
	WParamL uint16
	WParamH uint16
}

type INPUT struct {
	Type uint32
	_    [4]byte  // Padding to align to 8 bytes
	Data [32]byte // Union data (largest is MOUSEINPUT at 24 bytes, padded to 32)
}

type PROCESSENTRY32 struct {
	DwSize              uint32
	CntUsage            uint32
	Th32ProcessID       uint32
	Th32DefaultHeapID   uintptr
	Th32ModuleID        uint32
	CntThreads          uint32
	Th32ParentProcessID uint32
	PcPriClassBase      int32
	DwFlags             uint32
	SzExeFile           [MAX_PATH]uint16
}

type WindowInfo struct {
	Hwnd  uintptr
	Title string
	Pid   uint32
}

type WindowEvent struct {
	Hwnd  uintptr
	Title string
	Pid   uint32
	Class string
}

// SHELLEXECUTEINFO for ShellExecuteEx API
type SHELLEXECUTEINFO struct {
	CbSize       uint32
	FMask        uint32
	Hwnd         uintptr
	LpVerb       *uint16
	LpFile       *uint16
	LpParameters *uint16
	LpDirectory  *uint16
	NShow        int32
	HInstApp     uintptr
	LpIDList     uintptr
	LpClass      *uint16
	HkeyClass    uintptr
	DwHotKey     uint32
	HIcon        uintptr
	HProcess     uintptr
}
