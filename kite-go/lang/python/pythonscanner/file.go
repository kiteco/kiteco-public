package pythonscanner

import "go/token"

// File constructs a file for the given buffer
func File(buf []byte) *token.File {
	return LabelledFile(buf, "src.py")
}

// LabelledFile constructs a file for the given buffer with a label
func LabelledFile(buf []byte, label string) *token.File {
	fset := token.NewFileSet()
	file := fset.AddFile(label, -1, len(buf))
	file.SetLinesForContent(buf)
	return file
}
