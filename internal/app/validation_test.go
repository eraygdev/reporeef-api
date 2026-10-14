package app

import (
	"strings"
	"testing"
	"time"
)

// ═══════════════════════════════════════════════════════════
// validateText
// ═══════════════════════════════════════════════════════════

func TestValidateText(t *testing.T) {
	tests := []struct {
		name    string
		field   string
		value   string
		max     int
		re      string // regex adı — döngü içinde seçilir
		wantErr string // "" ise nil beklenir
	}{
		{name: "empty → nil", field: "title", value: "", max: 10, re: "Title", wantErr: ""},
		{name: "geçerli", field: "title", value: "hello", max: 10, re: "Title", wantErr: ""},
		{name: "çok uzun", field: "title", value: "hello world", max: 5, re: "Title", wantErr: "title_too_long"},
		{name: "geçersiz karakter", field: "title", value: "héllo", max: 10, re: "Title", wantErr: "title_invalid_char"},
		{name: "küfür", field: "title", value: "siktir", max: 10, re: "Title", wantErr: "title_profanity"},
		{name: "türkçe karakter geçerli", field: "title", value: "Merhaba Dünya", max: 20, re: "Title", wantErr: ""},
		{name: "multi-byte length check", field: "title", value: "İİİİİİ", max: 3, re: "Title", wantErr: "title_too_long"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var re = TitleRegex
			switch tc.re {
			case "Title":
				re = TitleRegex
			case "Text":
				re = TextRegex
			case "Username":
				re = UsernameRegex
			case "Bio":
				re = BioRegex
			}

			err := validateText(tc.field, tc.value, tc.max, re)
			checkErr(t, err, tc.wantErr)
		})
	}
}

// ═══════════════════════════════════════════════════════════
// validateTextMinMax
// ═══════════════════════════════════════════════════════════

func TestValidateTextMinMax(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		value    string
		min      int
		max      int
		required bool
		wantErr  string
	}{
		// required=true senaryoları
		{name: "required boş → too_short", field: "title", value: "", min: 3, max: 80, required: true, wantErr: "title_too_short"},
		{name: "required 2 karakter → too_short", field: "title", value: "ab", min: 3, max: 80, required: true, wantErr: "title_too_short"},
		{name: "required 3 karakter → ok", field: "title", value: "abc", min: 3, max: 80, required: true, wantErr: ""},
		{name: "required çok uzun → too_long", field: "title", value: strings.Repeat("a", 81), min: 3, max: 80, required: true, wantErr: "title_too_long"},

		// required=false senaryoları
		{name: "optional boş → ok", field: "longDescription", value: "", min: 0, max: 500, required: false, wantErr: ""},
		{name: "optional min>0 boş → ok", field: "longDescription", value: "", min: 5, max: 500, required: false, wantErr: ""},
		{name: "optional min>0 kısa → too_short", field: "longDescription", value: "ab", min: 5, max: 500, required: false, wantErr: "longDescription_too_short"},
		{name: "optional yeterli → ok", field: "longDescription", value: "abcde", min: 5, max: 500, required: false, wantErr: ""},

		// regex + profanity
		{name: "geçersiz karakter", field: "title", value: "abc<script>", min: 3, max: 80, required: true, wantErr: "title_invalid_char"},
		{name: "küfür", field: "title", value: "abc siktir", min: 3, max: 80, required: true, wantErr: "title_profanity"},

		// edge: tam sınırda
		{name: "tam min", field: "title", value: "abc", min: 3, max: 80, required: true, wantErr: ""},
		{name: "tam max", field: "title", value: strings.Repeat("a", 80), min: 3, max: 80, required: true, wantErr: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateTextMinMax(tc.field, tc.value, tc.min, tc.max, TitleRegex, tc.required)
			checkErr(t, err, tc.wantErr)
		})
	}
}

// ═══════════════════════════════════════════════════════════
// validateProfileInput (YENİ)
// ═══════════════════════════════════════════════════════════

