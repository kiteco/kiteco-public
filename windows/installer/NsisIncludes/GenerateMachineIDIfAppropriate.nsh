Var machine_id_already_existed
Var tentative_or_actual_machine_id

; (This function will write to the registry under certain conditions.)
Function ReadMachineIDOrGenerateIfAppropriate
	; Try reading an existing MachineID, if there is one

    ; order of precedence:
    ;   64-bit HKLM
    ;   32-bit HKLM
	ReadRegStr $0 HKLM "Software\Kite" "MachineID"
	${If} $0 == ""
		${Debug} "No 64bit HKLM MachineID found"

		SetRegView 32
		ReadRegStr $0 HKLM "Software\Kite" "MachineID"
		SetRegView 64

		${If} $0 == ""
			${Debug} "No 32bit HKLM MachineID found; no MachineID at all in the registry."
		${EndIf}
	${EndIf}
	${If} $0 == ""
		${Debug} "Generating new tentative MachineID..."
		StrCpy $machine_id_already_existed "no"

		; Generate and write a new value
		Call GenerateGuid
		Pop $0
		StrCpy $tentative_or_actual_machine_id $0
	${Else}
		${Debug} "MachineID already existed"
		StrCpy $machine_id_already_existed "yes"

		StrCpy $tentative_or_actual_machine_id $0
	${EndIf}
FunctionEnd


; Note: Must call GenerateMachineIDIfAppropriate before calling this function!
Function WriteTentativeOrActualMachineIDToRegistry
	${Debug} "Writing tentative (or actual) machine id to the registry..."

	${If} $tentative_or_actual_machine_id == ""
		; I can't think of any way that this might happen.
		${Debug} "MachineID is empty!  Failing."

		MessageBox MB_OK "Installation failed due to empty MachineID.  Please visit https://github.com/kiteco/issue-tracker to report this failure."
		Quit
	${EndIf}

	WriteRegStr HKLM "Software\Kite" "MachineID" "$tentative_or_actual_machine_id"

	; kited, KiteService, and Kite.exe are 64 bit so also write it there
	SetRegView 32
	WriteRegStr HKLM "Software\Kite" "MachineID" "$tentative_or_actual_machine_id"
	SetRegView 64
FunctionEnd


;Call GenerateGuid
;Pop $0 ;contains Guid
Function GenerateGuid
	  ; Guid has 128 bit = 16 byte = 32 hex characters
	  Push $R0
	  Push $R1
	  Push $R2
	  Push $R3
	  Push $R4
	  ;allocate space for character array
	  System::Alloc 16
	  ;get pointer to new space
	  Pop $R1
	  StrCpy $R0 "" ; init
	  ;call the CoCreateGuid api in the ole32.dll
	  System::Call 'ole32::CoCreateGuid(i R1) i .R2'
	  ;if 0 then continue
	  IntCmp $R2 0 continue
	  ; set error flag
	  SetErrors
	  goto done
	continue:
	  ;byte counter = 0
	  StrCpy $R3 0
	loop:
	    System::Call "*$R1(&v$R3, &i1 .R2)"
	    ;now $R2 is byte at offset $R3
	    ;convert to hex
	    IntFmt $R4 "%x" $R2
	    StrCpy $R4 "00$R4"
	    StrLen $R2 $R4
	    IntOp $R2 $R2 - 2
	    StrCpy $R4 $R4 2 $R2
	    ;append to result
	    StrCpy $R0 "$R0$R4"
	    ;increment byte counter
	    IntOp $R3 $R3 + 1
	    ;if less than 16 then continue
	    IntCmp $R3 16 0 loop
	done:
	  ;cleanup
	  System::Free $R1
	  Pop $R4
	  Pop $R3
	  Pop $R2
	  Pop $R1
	  Exch $R0
FunctionEnd
