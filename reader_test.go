package main

import (
	"bytes"
	"io"
	"testing"
)

func TestWordReader(t *testing.T) {
	b := bytes.NewReader(([]byte)(`hello
world 10 apple sc__y
34hee
`))
	r := NewWordReader(b)

	checkWord := func(word string) {
		if s, err := r.Read(); err != nil {
			t.Fatal(err)
		} else if s != word {
			t.Errorf("Unexpected word, want: %s, got: %s", word, s)
		}
	}
	checkErr := func(e error) {
		if _, err := r.Read(); err != e {
			t.Errorf("Unexpected error, want: %v, got: %v", e, err)
		}
	}

	checkWord("hello")
	checkWord("world")
	checkWord("apple")
	checkWord("sc")
	checkWord("y")
	checkWord("hee")
	checkErr(io.EOF)
	checkErr(io.EOF)
}