func TestValidateProfileInput(t *testing.T) {
	tests := []struct {
		name         string
		username     string
		bio          string
		wantUsername string
		wantBio      string
		wantErr      string
	}{
		// Username validasyonu
		{name: "boş username", username: "", bio: "", wantErr: "username_empty"},
		{name: "sadece boşluk", username: "   ", bio: "", wantErr: "username_empty"},
		{name: "user_ prefix", username: "user_123", bio: "", wantErr: "username_reserved"},
		{name: "USER_ büyük harf", username: "USER_abc", bio: "", wantErr: "username_reserved"},
		{name: "rezerve admin", username: "admin", bio: "", wantErr: "username_reserved"},
		{name: "rezerve ADMIN", username: "ADMIN", bio: "", wantErr: "username_reserved"},
		{name: "nokta başlangıç", username: ".eray", bio: "", wantErr: "username_invalid_format"},
		{name: "nokta son", username: "eray.", bio: "", wantErr: "username_invalid_format"},
		{name: "çift nokta", username: "eray..gdev", bio: "", wantErr: "username_invalid_format"},
		{name: "çift alt çizgi", username: "eray__gdev", bio: "", wantErr: "username_invalid_format"},
		{name: "çift tire", username: "eray--gdev", bio: "", wantErr: "username_invalid_format"},
		{name: "çok kısa", username: "ab", bio: "", wantErr: "username_too_short"},
		{name: "küfür username", username: "siktir", bio: "", wantErr: "username_profanity"},

		// Geçerli senaryolar
		{name: "basit", username: "eray", bio: "", wantUsername: "eray", wantBio: ""},
		{name: "tire ile", username: "eray-gdev", bio: "", wantUsername: "eray-gdev", wantBio: ""},
		{name: "alt çizgi ile", username: "eray_gdev", bio: "", wantUsername: "eray_gdev", wantBio: ""},
		{name: "nokta ile", username: "eray.gdev", bio: "", wantUsername: "eray.gdev", wantBio: ""},

		// Trim + sanitize
		{name: "trim", username: "  eray  ", bio: "  hello  ", wantUsername: "eray", wantBio: "hello"},
		{name: "html tag temizleme", username: "<script>eray</script>", bio: "", wantUsername: "eray", wantBio: ""},

		// Bio validasyonu
		{name: "bio çok uzun", username: "eray", bio: strings.Repeat("a", 151), wantErr: "bio_too_long"},
		{name: "bio küfür", username: "eray", bio: "siktir", wantErr: "bio_profanity"},
		{name: "bio geçerli", username: "eray", bio: "Merhaba dünya!", wantUsername: "eray", wantBio: "Merhaba dünya!"},
		{name: "bio boş", username: "eray", bio: "", wantUsername: "eray", wantBio: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotUsername, gotBio, err := validateProfileInput(tc.username, tc.bio)
			checkErr(t, err, tc.wantErr)

			if tc.wantErr == "" {
				if gotUsername != tc.wantUsername {
					t.Errorf("username = %q, want %q", gotUsername, tc.wantUsername)
				}
				if gotBio != tc.wantBio {
					t.Errorf("bio = %q, want %q", gotBio, tc.wantBio)
				}
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════
// validateSearchQuery (YENİ)
// ═══════════════════════════════════════════════════════════

func TestValidateSearchQuery(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr string
	}{
		{name: "empty → empty", input: "", want: ""},
		{name: "sadece boşluk", input: "   ", want: ""},
		{name: "basit", input: "eray", want: "eray"},
		{name: "trim", input: "  eray  ", want: "eray"},
		{name: "alt çizgi escape", input: "a_b", want: "a\\_b"},
		{name: "geçersiz karakter", input: "<script>", wantErr: "search_invalid_char"},
		{name: "küfür", input: "siktir", wantErr: "search_profanity"},
		{name: "çok uzun", input: strings.Repeat("a", 101), wantErr: "search_too_long"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := validateSearchQuery(tc.input)
			checkErr(t, err, tc.wantErr)

			if tc.wantErr == "" && got != tc.want {
				t.Errorf("query = %q, want %q", got, tc.want)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════
// escapeLikePattern (YENİ)
// ═══════════════════════════════════════════════════════════

func TestEscapeLikePattern(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "boş", input: "", want: ""},
		{name: "düz metin", input: "abc", want: "abc"},
		{name: "yüzde", input: "a%b", want: "a\\%b"},
		{name: "alt çizgi", input: "a_b", want: "a\\_b"},
		{name: "ters slash", input: "a\\b", want: "a\\\\b"},
		{name: "ters slash + yüzde", input: "\\%", want: "\\\\\\%"},
		{name: "karışık", input: "a%b_c\\d", want: "a\\%b\\_c\\\\d"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := escapeLikePattern(tc.input)
			if got != tc.want {
				t.Errorf("escapeLikePattern(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════
// isValidUsernameFormat + isAlnum
// ═══════════════════════════════════════════════════════════

func TestIsValidUsernameFormat(t *testing.T) {
	tests := []struct {
		name     string
		username string
		want     bool
	}{
		{name: "empty", username: "", want: false},
		{name: "tek harf", username: "a", want: true},
		{name: "tek rakam", username: "1", want: true},
		{name: "basit", username: "eray", want: true},
		{name: "tire", username: "eray-gdev", want: true},
		{name: "alt çizgi", username: "eray_gdev", want: true},
		{name: "nokta", username: "eray.gdev", want: true},
		{name: "rakam son", username: "eray123", want: true},

		{name: "nokta baş", username: ".eray", want: false},
		{name: "nokta son", username: "eray.", want: false},
		{name: "alt çizgi baş", username: "_eray", want: false},
		{name: "tire son", username: "eray-", want: false},
		{name: "çift nokta", username: "eray..gdev", want: false},
		{name: "çift alt çizgi", username: "eray__gdev", want: false},
		{name: "çift tire", username: "eray--gdev", want: false},
		{name: "karışık tekrar", username: "eray-._gdev", want: true}, // - . _ sıralı ama tekrar yok
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isValidUsernameFormat(tc.username)
			if got != tc.want {
				t.Errorf("isValidUsernameFormat(%q) = %v, want %v", tc.username, got, tc.want)
			}
		})
	}
}

func TestIsAlnum(t *testing.T) {
	tests := []struct {
		name string
		r    rune
		want bool
	}{
		{name: "a", r: 'a', want: true},
		{name: "z", r: 'z', want: true},
		{name: "A", r: 'A', want: true},
		{name: "Z", r: 'Z', want: true},
		{name: "0", r: '0', want: true},
		{name: "9", r: '9', want: true},
		{name: "nokta", r: '.', want: false},
		{name: "alt çizgi", r: '_', want: false},
		{name: "tire", r: '-', want: false},
		{name: "boşluk", r: ' ', want: false},
		{name: "ç", r: 'ç', want: true},
		{name: "ğ", r: 'ğ', want: true},
		{name: "ı", r: 'ı', want: true},
		{name: "İ", r: 'İ', want: true},
		{name: "ö", r: 'ö', want: true},
		{name: "ş", r: 'ş', want: true},
		{name: "ü", r: 'ü', want: true},
		{name: "emoji", r: '😀', want: false},
		{name: "0x80 (byte değil)", r: 0x80, want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isAlnum(tc.r)
			if got != tc.want {
				t.Errorf("isAlnum(%d) = %v, want %v", tc.r, got, tc.want)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════
// GitHub URL normalizasyonu
// ═══════════════════════════════════════════════════════════

func TestNormalizeGithubImageURL(t *testing.T) {
	const want = "https://raw.githubusercontent.com/user/repo/main/img.png"
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty", input: "", want: ""},
		{name: "blob", input: "https://github.com/user/repo/blob/main/img.png", want: want},
		{name: "protokolsüz blob", input: "github.com/user/repo/blob/main/img.png", want: want},
		{name: "raw", input: "https://github.com/user/repo/raw/main/img.png", want: want},
		{name: "zaten raw", input: want, want: want},
		{name: "query strip", input: "https://github.com/user/repo/blob/main/img.png?raw=true", want: want},
		{name: "fragment strip", input: "https://github.com/user/repo/blob/main/img.png#anchor", want: want},
		{name: "harici domain", input: "https://example.com/img.png", want: ""},
		{name: "tree (blob değil)", input: "https://github.com/user/repo/tree/main", want: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeGithubImageURL(tc.input)
			if got != tc.want {
				t.Errorf("normalizeGithubImageURL(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestIsValidGithubImageURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "empty → false", input: "", want: false},
		{name: "png", input: "https://raw.githubusercontent.com/u/r/main/x.png", want: true},
		{name: "jpg", input: "https://raw.githubusercontent.com/u/r/main/x.jpg", want: true},
		{name: "jpeg", input: "https://raw.githubusercontent.com/u/r/main/x.jpeg", want: true},
		{name: "gif", input: "https://raw.githubusercontent.com/u/r/main/x.gif", want: true},
		{name: "webp", input: "https://raw.githubusercontent.com/u/r/main/x.webp", want: true},
		{name: "büyük harf PNG", input: "https://raw.githubusercontent.com/u/r/main/x.PNG", want: true},
		{name: "exe", input: "https://raw.githubusercontent.com/u/r/main/x.exe", want: false},
		{name: "uzantısız", input: "https://raw.githubusercontent.com/u/r/main/x", want: false},
		{name: "blob (normalize edilir)", input: "https://github.com/u/r/blob/main/x.png", want: true},
		{name: "harici domain", input: "https://example.com/x.png", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isValidGithubImageURL(tc.input)
			if got != tc.want {
				t.Errorf("isValidGithubImageURL(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestNormalizeGithubRepoURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty", input: "", want: ""},
		{name: "protokollü", input: "https://github.com/user/repo", want: "https://github.com/user/repo"},
		{name: "protokolsüz", input: "github.com/user/repo", want: "https://github.com/user/repo"},
		{name: ".git strip", input: "https://github.com/user/repo.git", want: "https://github.com/user/repo"},
		{name: "trailing slash", input: "https://github.com/user/repo/", want: "https://github.com/user/repo"},
		{name: "query strip", input: "https://github.com/user/repo?x=1", want: "https://github.com/user/repo"},
		{name: "fragment strip", input: "https://github.com/user/repo#readme", want: "https://github.com/user/repo"},
		{name: "extra path strip", input: "https://github.com/user/repo/tree/main", want: "https://github.com/user/repo"},
		{name: "eksik repo", input: "https://github.com/user", want: ""},
		{name: "harici domain", input: "https://example.com/u/r", want: ""},
		{name: "sadece domain", input: "https://github.com/", want: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeGithubRepoURL(tc.input)
			if got != tc.want {
				t.Errorf("normalizeGithubRepoURL(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════
// Yardımcı
// ═══════════════════════════════════════════════════════════

// checkErr — test edilen fonksiyonun error code'unu doğrular.
// wantErr == "" → err nil olmalı.
// wantErr != "" → err.Error() == wantErr olmalı.
func checkErr(t *testing.T, err error, wantErr string) {
	t.Helper()
	if wantErr == "" {
		if err != nil {
			t.Errorf("beklenmeyen hata: %v", err)
		}
		return
	}
	if err == nil {
		t.Errorf("hata bekleniyordu (%s), nil geldi", wantErr)
		return
	}
	if err.Error() != wantErr {
		t.Errorf("hata = %q, want %q", err.Error(), wantErr)
	}
}

// ═══════════════════════════════════════════════════════════
// validateUsernameParam
// ═══════════════════════════════════════════════════════════

func TestValidateUsernameParam(t *testing.T) {
	tests := []struct {
		name     string
		username string
		wantErr  string
	}{
		{name: "boş", username: "", wantErr: "username_required"},
		{name: "geçerli basit", username: "eray", wantErr: ""},
		{name: "geçerli tire", username: "eray-gdev", wantErr: ""},
		{name: "geçerli nokta", username: "eray.gdev", wantErr: ""},
		{name: "geçerli türkçe", username: "çağrı", wantErr: ""},
		{name: "tam sınır 30", username: strings.Repeat("a", 30), wantErr: ""},
		{name: "çok uzun 31", username: strings.Repeat("a", 31), wantErr: "username_too_long"},
		{name: "geçersiz @", username: "eray@x", wantErr: "username_invalid_char"},
		{name: "boşluk", username: "eray gdev", wantErr: "username_invalid_char"},
		{name: "slash", username: "eray/gdev", wantErr: "username_invalid_char"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateUsernameParam(tc.username)
			checkErr(t, err, tc.wantErr)
		})
	}
}

// ═══════════════════════════════════════════════════════════
// validateProjectInput
// ═══════════════════════════════════════════════════════════

func TestValidateProjectInput(t *testing.T) {
	validBase := ProjectInput{
		Title:       "Test Projesi",
		Description: "Bu bir test projesi aciklamasidir, yeterince uzun.",
		GithubURL:   "https://github.com/user/repo",
	}

	tests := []struct {
		name       string
		input      ProjectInput
		isCreate   bool
		wantErr    string
		wantGithub string // başarı durumunda beklenen normalize hali (opsiyonel)
	}{
		// Mutlu yol
		{name: "geçerli create", input: validBase, isCreate: true, wantErr: ""},
		{name: "geçerli update", input: validBase, isCreate: false, wantErr: ""},

		// Zorunlu alanlar
		{name: "title boş", input: ProjectInput{Description: "yeterince uzun bir aciklama metni"}, isCreate: true, wantErr: "title_and_description_required"},
		{name: "description boş", input: ProjectInput{Title: "x"}, isCreate: true, wantErr: "title_and_description_required"},

		// Uzunluk / içerik
		{name: "title kısa", input: ProjectInput{Title: "ab", Description: "yeterince uzun bir aciklama metni", GithubURL: "https://github.com/u/r"}, isCreate: true, wantErr: "title_too_short"},
		{name: "description kısa", input: ProjectInput{Title: "abc", Description: "kısa", GithubURL: "https://github.com/u/r"}, isCreate: true, wantErr: "description_too_short"},
		{name: "title profanity", input: ProjectInput{Title: "siktir", Description: "yeterince uzun bir aciklama metni", GithubURL: "https://github.com/u/r"}, isCreate: true, wantErr: "title_profanity"},

		// GitHub URL
		{name: "github boş", input: ProjectInput{Title: "abc", Description: "yeterince uzun bir aciklama metni"}, isCreate: true, wantErr: "github_url_required"},
		{name: "github geçersiz", input: ProjectInput{Title: "abc", Description: "yeterince uzun bir aciklama metni", GithubURL: "https://example.com/x"}, isCreate: true, wantErr: "invalid_github_url"},
		{name: "github normalize", input: validBase, isCreate: true, wantErr: "", wantGithub: "https://github.com/user/repo"},

		// Demo URL
		{name: "demo geçersiz", input: ProjectInput{Title: "abc", Description: "yeterince uzun bir aciklama metni", GithubURL: "https://github.com/u/r", DemoURL: "ftp://x.com"}, isCreate: true, wantErr: "invalid_demo_url"},

		// Image URL
		{name: "image harici domain", input: ProjectInput{Title: "abc", Description: "yeterince uzun bir aciklama metni", GithubURL: "https://github.com/u/r", ImageURL: "https://example.com/x.png"}, isCreate: true, wantErr: "image_url_must_be_github"},

		// Uzunluk aşımı
		{name: "title çok uzun", input: ProjectInput{Title: strings.Repeat("a", 81), Description: "yeterince uzun bir aciklama metni", GithubURL: "https://github.com/u/r"}, isCreate: true, wantErr: "title_too_long"},
		{name: "github çok uzun", input: ProjectInput{Title: "abc", Description: "yeterince uzun bir aciklama metni", GithubURL: "https://github.com/u/" + strings.Repeat("a", 200)}, isCreate: true, wantErr: "github_url_too_long"},
		{name: "demo çok uzun", input: ProjectInput{Title: "abc", Description: "yeterince uzun bir aciklama metni", GithubURL: "https://github.com/u/r", DemoURL: "https://" + strings.Repeat("a", 200) + ".com"}, isCreate: true, wantErr: "demo_url_too_long"},
		{name: "image çok uzun", input: ProjectInput{Title: "abc", Description: "yeterince uzun bir aciklama metni", GithubURL: "https://github.com/u/r", ImageURL: "https://github.com/u/r/blob/main/" + strings.Repeat("a", 300) + ".png"}, isCreate: true, wantErr: "image_url_too_long"},

		// Profanity
		{name: "description profanity", input: ProjectInput{Title: "abc", Description: "yeterince uzun bir siktir aciklama", GithubURL: "https://github.com/u/r"}, isCreate: true, wantErr: "description_profanity"},
		{name: "longDescription profanity", input: ProjectInput{Title: "abc", Description: "yeterince uzun bir aciklama metni", LongDescription: "siktir", GithubURL: "https://github.com/u/r"}, isCreate: true, wantErr: "longDescription_profanity"},

		// Başarılı normalize branch'leri
		{name: "geçerli demo url", input: ProjectInput{Title: "abc", Description: "yeterince uzun bir aciklama metni", GithubURL: "https://github.com/u/r", DemoURL: "https://demo.example.com"}, isCreate: true, wantErr: ""},
		{name: "geçerli image url", input: ProjectInput{Title: "abc", Description: "yeterince uzun bir aciklama metni", GithubURL: "https://github.com/u/r", ImageURL: "https://github.com/u/r/blob/main/img.png"}, isCreate: true, wantErr: ""},
		{name: "geçerli longDescription", input: ProjectInput{Title: "abc", Description: "yeterince uzun bir aciklama metni", LongDescription: "Uzun bir aciklama metni.", GithubURL: "https://github.com/u/r"}, isCreate: true, wantErr: ""},

		// Kategoriler
		{name: "6 kategori (max 5)", input: ProjectInput{Title: "abc", Description: "yeterince uzun bir aciklama metni", GithubURL: "https://github.com/u/r", CategoryIDs: []int{1, 2, 3, 4, 5, 6}}, isCreate: true, wantErr: "too_many_categories"},

		// MaxContributors
		{name: "create default atanır", input: validBase, isCreate: true, wantErr: ""},
		{name: "create 15 geçersiz", input: ProjectInput{Title: "abc", Description: "yeterince uzun bir aciklama metni", GithubURL: "https://github.com/u/r", MaxContributors: 15}, isCreate: true, wantErr: "invalid_contributor_limit"},
		{name: "update maxContributors yok sayılır", input: ProjectInput{Title: "abc", Description: "yeterince uzun bir aciklama metni", GithubURL: "https://github.com/u/r", MaxContributors: 15}, isCreate: false, wantErr: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := validateProjectInput(tc.input, tc.isCreate)
			checkErr(t, err, tc.wantErr)

			if tc.wantErr == "" && tc.isCreate && got.MaxContributors == 0 {
				t.Errorf("create'te MaxContributors default atanmalıydı, 0 kaldı")
			}
			if tc.wantGithub != "" && got.GithubURL != tc.wantGithub {
				t.Errorf("GithubURL = %q, want %q", got.GithubURL, tc.wantGithub)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════
// canChangeUsername
// ═══════════════════════════════════════════════════════════

func TestCanChangeUsername(t *testing.T) {
	now := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)

	oneHourAgo := now.Add(-1 * time.Hour)
	fortySevenHoursAgo := now.Add(-47 * time.Hour)
	fortyEightHoursAgo := now.Add(-48 * time.Hour)
	seventyTwoHoursAgo := now.Add(-72 * time.Hour)

	tests := []struct {
		name          string
		lastChangedAt *time.Time
		now           time.Time
		want          bool
	}{
		{name: "hiç değişmemiş (nil)", lastChangedAt: nil, now: now, want: true},
		{name: "1 saat önce — cooldown", lastChangedAt: &oneHourAgo, now: now, want: false},
		{name: "47 saat önce — cooldown", lastChangedAt: &fortySevenHoursAgo, now: now, want: false},
		{name: "48 saat önce — tam sınır", lastChangedAt: &fortyEightHoursAgo, now: now, want: true},
		{name: "72 saat önce — serbest", lastChangedAt: &seventyTwoHoursAgo, now: now, want: true},
		{name: "1 saniye kala — cooldown", lastChangedAt: ptrTime(now.Add(-48*time.Hour + time.Second)), now: now, want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := canChangeUsername(tc.lastChangedAt, tc.now)
			if got != tc.want {
				t.Errorf("canChangeUsername() = %v, want %v", got, tc.want)
			}
		})
	}
}

// ptrTime — &time.Time döndürmek için küçük yardımcı.
func ptrTime(t time.Time) *time.Time {
	return &t
}
