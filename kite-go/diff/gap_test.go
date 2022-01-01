package diff

import (
	"bytes"
	"testing"
)

func TestGapBuffer_New(t *testing.T) {
	buf := []byte("Here is some text")
	gb := NewGapBuffer(buf)
	if gb.gapStart != 0 {
		t.Errorf("Incorrect initial gapStart. Expected %d, got %d",
			0, gb.gapStart)
	}
	if gb.gapEnd != DefaultGapSize {
		t.Errorf("Incorrect initial gapEnd. Expected %d, got %d",
			DefaultGapSize, gb.gapEnd)
	}
}

func TestGapBuffer_EmptyInsert(t *testing.T) {
	buf := []byte("")
	gb := NewGapBuffer(buf)
	gb.Insert(12, []byte("hello"))

	got := gb.Bytes()
	expected := bytes.Repeat([]byte{0}, 12)
	expected = append(expected, []byte("hello")...)
	if !bytes.Equal(got, expected) {
		t.Errorf("GapBuffer Insert failed: Expected \"%s\", got \"%s\"",
			expected, got)
	}
}

func TestGapBuffer_EmptyDelete(t *testing.T) {
	buf := []byte("")
	gb := NewGapBuffer(buf)
	gb.Delete(12, []byte("hello"), false)

	got := gb.Bytes()
	expected := bytes.Repeat([]byte{0}, 12)
	if !bytes.Equal(got, expected) {
		t.Errorf("GapBuffer Delete failed: Expected \"%x\", got \"%x\"",
			expected, got)
	}
}

func TestGapBuffer_OutOfBoundsDelete(t *testing.T) {
	buf := []byte("")
	gb := NewGapBuffer(buf)
	ins := []byte("hello world, whats up?")
	gb.Insert(0, ins)
	gb.Delete(15, []byte("something different"), false)

	got := gb.Bytes()
	expected := ins[:15]
	if !bytes.Equal(got, expected) {
		t.Errorf("GapBuffer Delete failed: Expected \"%s\", got \"%s\"",
			expected, got)
	}
}

func TestGapBuffer_Insert(t *testing.T) {
	buf := []byte("Here is some text")
	gb := NewGapBuffer(buf)
	gb.Insert(12, []byte(" more"))

	got := gb.Bytes()
	expected := []byte("Here is some more text")
	if !bytes.Equal(got, expected) {
		t.Errorf("GapBuffer Insert failed: Expected \"%s\", got \"%s\"",
			expected, got)
	}
}

func TestGapBuffer_LargeInsert(t *testing.T) {
	gb := NewGapBuffer([]byte(""))
	insert := bytes.Repeat([]byte("A"), 200)
	gb.Insert(4, insert)

	got := gb.Bytes()
	expected := bytes.Repeat([]byte{0}, 4)
	expected = append(expected, insert...)

	if !bytes.Equal(got, expected) {
		t.Errorf("GapBuffer large Insert failed: Expected \"%s\", got \"%s\"",
			expected, got)
	}
}

func TestGapBuffer_Delete(t *testing.T) {
	buf := []byte("Here is some text")
	gb := NewGapBuffer(buf)
	_ = gb.Delete(8, []byte("some "), true)

	got := gb.Bytes()
	expected := []byte("Here is text")
	if !bytes.Equal(got, expected) {
		t.Errorf("GapBuffer Delete failed: Expected \"%s\", got \"%s\"",
			expected, got)
	}
}

func TestGapBuffer_DeleteVerification(t *testing.T) {
	buf := []byte("Here is some text")
	gb := NewGapBuffer(buf)
	err := gb.Delete(8, []byte("foo"), true)
	if err != ErrDeleteMismatch {
		t.Errorf("Expected delete verification to fail. Succeeded.")
	}
	err = gb.Delete(8, []byte("some"), true)
	if err != nil {
		t.Errorf("Expected delete verification to succeed. Got: %s", err)
	}
}
