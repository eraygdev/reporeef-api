package app

import "testing"

func TestIsAllowedContributorLimit(t *testing.T) {
	tests := []struct {
		name  string
		limit int
		want  bool
	}{
		{name: "5 geçerli", limit: 5, want: true},
		{name: "10 geçerli", limit: 10, want: true},
		{name: "20 geçerli", limit: 20, want: true},
		{name: "50 geçerli", limit: 50, want: true},
		{name: "0 geçersiz", limit: 0, want: false},
		{name: "1 geçersiz", limit: 1, want: false},
		{name: "6 geçersiz (listede yok)", limit: 6, want: false},
		{name: "15 geçersiz", limit: 15, want: false},
		{name: "100 geçersiz", limit: 100, want: false},
		{name: "negatif", limit: -1, want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isAllowedContributorLimit(tc.limit)
			if got != tc.want {
				t.Errorf("isAllowedContributorLimit(%d) = %v, want %v", tc.limit, got, tc.want)
			}
		})
	}
}
