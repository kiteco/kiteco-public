// +build ignore

package text

func testDedupedFuncCall() {
	// TEST
	// f, err := os.Open(path)
	// if err != nil {
	//     log.Fatalln(err)
	// }
	// defer f.$
	// @0 Close()
	// @! Close(
	// @! Close
	// status: ok
}
