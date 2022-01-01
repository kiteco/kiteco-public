package fs

import (
	"log"

	"github.com/winlabs/gowin32"
)

var (
	programFiles    string
	programFilesX86 string
	localAppData    string
	roamingAppData  string
)

func init() {
	var err error

	if programFiles, err = gowin32.GetKnownFolderPath(gowin32.KnownFolderProgramFiles); err != nil {
		log.Println("error retrieving programFiles path", err)
	}
	if programFilesX86, err = gowin32.GetKnownFolderPath(gowin32.KnownFolderProgramFilesX86); err != nil {
		log.Println("error retrieving programFilesX86 path", err)
	}
	if localAppData, err = gowin32.GetKnownFolderPath(gowin32.KnownFolderLocalAppData); err != nil {
		log.Println("error retrieving localAppData path", err)
	}
	if roamingAppData, err = gowin32.GetKnownFolderPath(gowin32.KnownFolderRoamingAppData); err != nil {
		log.Println("error retrieving roamingAppData path", err)
	}
}

// LocalAppData returns a path of the Local App Data folder.
func LocalAppData() string {
	return localAppData
}

// ProgramFiles returns a path of the Program Files folder.
func ProgramFiles() string {
	return programFiles
}

// ProgramFilesX86 returns the path of the Program Files X86 folder.
func ProgramFilesX86() string {
	return programFilesX86
}

// RoamingAppData returns the path of the system Roaming App Data folder.
func RoamingAppData() string {
	return roamingAppData
}
