// +build windows

package performance

/*
#cgo LDFLAGS: -lPsapi

// Copied from SO link: /questions/63166/how-to-determine-cpu-and-memory-consumption-from-inside-a-process

#ifdef _WIN32_WINNT
#undef _WIN32_WINNT
#endif

#define _WIN32_WINNT 0x0600

#include <windows.h>
#include <psapi.h>
#include <stdint.h>
#include <time.h>

static int numProcessors;
static HANDLE self;

void init() {
    SYSTEM_INFO sysInfo;
    GetSystemInfo(&sysInfo);
    numProcessors = sysInfo.dwNumberOfProcessors;
    self = GetCurrentProcess();
}

float cpuUsage() {
    FILETIME ftime, fsys, fuser;
    ULARGE_INTEGER nowStart, sysStart, userStart;
    ULARGE_INTEGER nowEnd, sysEnd, userEnd;
    double percent;

    GetSystemTimeAsFileTime(&ftime);
    memcpy(&nowStart, &ftime, sizeof(FILETIME));

    GetProcessTimes(self, &ftime, &ftime, &fsys, &fuser);
    memcpy(&sysStart, &fsys, sizeof(FILETIME));
    memcpy(&userStart, &fuser, sizeof(FILETIME));

    Sleep(1000);

    GetSystemTimeAsFileTime(&ftime);
    memcpy(&nowEnd, &ftime, sizeof(FILETIME));

    GetProcessTimes(self, &ftime, &ftime, &fsys, &fuser);
    memcpy(&sysEnd, &fsys, sizeof(FILETIME));
    memcpy(&userEnd, &fuser, sizeof(FILETIME));

    percent = (sysEnd.QuadPart - sysStart.QuadPart) + (userEnd.QuadPart - userStart.QuadPart);
    percent /= (nowEnd.QuadPart - nowStart.QuadPart);
    percent /= numProcessors;
    return percent * 100;
}

size_t memoryUsage() {
    PROCESS_MEMORY_COUNTERS pmc;
    GetProcessMemoryInfo(GetCurrentProcess(), &pmc, sizeof(pmc));
    size_t physMemUsedByMe = pmc.WorkingSetSize;
    return physMemUsedByMe;
}

*/
import "C"
import (
	"strconv"
	"syscall"
	"unsafe"

	"github.com/winlabs/gowin32"
)

var (
	k32                   = syscall.NewLazyDLL("kernel32.dll")
	procGetModuleHandle   = k32.NewProc("GetModuleHandleW")
	procGetModuleFileName = k32.NewProc("GetModuleFileNameW")
)

func init() {
	C.init()
}

// memoryUsage returns the amount of memory that the menubar is currently using
func memoryUsage() int64 {
	return int64(C.memoryUsage())
}

// osVersion returns an empty string
func osVersion() string {
	// multiple ways of doing this:
	// 1. more direct, but undocumented, and requiring custom CGO shims: SO link: /a/36545162
	// 2. another undocumented approach: https://docs.microsoft.com/en-us/windows/win32/api/lmwksta/nf-lmwksta-netwkstagetinfo
	// 3. https://docs.microsoft.com/en-us/windows/win32/sysinfo/getting-the-system-version, which outlines two approaches
	// We use one of the officially recommended approaches.
	// See also https://github.com/yaochenzhi/datadog-agent/blob/ca192a1/pkg/util/winutil/winver.go#L32

	h, err := getModuleHandle("kernel32.dll")
	if err != nil {
		return ""
	}
	fullpath, err := getModuleFileName(h)
	if err != nil {
		return ""
	}

	vData, err := gowin32.GetFileVersion(fullpath)
	if err != nil || vData == nil {
		return ""
	}
	vInfo, err := vData.GetFixedFileInfo()
	if err != nil || vInfo == nil {
		return ""
	}
	return strconv.FormatInt(int64(vInfo.ProductVersion.Major), 10)
}

func getModuleHandle(fname string) (handle uintptr, err error) {
	file := syscall.StringToUTF16Ptr(fname)
	handle, _, err = procGetModuleHandle.Call(uintptr(unsafe.Pointer(file)))
	if handle == 0 {
		return handle, err
	}
	return handle, nil
}

func getModuleFileName(h uintptr) (fname string, err error) {
	fname = ""
	err = nil
	var sizeIncr = uint32(1024)
	var size = sizeIncr
	for {
		buf := make([]uint16, size)
		ret, _, err := procGetModuleFileName.Call(h, uintptr(unsafe.Pointer(&buf[0])), uintptr(size))
		if ret == uintptr(size) || err == syscall.ERROR_INSUFFICIENT_BUFFER {
			size += sizeIncr
			continue
		} else if err != nil {
			fname = syscall.UTF16ToString(buf)
		}
		break
	}
	return
}

// cpuUsage gets the current CPU usage. It blocks for one second to sample
// times.
func cpuUsage() float64 {
	return float64(C.cpuUsage())
}
