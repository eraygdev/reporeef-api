package profanity

import "testing"

// TestContains — küfür filtresi davranışı.
// En kritik kural: exact word match. "götür" masum, "göt" değil.
func TestContains(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		// ─── KELİME SINIRI (en kritik test) ───
		{name: "göt tek kelime", input: "göt", want: true},
		{name: "götür masum (substring)", input: "götür", want: false},
		{name: "göt veren (ayrı kelime)", input: "göt veren", want: true},
		{name: "götür beni masum", input: "götür beni", want: false},
		{name: "siklon masum (sik substring)", input: "siklon", want: false},

		// ─── TÜRKÇE KÜFÜRLER ───
		{name: "sik", input: "sik", want: true},
		{name: "sikeyim", input: "sikeyim", want: true},
		{name: "siktir", input: "siktir", want: true},
		{name: "amk", input: "amk", want: true},
		{name: "amq", input: "amq", want: true},
		{name: "aq", input: "aq", want: true},
		{name: "orospu", input: "orospu", want: true},
		{name: "yarrak", input: "yarrak", want: true},
		{name: "bok", input: "bok", want: true},
		{name: "boktan", input: "boktan", want: true},
		{name: "defol", input: "defol", want: true},
		{name: "pezevenk", input: "pezevenk", want: true},

		// ─── İNGİLİZCE KÜFÜRLER ───
		{name: "fuck", input: "fuck", want: true},
		{name: "fucking", input: "fucking", want: true},
		{name: "bitch", input: "bitch", want: true},
		{name: "asshole", input: "asshole", want: true},
		{name: "nigger", input: "nigger", want: true},
		{name: "faggot", input: "faggot", want: true},
		{name: "retard", input: "retard", want: true},
		{name: "stfu", input: "stfu", want: true},

		// ─── BÜYÜK HARF ───
		{name: "uppercase SIKTIR", input: "SIKTIR", want: true},
		{name: "uppercase AMK", input: "AMK", want: true},
		{name: "uppercase FUCK", input: "FUCK", want: true},
		{name: "mixed case FuCkInG", input: "FuCkInG", want: true},

		// ─── TÜRKÇE KARAKTER NORMALİZASYONU ───
		{name: "türkçe İ büyük", input: "SİKTİR", want: true},
		{name: "türkçe Ş", input: "ŞİKTİR", want: true}, // ş→s, İ→i, k, t, i, r = siktir
		{name: "türkçe Ö", input: "GÖT", want: true},    // göt → got
		{name: "türkçe Ü", input: "GÖTÜR", want: false}, // götür → gotur (masum)

		// ─── LEET-SPEAK ───
		{name: "leet 1", input: "s1kt1r", want: true},
		{name: "leet 0", input: "b0k", want: true},
		{name: "leet 3", input: "n1gg3r", want: true},
		{name: "leet @", input: "@sshole", want: true},
		{name: "leet $", input: "$hit", want: true},

		// ─── TEMİZ METİN ───
		{name: "hello world", input: "hello world", want: false},
		{name: "türkçe cümle", input: "merhaba dünya", want: false},
		{name: "proje başlığı", input: "Açık kaynak proje yönetimi", want: false},

		// ─── BOŞ / WHITESPACE ───
		{name: "empty", input: "", want: false},
		{name: "sadece boşluk", input: "   ", want: false},
		{name: "tab", input: "\t\t", want: false},
		{name: "yeni satır", input: "\n\n", want: false},

		// ─── ÇOKLU SATIR ───
		{name: "multiline with küfür", input: "hello\nsiktir\nworld", want: true},
		{name: "multiline temiz", input: "hello\nworld\nfoo", want: false},

		// ─── CÜMLE İÇİNDE ───
		{name: "amk proje", input: "amk proje", want: true},
		{name: "cümle sonunda", input: "bu ne amk", want: true},
		{name: "cümle başında", input: "siktir git", want: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Contains(tc.input)
			if got != tc.want {
				t.Errorf("Contains(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

// TestNormalize — normalizasyon davranışı (lowercase + türkçe + leet + whitespace).
func TestNormalize(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "lowercase", input: "HELLO", want: "hello"},
		{name: "türkçe İ", input: "SİKTİR", want: "siktir"},
		{name: "türkçe Ş", input: "ŞAL", want: "sal"},
		{name: "türkçe Ğ", input: "DAĞ", want: "dag"},
		{name: "türkçe Ç", input: "ÇAY", want: "cay"},
		{name: "türkçe Ö", input: "GÖL", want: "gol"},
		{name: "türkçe Ü", input: "GÜL", want: "gul"},
		{name: "türkçe ı", input: "KIŞ", want: "kis"},

		{name: "leet 1", input: "s1kt1r", want: "siktir"},
		{name: "leet 0", input: "b0k", want: "bok"},
		{name: "leet 3", input: "n1gg3r", want: "nigger"},
		{name: "leet @", input: "@ss", want: "ass"},
		{name: "leet 4", input: "4m", want: "am"},
		{name: "leet 5", input: "5ik", want: "sik"},
		{name: "leet 7", input: "7asak", want: "tasak"},
		{name: "leet $", input: "$hit", want: "shit"},

		{name: "noktalama boşluğa", input: "hello,world", want: "hello world"},
		{name: "nokta", input: "hello.world", want: "hello world"},
		{name: "ünlem", input: "wow!!!", want: "wow"},
		{name: "parantez", input: "(test)", want: "test"},
		{name: "slash", input: "a/b", want: "a b"},

		{name: "çoklu boşluk", input: "a    b", want: "a b"},
		{name: "baş/son boşluk", input: "   hello   ", want: "hello"},
		{name: "tab ve newline", input: "a\t\tb\n\nc", want: "a b c"},
		{name: "karışık", input: "  Hello,,,World!!!  ", want: "hello world"},

		{name: "empty", input: "", want: ""},
		{name: "sadece boşluk", input: "   ", want: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := normalize(tc.input)
			if got != tc.want {
				t.Errorf("normalize(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
