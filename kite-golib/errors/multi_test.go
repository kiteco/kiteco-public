package errors

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAppendNil(t *testing.T) {
	err := New("error")
	errs := Append(nil, err).sliceNoCopy()
	require.Len(t, errs, 1)
	require.Equal(t, err, errs[0])

	errs = Append(errorSlice([]error{err}), nil).sliceNoCopy()
	require.Len(t, errs, 1)
	require.Equal(t, err, errs[0])
}

func TestAppendMultiMulti(t *testing.T) {
	err0 := New("error0")
	err1 := New("error1")
	err2 := New("error2")
	err3 := New("error3")

	var errs01 Errors
	errs01 = Append(errs01, err0)
	errs01 = Append(errs01, err1)
	var errs23 Errors
	errs23 = Append(errs23, err2)
	errs23 = Append(errs23, err3)

	errs := Append(errs01, errs23).sliceNoCopy()
	require.Len(t, errs, 4)
	require.Equal(t, err0, errs[0])
	require.Equal(t, err1, errs[1])
	require.Equal(t, err2, errs[2])
	require.Equal(t, err3, errs[3])
}

func TestCombineNil(t *testing.T) {
	err := New("error")
	require.Equal(t, err, Combine(err, nil))
	require.Equal(t, err, Combine(nil, err))
}

func TestCombineBasic(t *testing.T) {
	err0 := New("error0")
	err1 := New("error1")

	errs := Combine(err0, err1).(Errors).sliceNoCopy()
	require.Len(t, errs, 2)
	require.Equal(t, err0, errs[0])
	require.Equal(t, err1, errs[1])
}

func TestCombineMulti(t *testing.T) {
	err0 := New("error0")
	err1 := New("error1")
	err2 := New("error2")
	err3 := New("error3")

	var errs01 Errors
	errs01 = Append(errs01, err0)
	errs01 = Append(errs01, err1)
	var errs23 Errors
	errs23 = Append(errs23, err2)
	errs23 = Append(errs23, err3)

	errs := Combine(err1, errs23).(Errors).sliceNoCopy()
	require.Len(t, errs, 3)
	require.Equal(t, err1, errs[0])
	require.Equal(t, err2, errs[1])
	require.Equal(t, err3, errs[2])

	errs = Combine(errs01, err2).(Errors).sliceNoCopy()
	require.Len(t, errs, 3)
	require.Equal(t, err0, errs[0])
	require.Equal(t, err1, errs[1])
	require.Equal(t, err2, errs[2])
	err2Ref := &errs[2]

	errs = Combine(errs01, errs23).(Errors).sliceNoCopy()
	require.Len(t, errs, 4)
	require.Equal(t, err0, errs[0])
	require.Equal(t, err1, errs[1])
	require.Equal(t, err2, errs[2])
	require.Equal(t, err3, errs[3])

	// make sure the second combine doesn't overwrite the first
	require.Equal(t, err2, *err2Ref)
}
