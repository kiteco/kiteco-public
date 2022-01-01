package mockserver

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
)

// ReadResponse returns the response as a string. It returns an empty string when an error ocurred to simplify the test code.
func ReadResponse(resp *http.Response) string {
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	return string(data)
}

// ReadResponseRecorder returns the response as a string. It returns an empty string when an error ocurred to simplify the test code.
func ReadResponseRecorder(resp *httptest.ResponseRecorder) string {
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	return string(data)
}
