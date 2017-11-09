package main

import (
	"testing"
)

func TestInsertEveryN(t *testing.T) {
	testCases := []struct {
		what, where string
		n           int
		out         string
	}{
		{},
		{"", "abc", 0, "abc"},
		{"", "abc", 1, "abc"},
		{"", "abc", 2, "abc"},
		{"", "abc", 3, "abc"},
		{"", "abc", 4, "abc"},
		{"12", "abcdefgh", 0, "abcdefgh"},
		{"12", "abcdefgh", 1, "a12b12c12d12e12f12g12h"},
		{"12", "abcdefgh", 2, "ab12cd12ef12gh"},
		{"12", "abcdefgh", 3, "abc12def12gh"},
		{"12", "abcdefgh", 4, "abcd12efgh"},
		{"12", "abcdefgh", 5, "abcde12fgh"},
		{"12", "abcdefgh", 6, "abcdef12gh"},
		{"12", "abcdefgh", 7, "abcdefg12h"},
		{"12", "abcdefgh", 8, "abcdefgh"},
		{"12", "abcdefgh", 9, "abcdefgh"},
		{"一二", "あえいおうたてち", 4, "あえいお一二うたてち"},
		{"", "", 0, ""},
	}

	for _, tc := range testCases {
		if out := insertEveryN(tc.what, tc.where, tc.n); out != tc.out {
			t.Errorf("insertEveryN(%s, %s, %d) = %s, want %s", tc.what, tc.where, tc.n, out, tc.out)
		}
	}
}
