package bufutil

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

func intToBytes(val int64, bo binary.ByteOrder) []byte {
	buf := &bytes.Buffer{}
	binary.Write(buf, bo, val)
	return buf.Bytes()
}

func uintToBytes(val uint64, bo binary.ByteOrder) []byte {
	buf := &bytes.Buffer{}
	binary.Write(buf, bo, val)
	return buf.Bytes()
}

// --

// IntToBytes returns a []byte with the provided int64 encoded
// in binary.LittleEndian byte order.
func IntToBytes(val int64) []byte {
	return intToBytes(val, binary.LittleEndian)
}

// UintToBytes returns a []byte with the provided uint64 encoded
// in binary.LittleEndian byte order.
func UintToBytes(val uint64) []byte {
	return uintToBytes(val, binary.LittleEndian)
}

// IntToBytesReverse returns a []byte with the provided int64 encoded
// in binary.BigEndian byte order.
func IntToBytesReverse(val int64) []byte {
	return intToBytes(val, binary.BigEndian)
}

// UintToBytesReverse returns a []byte with the provided uint64 encoded
// in binary.BigEndian byte order.
func UintToBytesReverse(val uint64) []byte {
	return uintToBytes(val, binary.BigEndian)
}

// --

func bytesToInt(buf []byte, bo binary.ByteOrder) int64 {
	var val int64
	reader := bytes.NewBuffer(buf)
	binary.Read(reader, bo, &val)
	return val
}

func bytesToUint(buf []byte, bo binary.ByteOrder) uint64 {
	var val uint64
	reader := bytes.NewBuffer(buf)
	binary.Read(reader, bo, &val)
	return val
}

// --

// BytesToInt returns the int64 encoded by the provided buf,
// assuming buf was encoded in binary.LittleEndian byte order.
func BytesToInt(buf []byte) int64 {
	return bytesToInt(buf, binary.LittleEndian)
}

// BytesToUint returns the uint64 encoded by the provided buf,
// assuming buf was encoded in binary.LittleEndian byte order.
func BytesToUint(buf []byte) uint64 {
	return bytesToUint(buf, binary.LittleEndian)
}

// BytesToIntReverse returns the int64 encoded by the provided buf,
// assuming buf was encoded in binary.BigEndian byte order.
func BytesToIntReverse(buf []byte) int64 {
	return bytesToInt(buf, binary.BigEndian)
}

// BytesToUintReverse returns the uint64 encoded by the provided buf,
// assuming buf was encoded in binary.BigEndian byte order.
func BytesToUintReverse(buf []byte) uint64 {
	return bytesToUint(buf, binary.BigEndian)
}

// The following are implementations of fmt.Stringer for each of the types above.

// IntStringer is a type
type IntStringer []byte

func (i IntStringer) String() string {
	val := BytesToInt(i)
	return fmt.Sprintf("%d", val)
}

// --

// UintStringer is a type
type UintStringer []byte

func (i UintStringer) String() string {
	val := BytesToUint(i)
	return fmt.Sprintf("%d", val)
}

// --

// ReversedIntStringer is a type
type ReversedIntStringer []byte

func (r ReversedIntStringer) String() string {
	val := BytesToIntReverse(r)
	return fmt.Sprintf("%d", val)
}

// --

// ReversedUintStringer is a type
type ReversedUintStringer []byte

func (r ReversedUintStringer) String() string {
	val := BytesToUintReverse(r)
	return fmt.Sprintf("%d", val)
}
