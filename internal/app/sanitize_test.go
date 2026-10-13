package app

import "testing"

func TestSanitizeText(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty", input: "", want: ""},
		{name: "normal metin", input: "hello world", want: "hello world"},
		{name: "trim baş/son", input: "  hello  ", want: "hello"},
		{name: "türkçe metin", input: "Merhaba Dünya", want: "Merhaba Dünya"},

		// HTML tag temizleme
		{name: "script tag", input: "<script>alert(1)</script>", want: "alert(1)"},
		{name: "bold tag", input: "<b>bold</b>", want: "bold"},
		{name: "anchor tag", input: "<a href='x'>link</a>", want: "link"},
		{name: "img tag", input: "<img src='x.png'/>", want: ""},
		{name: "nested tag", input: "<div><span>x</span></div>", want: "x"},

		// Şema temizleme
		{name: "javascript scheme", input: "javascript:alert(1)", want: "alert(1)"},
		{name: "data scheme", input: "data:text/html", want: "text/html"},

		// Event handler
		{name: "onerror", input: "onerror=alert(1)", want: "alert(1)"},
		{name: "onload", input: "onload=evil()", want: "evil()"},

		// Açık kalan tag başlangıcı
		{name: "açık script", input: "<script", want: ""},
		{name: "açık kapanış", input: "</script", want: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := sanitizeText(tc.input)
			if got != tc.want {
				t.Errorf("sanitizeText(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestNullableString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantNil bool
		wantVal string
	}{
		{name: "empty → nil", input: "", wantNil: true},
		{name: "boşluk → string", input: " ", wantNil: false, wantVal: " "},
		{name: "normal", input: "hello", wantNil: false, wantVal: "hello"},
		{name: "trim edilmez", input: "  x  ", wantNil: false, wantVal: "  x  "},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := nullableString(tc.input)
			if tc.wantNil {
				if got != nil {
					t.Errorf("nullableString(%q) = %v, want nil", tc.input, got)
				}
				return
			}
			s, ok := got.(string)
			if !ok {
				t.Errorf("nullableString(%q) = %T, want string", tc.input, got)
				return
			}
			if s != tc.wantVal {
				t.Errorf("nullableString(%q) = %q, want %q", tc.input, s, tc.wantVal)
			}
		})
	}
}
