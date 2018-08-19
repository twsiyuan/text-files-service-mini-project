package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestDirStatistics(t *testing.T) {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir = filepath.Join(dir, "/test.folder.34238sxe")
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	if err := os.MkdirAll(dir+"/a", os.ModePerm); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(dir+"/b", os.ModePerm); err != nil {
		t.Fatal(err)
	}

	if err := ioutil.WriteFile(filepath.Join(dir, "/a.txt"), ([]byte)("hi hi"), os.ModePerm); err != nil {
		t.Fatal(err)
	}

	if err := ioutil.WriteFile(filepath.Join(dir, "/b.txt"), ([]byte)("world"), os.ModePerm); err != nil {
		t.Fatal(err)
	}

	stat, err := dirStatistics(dir)
	if err != nil {
		t.Fatal(err)
	}

	if stat.NumFiles != 2 {
		t.Errorf("Unexpected NumFiles, want: 2, got: %d", stat.NumFiles)
	}
	if !floatEquals(stat.AvgNumAlphaCharsPerFile, 4.5) {
		t.Errorf("Unexpected AvgNumAlphaCharsPerFile, want: 4.5, got: %.2f", stat.AvgNumAlphaCharsPerFile)
	}
	if !floatEquals(stat.AvgWordLength, 3) {
		t.Errorf("Unexpected AvgWordLength, want: 3, got: %.2f", stat.AvgWordLength)
	}
}

func floatEquals(a, b float64) bool {
	const EPSILON float64 = 0.00000001
	return (a-b) < EPSILON && (b-a) < EPSILON
}
