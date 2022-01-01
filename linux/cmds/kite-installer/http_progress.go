package main

// httpProgress can used in a io.TeeReader to print download information for a HTTP download
// it calls the onProgress callback function when new data was received
type httpProgress struct {
	total      int64
	received   int64
	onProgress func(received int64, total int64)
}

func (h *httpProgress) Write(p []byte) (int, error) {
	length := len(p)
	h.received += int64(length)
	if h.onProgress != nil {
		h.onProgress(h.received, h.total)
	}
	return length, nil
}
