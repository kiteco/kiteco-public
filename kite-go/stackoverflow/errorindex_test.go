package stackoverflow

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/stretchr/testify/assert"
)

func TestErrorIndex_LookupPosts(t *testing.T) {
	index := make(errorIndex)
	index[errorKey{lang.Python, 123}] = []int{777, 888}
	assert.Equal(t, []int{777, 888}, index.LookupPosts(lang.Python, 123))
	assert.Nil(t, index.LookupPosts(lang.Golang, 123))
	assert.Nil(t, index.LookupPosts(lang.Python, 0))
}

func TestErrorIndex_LoadJson(t *testing.T) {
	data := `{"items": [
		{"error_id": 123, "post_ids": [1, 2, 3]},
		{"error_id": 456, "post_ids": [4, 5, 6]}
	]}`

	index := make(errorIndex)
	buf := bytes.NewBufferString(data)
	err := loadJSON(buf, lang.Cpp, index)

	if err != nil {
		fmt.Println(err)
	}

	assert.Nil(t, err)

	expected := errorIndex{
		errorKey{lang.Cpp, 123}: []int{1, 2, 3},
		errorKey{lang.Cpp, 456}: []int{4, 5, 6},
	}

	assert.Equal(t, expected, index)
}
