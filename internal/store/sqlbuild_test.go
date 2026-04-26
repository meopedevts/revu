package store

import "testing"

func TestInClause(t *testing.T) {
	cases := []struct {
		name string
		n    int
		want string
	}{
		{"zero", 0, "()"},
		{"one", 1, "(?)"},
		{"two", 2, "(?,?)"},
		{"five", 5, "(?,?,?,?,?)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := inClause(tc.n)
			if got != tc.want {
				t.Fatalf("inClause(%d) = %q, want %q", tc.n, got, tc.want)
			}
		})
	}
}
