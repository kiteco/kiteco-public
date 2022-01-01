
; Usage:
;   Call CheckAlreadyRunningInstallOrUninstall
; 
; warning: clobbers $R0
Function CheckAlreadyRunningInstallOrUninstall
	System::Call 'kernel32::CreateMutex(i 0, i 0, t "KiteSetup") ?e'
  Pop $R0
  ${If} $R0 != 0
		${Debug} "Already running install or uninstall detected.  Quiting after telling the user..."
  	MessageBox MB_OK|MB_ICONINFORMATION "An installation or uninstallation of Kite is already running.  Setup will now exit."
  	Quit
  ${EndIf}
FunctionEnd

; Usage:
; 	Call SilentCheckAlreadyRunningInstallOrUninstall
; 	Pop $0
;		(check for $0 != 0)
Function SilentCheckAlreadyRunningInstallOrUninstall
	System::Call 'kernel32::CreateMutex(i 0, i 0, t "KiteSetup") ?e'
FunctionEnd

; it's bad practice to duplicate code for an uninstallation function, but ah well
; for more detail, see http://nsis.sourceforge.net/Sharing_functions_between_Installer_and_Uninstaller
Function un.CheckAlreadyRunningInstallOrUninstall
	System::Call 'kernel32::CreateMutex(i 0, i 0, t "KiteSetup") ?e'
  Pop $R0
  ${If} $R0 != 0
		${un.Debug} "Already running install or uninstall detected.  Quiting after telling the user..."
  	MessageBox MB_OK|MB_ICONINFORMATION "An installation or uninstallation of Kite is already running.  Setup will now exit."
  	Quit
  ${EndIf}
FunctionEnd