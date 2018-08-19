package main

import (
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/montanaflynn/stats"
)

type stat struct {
	NumFiles                int
	AvgNumAlphaCharsPerFile float64
	StdNumAlphaCharsPerFile float64
	AvgWordLength           float64
	StdWordLength           float64
	TotalBytes              int64
}

func dirStatistics(dirname string) (*stat, error) {
	if info, err := os.Stat(dirname); err != nil {
		return nil, err
	} else if !info.IsDir() {
		return nil, errors.New("Not folder")
	}

	s := &stat{}
	files, err := ioutil.ReadDir(dirname)
	if err != nil {
		return nil, err
	}
	wordLens := make([]float64, 0)
	alphaCharsPerFile := make([]float64, 0)
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		s.NumFiles++
		s.TotalBytes += file.Size()

		f, err := os.Open(filepath.Join(dirname, "/", file.Name()))
		if err != nil {
			return nil, err
		}
		reader := NewWordReader(f)
		alphaChars := float64(0)
		for {
			s, err := reader.Read()
			if err == io.EOF {
				break
			}
			n := float64(len(s))
			alphaChars += n
			wordLens = append(wordLens, n)
		}
		alphaCharsPerFile = append(alphaCharsPerFile, alphaChars)
		f.Close()
	}
	s.AvgNumAlphaCharsPerFile, _ = stats.Mean(alphaCharsPerFile)
	s.StdNumAlphaCharsPerFile, _ = stats.StandardDeviation(alphaCharsPerFile)
	s.AvgWordLength, _ = stats.Mean(wordLens)
	s.StdWordLength, _ = stats.StandardDeviation(wordLens)

	return s, nil
}
