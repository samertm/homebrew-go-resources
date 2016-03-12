package main

import (
	"testing"
)

func TestNormalizeClonePath(t *testing.T) {
	testData := []struct {
		input string
		want  string
	}{
		{
			"https://github.com/normal/repo",
			"https://github.com/normal/repo",
		}, {
			"git@github.com:normal/repo",
			"https://github.com/normal/repo",
		}, {
			"ssh://git@github.com/normal/repo",
			"https://github.com/normal/repo",
		}, {
			"https://username@github.com/normal/repo",
			"https://github.com/normal/repo",
		}, {
			"https://username:password@github.com/normal/repo",
			"https://github.com/normal/repo",
		},
	}

	for _, d := range testData {
		got, err := normalizeClonePath(d.input)
		if err != nil {
			t.Errorf("Input %q, got error %+v", d.input, err)
			continue
		}
		if got != d.want {
			t.Errorf("Input %q, expected %q, got %q", d.input, d.want, got)
		}
	}
}
