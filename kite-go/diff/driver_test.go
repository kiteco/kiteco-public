package diff

import (
	"bytes"
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/kiteco/kiteco/kite-go/event"
	"github.com/kiteco/kiteco/kite-go/proto"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// -- Test Random Edits
// TODO(juan): use go-fuzz package
const (
	maxDocLength        = 500
	nDocs               = 5000
	maxLengthEditStream = 100
	maxDiffLength       = 10
	maxDiffs            = 10
)

var characters = []byte(`1234567890!@#$%^&*()_-=+~\|]}[{'";:/?.>,<abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ`)

func randomCharacter() string {
	idx := rand.Intn(len(characters))
	return string(characters[idx])
}

func randomASCIIDocument() string {
	length := rand.Intn(maxDocLength)
	var doc string
	for i := 0; i < length; i++ {
		doc += randomCharacter()
	}
	return doc
}

// randomEdits applies a series of random changes to the provided document
func randomASCIIEdits(doc string) string {
	nChanges := rand.Intn(maxDiffs)
	for i := 0; i < nChanges; i++ {
		choice := rand.Float64()
		switch {
		case choice < 0.33:
			// insert
			var pos int
			if len(doc) > 0 {
				pos = rand.Intn(len(doc))
			}

			var parts []string
			length := rand.Intn(maxDiffLength)
			for i := 0; i < length; i++ {
				parts = append(parts, randomCharacter())
			}

			doc = strings.Join([]string{
				doc[:pos],
				strings.Join(parts, ""),
				doc[pos:],
			}, "")

		case choice < 0.66:
			// delete
			if len(doc) == 0 {
				continue
			}

			pos := rand.Intn(len(doc))
			toDelete := rand.Intn(maxDiffLength-1) + 1
			if pos+toDelete < len(doc) {
				doc = doc[:pos] + doc[pos+toDelete:]
			} else {
				doc = doc[:pos]
			}

		default:
			// no changes
		}
	}
	return doc
}

func diffPtrs(diffs []event.Diff) []*event.Diff {
	var ptrs []*event.Diff
	for i := range diffs {
		ptrs = append(ptrs, &diffs[i])
	}
	return ptrs
}

func diffString(diff *event.Diff) string {
	return fmt.Sprintf("{%s @ %d %v}", diff.GetType().String(), diff.GetOffset(), []byte(diff.GetText()))
}

func editsString(edits []event.Diff) string {
	var diffs []string
	for _, diff := range edits {
		diffs = append(diffs, diffString(&diff))
	}
	return strings.Join(diffs, ", ")
}

func TestRandomASCIIEdits(t *testing.T) {
	var nFail int
	for i := 0; i < nDocs; i++ {
		driver := NewBufferDriver()
		differ := NewDiffer()

		// +1 for initial insert
		nEdits := rand.Intn(maxLengthEditStream) + 1
		var edits []event.Diff

		var doc string
		newDoc := randomASCIIDocument()

	EditLoop:
		for j := 0; j < nEdits; j++ {
			// calculate diffs
			diffs := differ.Diff(doc, newDoc)

			edits = append(edits, diffs...)

			// update driver
			driver.HandleEvent(kitectx.Background(), &event.Event{
				Action: proto.String("edit"),
				Diffs:  diffPtrs(diffs),
			})

			if newDoc != string(driver.Bytes()) {
				t.Errorf("text not equal. Edit Stream\n%s\n", editsString(edits))
				t.Errorf("\nExpected\n%v\nActual\n%v\n", []byte(newDoc), driver.Bytes())
				nFail++
				break EditLoop
			}

			doc = newDoc
			newDoc = randomASCIIEdits(doc)
		}
	}
	if nFail > 0 {
		t.Errorf("nFail %d, nOk %d, total %d", nFail, nDocs-nFail, nDocs)
	}
}

func randomByteDocument() []byte {
	length := rand.Intn(maxDocLength)
	bytes := make([]byte, length)
	rand.Read(bytes)
	return bytes
}

func randomByteEdits(docOrig []byte) []byte {
	var doc []byte
	copy(doc, docOrig)
	nChanges := rand.Intn(maxDiffs)
	for i := 0; i < nChanges; i++ {
		choice := rand.Float64()
		switch {
		case choice < 0.33:
			// insert
			var pos int
			if len(doc) > 0 {
				pos = rand.Intn(len(doc))
			}

			length := rand.Intn(maxDiffLength)
			insert := make([]byte, length)
			rand.Read(insert)

			doc = bytes.Join([][]byte{
				doc[:pos],
				insert,
				doc[pos:],
			}, nil)

		case choice < 0.66:
			// delete
			if len(doc) == 0 {
				continue
			}

			pos := rand.Intn(len(doc))
			toDelete := rand.Intn(maxDiffLength-1) + 1
			if pos+toDelete < len(doc) {
				doc = bytes.Join([][]byte{
					doc[:pos],
					doc[pos+toDelete:],
				}, nil)
			} else {
				doc = doc[:pos]
			}

		default:
			// no changes
		}
	}
	return doc
}

func TestRandomByteEdits(t *testing.T) {
	var nFail int
	for i := 0; i < nDocs; i++ {
		driver := NewBufferDriver()
		differ := NewDiffer()

		// +1 for initial insert
		nEdits := rand.Intn(maxLengthEditStream) + 1
		var edits []event.Diff

		var doc []byte
		newDoc := randomByteDocument()

	EditLoop:
		for j := 0; j < nEdits; j++ {
			// calculate diffs
			diffs := differ.Diff(string(doc), string(newDoc))

			// hacky normalization of bytes
			newDoc = []byte(string([]rune(string(newDoc))))

			edits = append(edits, diffs...)

			// update driver
			driver.HandleEvent(kitectx.Background(), &event.Event{
				Action: proto.String("edit"),
				Diffs:  diffPtrs(diffs),
			})

			if !bytes.Equal(newDoc, driver.Bytes()) {
				t.Errorf("text not equal. Edit Stream\n%s\n", editsString(edits))
				t.Errorf("\nExpected\n%q\nActual\n%q\n", newDoc, driver.Bytes())
				nFail++
				break EditLoop
			}

			doc = newDoc
			newDoc = randomByteEdits(doc)
		}
	}

	if nFail > 0 {
		t.Errorf("nFail %d, nOk %d, total %d", nFail, nDocs-nFail, nDocs)
	}
}
