package main

import (
	"io"
)

// WordReader is the interface that use read word ([a-zA-Z]+)
//
// When Read encounters end-of-file condition, it returns io.EOF
type WordReader interface {
	Read() (string, error)
}

// NewWordReader returns a new Reader that reads word
func NewWordReader(r io.Reader) WordReader {
	return &wordReader{
		reader: r,
		rbuf:   make([]byte, 1024),
	}
}

type wordReader struct {
	reader io.Reader
	rbuf   []byte
	pos    int
	wbuf   []byte
}

func (r wordReader) process(c byte) byte {
	// ASCII only
	if c >= 'a' && c <= 'z' {
		return c
	} else if c >= 'A' && c <= 'Z' {
		return c
	} else {
		return 0
	}
}

func (r *wordReader) Read() (string, error) {
	for {
		for r.pos < len(r.rbuf) {
			b := r.process(r.rbuf[r.pos])
			r.pos++
			if b == 0 {
				if l := len(r.wbuf); l > 0 {
					temp := make([]byte, l)
					copy(temp, r.wbuf)
					r.wbuf = r.wbuf[:0]
					return string(temp), nil
				}
			} else {
				r.wbuf = append(r.wbuf, b)
			}
		}

		_, err := r.reader.Read(r.rbuf)
		if err == io.EOF {
			if len(r.wbuf) > 0 {
				w := string(r.wbuf)
				r.wbuf = nil
				return w, nil
			}
			return "", io.EOF
		} else if err != nil {
			panic(err)
		}
		r.pos = 0
	}
}
