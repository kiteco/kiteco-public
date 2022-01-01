package version

import (
	"log"

	"github.com/kardianos/osext"
	"github.com/kiteco/kiteco/kite-go/client/internal/reg"
	"github.com/winlabs/gowin32"
)

func fileInfo() (*gowin32.FixedFileInfo, error) {
	self, err := osext.Executable()
	if err != nil {
		return nil, err
	}

	buf, err := gowin32.GetFileVersion(self)
	if err != nil {
		return nil, err
	}

	return buf.GetFixedFileInfo()
}

// Version returns a string representation of the current Kite version. This
// is the string that appears, for example, in the menubar item.
func Version() string {
	info, err := fileInfo()
	if err != nil {
		log.Println("error getting version:", err)
		return "unknown"
	}
	return info.ProductVersion.String()
}

// IsDebugBuild returns true if this is a debug build (which is true whenever
// Kite was not built by the distribution scripts).
func IsDebugBuild() bool {
	if Version() == "1.0.0.0" {
		return true
	}

	info, err := fileInfo()
	if err != nil {
		log.Println("error checking for debug build:", err)
		return true
	}
	return info.FileFlags&gowin32.VerFileDebug != 0
}

// IsDevMode returns true if the Windows registry marks this as debug build
func IsDevMode() bool {
	if IsDebugBuild() {
		return true
	}
	_, err := reg.IsDebug()
	return err == nil
}
