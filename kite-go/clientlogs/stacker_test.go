package clientlogs

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_SampleLog(t *testing.T) {
	logdata, err := ioutil.ReadFile("test/sample1.txt")
	require.NoError(t, err)

	errStr, traceback := findCrash(logdata)
	require.EqualValues(t, "Exception 0xc0000005 0x0 0xc003818f60 0x7ffa24904e10", errStr)

	stacker := loggedClientError{errStr, traceback}
	frames := stacker.Stack()
	require.Len(t, frames, 13)

	// syscall.Syscall(0x7ffa2558b4a0, 0x2, 0xc003818f60, 0xc00380b7b0, 0x0, 0x0, 0x0, 0x0)
	//	c:/go/src/runtime/syscall_windows.go:188 +0xe9
	first := frames[0]
	require.EqualValues(t, "syscall.Syscall", first.Function)
	require.EqualValues(t, "c:/go/src/runtime/syscall_windows.go", first.File)
	require.EqualValues(t, 188, first.Line)

	//github.com/kiteco/kiteco/vendor/github.com/go-ole/go-ole.CLSIDFromProgID(0x144416a1, 0xd, 0x10979d3, 0x142da100, 0xc003807110)
	//D:/containers/containers/0000334m612/tmp/build/140ee389/gopath/src/github.com/kiteco/kiteco/vendor/github.com/go-ole/go-ole/com.go:121 +0xbb
	third := frames[2]
	require.EqualValues(t, "github.com/kiteco/kiteco/vendor/github.com/go-ole/go-ole.CLSIDFromProgID", third.Function)
	require.EqualValues(t, "D:/containers/containers/0000334m612/tmp/build/140ee389/gopath/src/github.com/kiteco/kiteco/vendor/github.com/go-ole/go-ole/com.go", third.File)
	require.EqualValues(t, 121, third.Line)
}

func Test_SampleLog2(t *testing.T) {
	logdata, err := ioutil.ReadFile("test/sample2.txt")
	require.NoError(t, err)

	// fatal error: concurrent map writes
	//
	//goroutine 70 [running]:

	errStr, traceback := findCrash(logdata)
	require.EqualValues(t, "fatal error: concurrent map writes", errStr)

	stacker := loggedClientError{errStr, traceback}
	frames := stacker.Stack()
	require.Len(t, frames, 13)

	//runtime.throw(0x14445005, 0x15)
	//	c:/go/src/runtime/panic.go:1116 +0x79 fp=0xc00dbdf8a0 sp=0xc00dbdf870 pc=0x439c29
	first := frames[0]
	require.EqualValues(t, "runtime.throw", first.Function)
	require.EqualValues(t, "c:/go/src/runtime/panic.go", first.File)
	require.EqualValues(t, 1116, first.Line)

	//runtime.mapassign_faststr(0x141f8c80, 0xc000c83ec0, 0xc010134540, 0x6, 0xc000306420)
	//	c:/go/src/runtime/map_faststr.go:291 +0x3e5 fp=0xc00dbdf908 sp=0xc00dbdf8a0 pc=0x415c35
	second := frames[1]
	require.EqualValues(t, "runtime.mapassign_faststr", second.Function)
	require.EqualValues(t, "c:/go/src/runtime/map_faststr.go", second.File)
	require.EqualValues(t, 291, second.Line)
}

func Test_SampleShortLog(t *testing.T) {
	logdata, err := ioutil.ReadFile("test/sample-short.txt")
	require.NoError(t, err)

	errStr, traceback := findCrash(logdata)
	require.EqualValues(t, "Exception 0xc0000005 0x0 0xc003818f60 0x7ffa24904e10", errStr)

	stacker := loggedClientError{errStr, traceback}
	frames := stacker.Stack()
	require.Len(t, frames, 1)
}

func Test_EmptyLog(t *testing.T) {
	logdata, err := ioutil.ReadFile("test/sample-empty.txt")
	require.NoError(t, err)

	errStr, traceback := findCrash(logdata)
	require.EqualValues(t, "Exception 0xc0000005 0x0 0xc003818f60 0x7ffa24904e10", errStr)

	stacker := loggedClientError{errStr, traceback}
	frames := stacker.Stack()
	require.Nil(t, frames)
}
