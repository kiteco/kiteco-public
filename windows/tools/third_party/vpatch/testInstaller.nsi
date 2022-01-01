;-------------------------------------------------------------------------------
;Some defines
;-------------------------------------------------------------------------------

; Where are the files to be installed located.
!define PATCH_SOURCE_ROOT "..\DirectoryDiffTest\dir2"

; Where are the "*.pat" files.
!define PATCH_FILES_ROOT "patchFiles"

; The default installation directory
InstallDir "C:\temp\TestInstaller"

; Directory to which the files will be installed.
!define PATCH_INSTALL_ROOT $INSTDIR

!include "patch.nsi"

;-------------------------------------------------------------------------------
; Installer fundamentals...
;------------------------------------------------------------------------------- 

; The name of the installer
Name "Test patch installer"

; The file to write
OutFile "testInstaller.exe"

; Show details
ShowInstDetails show

;-------------------------------------------------------------------------------
;Stuff to install
;------------------------------------------------------------------------------- 


Section "Test Installer Core"

  SectionIn RO
  
  SetOutPath $INSTDIR
  
  Call patchFilesRemoved
  Call patchDirectoriesRemoved
  Call patchDirectoriesAdded
  Call patchFilesAdded
  Call patchFilesModified

SectionEnd


