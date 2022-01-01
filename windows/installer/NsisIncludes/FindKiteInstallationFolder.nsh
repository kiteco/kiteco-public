!include "NsisIncludes\GetParent.nsh"

; return value is passed via the stack (call 'Pop $0')
; return without the trailing '\'.
; return "" if not found
; this function clobbers $0
; the returned directory should contain "kited.exe"
Function FindKiteInstallationFolder
	
	; check the registry 'Run' entry
	; 
	; NOTE THAT THIS WILL FAIL IF THE UPDATER IS BEING RAN FROM THE SERVICE/'SYSTEM' ACCOUNT
	; 
	; It will also fail if the user disabled 'Startup with Windows' through the tray icon
	; or if the user has disabled autostart through the copilot settings.
	${Debug} "Trying to find installation directory via 'Run' registry value..."
	ReadRegStr $0 HKCU "Software\Microsoft\Windows\CurrentVersion\Run" "Kite"
	${If} $0 == ""
		${Debug} "Could not find any 'Run' value for Kite, most likely because updater is running as a different user"
		Goto check_updater_dir
	${EndIf}
	Push $0
	Call GetParent
	Pop $0
	StrCpy $0 "$0\kited.exe"
	IfFileExists $0 0 check_updater_dir
	Push $0
	Call GetParent
	Return
	
	check_updater_dir:
	; check the updater's directory
	${Debug} "Trying to find installation directory via looking in our current directory..."
	${GetExePath} $0
	StrCpy $0 "$0\kited.exe"
	IfFileExists $0 0 check_installpath
	Push $0
	Call GetParent
	Return

	check_installpath:
	${Debug} "Trying to find installation directory through 'InstallPath' registry entry..."
	ReadRegStr $0 HKLM "Software\Kite\AppData" "InstallPath"
	${If} $0 != ""
		StrCpy $0 "$0\kited.exe"
		IfFileExists $0 0 check_program_files
		Push $0
		Call GetParent
		Return
	${EndIf}

	check_program_files:
	; check %PROGRAMFILES%
	${Debug} "Trying to find installation directory via looking in %programfiles64%..."
	StrCpy $0 "$PROGRAMFILES64\Kite\kited.exe"
	IfFileExists $0 0 fail
	Push $0
	Call GetParent
	Return
	
	fail:
	Push ""
FunctionEnd
