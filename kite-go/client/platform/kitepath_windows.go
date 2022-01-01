package platform

import (
	"path/filepath"
	"syscall"
	"unsafe"
)

var (
	shell32                  = syscall.NewLazyDLL("shell32.dll")
	ole32                    = syscall.NewLazyDLL("ole32.dll")
	procSHGetKnownFolderPath = shell32.NewProc("SHGetKnownFolderPath")
	procCoTaskMemFree        = ole32.NewProc("CoTaskMemFree")

	// GUIDs are from here:
	//   https://msdn.microsoft.com/en-us/library/windows/desktop/dd378457(v=vs.85).aspx
	// They are unpacked manually into the syscall.GUID struct.

	// FOLDERID_LocalAppData: {F1B32785-6FBA-4FCF-9D55-7B8E7F157091}
	folderIDLocalAppData = syscall.GUID{
		0xF1B32785,
		0x6FBA,
		0x4FCF,
		[8]byte{0x9D, 0x55, 0x7B, 0x8E, 0x7F, 0x15, 0x70, 0x91},
	}

	// FOLDERID_RoamingAppData: {3EB685DB-65F9-4CF6-A03A-E3EF65729F3D}
	folderIDRoamingAppData = syscall.GUID{
		0x3EB685DB,
		0x65F9,
		0x4CF6,
		[8]byte{0xA0, 0x3A, 0xE3, 0xEF, 0x65, 0x72, 0x9F, 0x3D},
	}
)

func knownFolderPath(folderID syscall.GUID) string {
	var pwstr uintptr

	// NOTE(vincent): ignore the returned HRESULT, because, according to https://msdn.microsoft.com/en-us/library/windows/desktop/bb762188(v=vs.85).aspx
	//  - the E_FAIL error can't be returned since the rfid we pass is static and well-defined.
	//  - the E_INVALIDARG error, as far as I know, can't be returned either since Roaming/AppData is always there.
	procSHGetKnownFolderPath.Call(
		uintptr(unsafe.Pointer(&folderID)),
		uintptr(uint32(0)),
		uintptr(unsafe.Pointer(nil)),
		uintptr(unsafe.Pointer(&pwstr)),
	)
	defer procCoTaskMemFree.Call(pwstr)

	return utf16PtrToString(pwstr)
}

func utf16PtrToString(str uintptr) string {
	// TODO(vincent): see if we can do anything about go vet complaining
	ptr := unsafe.Pointer(str)
	return syscall.UTF16ToString((*[1 << 16]uint16)(ptr)[:])
}

// kiteRoot returns the directory containing kite configuration and session files.
func kiteRoot() string {
	return filepath.Join(knownFolderPath(folderIDLocalAppData), "Kite")
}
