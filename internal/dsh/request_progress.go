package dsh

import "io"

type requestProgressBody struct {
	io.ReadCloser
	sent, total int64
	changed     func(int64, int64)
}

func (r *requestProgressBody) Read(p []byte) (int, error) {
	n, err := r.ReadCloser.Read(p)
	if n > 0 {
		r.sent += int64(n)
		r.changed(r.sent, r.total)
	}
	return n, err
}
