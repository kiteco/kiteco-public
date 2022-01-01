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
)

func createLockParams(lockType int16) *syscall.Flock_t {
	return &syscall.Flock_t{
		Type:   lockType,
		Start:  0,
		Len:    1,
		Whence: 0,
	}
}

func tryLockFileImpl(file *os.File) (err error) {
	return syscall.FcntlFlock(
		file.Fd(),
		syscall.F_SETLK,
		createLockParams(syscall.F_WRLCK))
}

func lockFileImpl(file *os.File) (err error) {
	return syscall.FcntlFlock(
		file.Fd(),
		syscall.F_SETLKW,
		createLockParams(syscall.F_WRLCK))
}

func unlockFileImpl(file *os.File) (err error) {
	return syscall.FcntlFlock(
		file.Fd(),
		syscall.F_SETLK,
		createLockParams(syscall.F_UNLCK))
}
