!macro Debug LogString
!verbose push
!verbose 3
	Push $0 ; backup registers
	Push $1
	Push $2
	
	; we have to do this here so that if LogString contains a reference to
	;   one of the registers then the correct value will end up in the string
	;   we print bc we haven't mutated any of the registers yet.
	StrCpy $0 "${LogString}"
	
	; backup error flag
	StrCpy $1 "noerrorset"
	IfErrors 0 +2
	StrCpy $1 "errorset"
	Push $1
	
	;DetailPrint "$0"
	
	ReadRegStr $1 HKLM "Software\Kite" "IsDebug"
	${If} $1 == "true"
	${OrIf} $1 == "True"
		; prepend the process name to the debug string
		${GetProcessInfo} 0 $2 $2 $2 $1 $2
		${StrRep} $1 $1 ".exe" ""
		StrCpy $0 "$1 (Kite): $0"
		
		System::Call "kernel32::OutputDebugString(tr0)v"
	${EndIf}
	
	; restore error flag
	Pop $1
	${If} $1 == "noerrorset"
		ClearErrors
	${Else}
		SetErrors
	${EndIf}
	
	Pop $2
	Pop $1
	Pop $0
!verbose pop
!macroend
!define Debug "!insertmacro Debug"

!macro un.Debug LogString
!verbose push
!verbose 3
	Push $0 ; backup registers
	Push $1
	Push $2
	
	; we have to do this here so that if LogString contains a reference to
	; one of the registers then the correct value will end up in the string
	; we print bc we haven't mutated any of the registers yet.
	StrCpy $0 "${LogString}"
	
	; backup error flag
	StrCpy $1 "noerrorset"
	IfErrors 0 +2
	StrCpy $1 "errorset"
	Push $1
	
	;DetailPrint "$0"
	
	ReadRegStr $1 HKLM "Software\Kite" "IsDebug"
	${If} $1 == "true"
	${OrIf} $1 == "True"
		; note: getting the process name dynamically (as above) doesn't work for uninstallers
		; since NSIS copies the uninstaller to the windows temp directory with a random name
		; so that the original uninstaller can be deleted if appropriate.
		StrCpy $0 "KiteUninstaller: $0"
		
		System::Call "kernel32::OutputDebugString(tr0)v"
	${EndIf}
	
	; restore error flag
	Pop $1
	${If} $1 == "noerrorset"
		ClearErrors
	${Else}
		SetErrors
	${EndIf}
	
	Pop $2
	Pop $1
	Pop $0
!verbose pop
!macroend
!define un.Debug "!insertmacro un.Debug"