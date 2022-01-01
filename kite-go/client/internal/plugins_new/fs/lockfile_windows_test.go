package fs

// taken from https://github.com/giancosta86/LockAPI/blob/master/lockapi
// Apache 2 License

/*§
  ===========================================================================
  LockAPI
  ===========================================================================
  Copyright (C) 2015 Gianluca Costa
  ===========================================================================
  Licensed under the Apache License, Version 2.0 (the "License");
  you may not use this file except in compliance with the License.
  You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

  Unless required by applicable law or agreed to in writing, software
  distributed under the License is distributed on an "AS IS" BASIS,
  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  See the License for the specific language governing permissions and
  limitations under the License.
  ===========================================================================
*/

import (
	"os"
	"syscall"
	"unsafe"
)

const lockFileExclusiveLockFlag = 0x2
const lockFileFailImmediately = 0x1

func callKernel32Procedure(procedureName string, procedureCaller func(uintptr) error) (err error) {
	kernelHandle, err := syscall.LoadLibrary("kernel32.dll")
	if err != nil {
		return err
	}
	defer syscall.FreeLibrary(kernelHandle)

	procedureAddress, err := syscall.GetProcAddress(kernelHandle, procedureName)
	if err != nil {
		return err
	}

	return procedureCaller(procedureAddress)
}

func lockFileEx(file *os.File, flags uintptr) (err error) {
	return callKernel32Procedure("LockFileEx", func(procedureAddress uintptr) (err error) {
		overlapped := syscall.Overlapped{}

		lockResult, _, verboseErr := syscall.Syscall6(
			procedureAddress,
			6,

			file.Fd(),
			flags,
			0,
			1,
			0,
			uintptr(unsafe.Pointer(&overlapped)))

		if int(lockResult) == 0 {
			return verboseErr
		}

		return nil
	})
}

func unlockFileEx(file *os.File) (err error) {
	return callKernel32Procedure("UnlockFileEx", func(procedureAddress uintptr) (err error) {
		overlapped := syscall.Overlapped{}

		unlockResult, _, verboseErr := syscall.Syscall6(
			procedureAddress,
			5,

			file.Fd(),
			0,
			1,
			0,
			uintptr(unsafe.Pointer(&overlapped)),
			0)

		if int(unlockResult) == 0 {
			return verboseErr
		}

		return nil
	})
}
