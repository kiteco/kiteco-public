// +build ignore

package text

func testNoCompletionsInComments() {
	// TEST
	// strings.Split(input, "\n") // no comp$ please
	// @EXACT
	// status: ok
}
