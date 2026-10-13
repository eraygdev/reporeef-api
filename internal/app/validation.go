package app

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/eraygdev/reporeef-api/internal/profanity"
)

// validateText uzunluk + regex + küfür kontrolü yapar.
// Hata durumunda error code döner (ör. "title_too_long", "title_invalid_char", "title_profanity").
func validateText(field, value string, max int, re *regexp.Regexp) error {
	if utf8.RuneCountInString(value) > max {
		return fmt.Errorf("%s_too_long", field)
	}

	if value == "" {
		return nil
	}

	if !re.MatchString(value) {
		return fmt.Errorf("%s_invalid_char", field)
	}

	if profanity.Contains(value) {
		return fmt.Errorf("%s_profanity", field)
	}

	return nil
}

// validateTextMinMax min + max + regex + küfür kontrolü yapar.
// required=true ise boş değer kabul edilmez.
func validateTextMinMax(field, value string, min, max int, re *regexp.Regexp, required bool) error {
	length := utf8.RuneCountInString(value)

	if required && length < min {
		return fmt.Errorf("%s_too_short", field)
	}
	if !required && length > 0 && length < min {
		return fmt.Errorf("%s_too_short", field)
	}
	if length > max {
		return fmt.Errorf("%s_too_long", field)
	}
	if length == 0 {
		return nil
	}
	if !re.MatchString(value) {
		return fmt.Errorf("%s_invalid_char", field)
	}
	if profanity.Contains(value) {
		return fmt.Errorf("%s_profanity", field)
	}
	return nil
}

// normalizeGithubImageURL — GitHub görsel URL'lerini raw formata çevirir.
// Kullanıcı hangi formatta yapıştırırsa yapıştırsın, tek formata indirger.
// Geçersizse boş string döner.
func normalizeGithubImageURL(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}

	// Protokol yoksa https:// ekle (sadece github domain'leri için)
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		if strings.HasPrefix(rawURL, "github.com/") || strings.HasPrefix(rawURL, "raw.githubusercontent.com/") {
			rawURL = "https://" + rawURL
		}
	}

	if idx := strings.Index(rawURL, "?"); idx != -1 {
		rawURL = rawURL[:idx]
	}
	if idx := strings.Index(rawURL, "#"); idx != -1 {
		rawURL = rawURL[:idx]
	}

	if strings.HasPrefix(rawURL, "https://raw.githubusercontent.com/") {
		return rawURL
	}

	const blobPrefix = "https://github.com/"
	const rawPrefix = "https://raw.githubusercontent.com/"

	if !strings.HasPrefix(rawURL, blobPrefix) {
		return ""
	}

	rest := strings.TrimPrefix(rawURL, blobPrefix)

	if idx := strings.Index(rest, "/blob/"); idx != -1 {
		rest = rest[:idx] + "/" + rest[idx+len("/blob/"):]
	} else if idx := strings.Index(rest, "/raw/"); idx != -1 {
		rest = rest[:idx] + "/" + rest[idx+len("/raw/"):]
	} else {
		return ""
	}

	return rawPrefix + rest
}

// isValidGithubImageURL — normalize edilmiş URL geçerli mi?
// Uzantı kontrolü + path'in en az 4 parça olması (user/repo/branch/file).
func isValidGithubImageURL(rawURL string) bool {
	normalized := normalizeGithubImageURL(rawURL)
	if normalized == "" {
		return false
	}

	lower := strings.ToLower(normalized)
	allowedExts := []string{".png", ".jpg", ".jpeg", ".gif", ".webp"}
	for _, ext := range allowedExts {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}

// normalizeGithubRepoURL — GitHub repo URL'sini normalize eder.
func normalizeGithubRepoURL(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}

	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		if strings.HasPrefix(rawURL, "github.com/") {
			rawURL = "https://" + rawURL
		}
	}

	if idx := strings.Index(rawURL, "?"); idx != -1 {
		rawURL = rawURL[:idx]
	}
	if idx := strings.Index(rawURL, "#"); idx != -1 {
		rawURL = rawURL[:idx]
	}

	rawURL = strings.TrimSuffix(rawURL, ".git")
	rawURL = strings.TrimSuffix(rawURL, "/")

	const prefix = "https://github.com/"
	if !strings.HasPrefix(rawURL, prefix) {
		return ""
	}
	rest := strings.TrimPrefix(rawURL, prefix)

	parts := strings.Split(rest, "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return ""
	}

	return prefix + parts[0] + "/" + parts[1]
}

// checkGithubRepoPublic — git-upload-pack endpoint'i ile repo'nun public olup olmadığını kontrol eder.
// API rate limiti yoktur. Fail-open: ağ hatası durumunda nil döner.
func checkGithubRepoPublic(repoURL string) error {
	normalized := normalizeGithubRepoURL(repoURL)
	if normalized == "" {
		return fmt.Errorf("invalid_github_url")
	}

	checkURL := normalized + "/info/refs?service=git-upload-pack"

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("GET", checkURL, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", "RepoReef-Backend")

	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("github_repo_not_accessible")
	}
	return nil
}

// isValidUsernameFormat — baş/son noktalama ve arka arkaya tekrar kontrolü.
// `UsernameRegex` karakter setini kontrol eder, bu fonksiyon düzeni kontrol eder.
func isValidUsernameFormat(username string) bool {
	if username == "" {
		return false
	}

	// Baş/son karakterler harf veya rakam olmalı
	first := username[0]
	last := username[len(username)-1]
	if !isAlnum(first) || !isAlnum(last) {
		return false
	}

	// Arka arkaya aynı noktalama karakterleri yasak
	if strings.Contains(username, "..") ||
		strings.Contains(username, "__") ||
		strings.Contains(username, "--") {
		return false
	}

	return true
}

// isAlnum — ASCII harf veya rakam mı?
// Türkçe karakterleri de kabul eder (byte bazlı kontrol yerine rune).
func isAlnum(b byte) bool {
	return (b >= 'a' && b <= 'z') ||
		(b >= 'A' && b <= 'Z') ||
		(b >= '0' && b <= '9') ||
		b >= 0x80 // UTF-8 continuation / Türkçe karakter başlangıcı
}

// validateProfileInput — kullanıcı profil güncellemesi için username + bio validasyonu.
// Saf (pure) — handler'dan bağımsız, test edilebilir.
// Hata durumunda error code string'i döner (örn. "username_reserved").
//
// Dönüş: (temizlenmiş username, temizlenmiş bio, hata)
func validateProfileInput(username, bio string) (string, string, error) {
	username = sanitizeText(strings.TrimSpace(username))
	bio = sanitizeText(strings.TrimSpace(bio))

	if username == "" {
		return "", "", fmt.Errorf("username_empty")
	}

	// `user_` prefix reserved — geçici onboarding isimleri ile çakışmasın
	if strings.HasPrefix(strings.ToLower(username), "user_") {
		return "", "", fmt.Errorf("username_reserved")
	}

	// Rezerve isim listesi
	if ReservedUsernames[strings.ToLower(username)] {
		return "", "", fmt.Errorf("username_reserved")
	}

	// Format — baş/son noktalama + arka arkaya tekrar
	if !isValidUsernameFormat(username) {
		return "", "", fmt.Errorf("username_invalid_format")
	}

	if err := validateTextMinMax("username", username, MinUsernameLen, MaxUsernameLen, UsernameRegex, true); err != nil {
		return "", "", err
	}
	if err := validateText("bio", bio, MaxBioLen, BioRegex); err != nil {
		return "", "", err
	}

	return username, bio, nil
}

// validateSearchQuery — kullanıcı araması için sorgu validasyonu + LIKE escape.
// Boş sorgu geçerlidir, boş string döner (nil hata).
//
// Dönüş: (escape edilmiş sorgu, hata)
func validateSearchQuery(query string) (string, error) {
	query = strings.TrimSpace(query)

	if query == "" {
		return "", nil
	}

	if err := validateText("search", query, MaxSearchLen, UsernameRegex); err != nil {
		return "", err
	}

	return escapeLikePattern(query), nil
}

// escapeLikePattern — PostgreSQL LIKE için %, _, \ kaçışı.
// Sıra önemli: önce \ (escape karakteri), sonra % ve _.
func escapeLikePattern(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
}

// validateUsernameParam — URL path parametresi olarak gelen username'i doğrular.
//
// Kullanıcının DB'de var olduğunu kontrol etmez (o handler'ın işi).
// Sadece giriş formatını doğrular: boş değil, makul uzunlukta, geçerli karakter.
//
// Reserved/profanity kontrolü YAPMAZ — bu endpoint kayıtlı kullanıcıyı
// getirmek için kullanılır; eski/uzun kayıtlar sorun çıkarmasın.
func validateUsernameParam(username string) error {
	if username == "" {
		return fmt.Errorf("username_required")
	}

	if utf8.RuneCountInString(username) > MaxUsernameLen {
		return fmt.Errorf("username_too_long")
	}

	if !UsernameRegex.MatchString(username) {
		return fmt.Errorf("username_invalid_char")
	}

	return nil
}

// ─────────────────────────────────────────────────────────
// PROJECT VALIDATION
// ─────────────────────────────────────────────────────────

// ProjectInput — create/update handler'larının ortak girdisi.
type ProjectInput struct {
	Title           string `json:"title"`
	Description     string `json:"description"`
	LongDescription string `json:"longDescription"`
	GithubURL       string `json:"githubUrl"`
	DemoURL         string `json:"demoUrl"`
	ImageURL        string `json:"imageUrl"`
	CategoryIDs     []int  `json:"categoryIds"`
	MaxContributors int    `json:"maxContributors"`
}

// validateProjectInput — proje create/update için saf validasyon + normalizasyon.
//
// Sadece format/uzunluk/içerik doğrular. Şunları YAPMAZ:
//   - GitHub repo public kontrolü (HTTP) → handler'da: checkGithubRepoPublic
//   - Görsel boyut kontrolü (HTTP) → handler'da: checkGithubImageSize
//   - Kategori ID geçerliliği (DB) → handler'da: validateCategoryIDs
//
// isCreate=true  → MaxContributors boşsa default ata, izinli listede mi kontrol et
// isCreate=false → MaxContributors yok sayılır (edit'te değişmez)
//
// Dönüş: sanitize + normalize edilmiş ProjectInput + error
func validateProjectInput(input ProjectInput, isCreate bool) (ProjectInput, error) {
	if input.Title == "" || input.Description == "" {
		return input, fmt.Errorf("title_and_description_required")
	}

	if isCreate {
		if input.MaxContributors == 0 {
			input.MaxContributors = DefaultContributorLimit
		}
		if !isAllowedContributorLimit(input.MaxContributors) {
			return input, fmt.Errorf("invalid_contributor_limit")
		}
	}

	// Sanitize
	input.Title = sanitizeText(input.Title)
	input.Description = sanitizeText(input.Description)
	input.LongDescription = sanitizeText(input.LongDescription)

	// Metin validasyonları (min + max + regex + profanity)
	if err := validateTextMinMax("title", input.Title, MinTitleLen, MaxTitleLen, TitleRegex, true); err != nil {
		return input, err
	}
	if err := validateTextMinMax("description", input.Description, MinDescriptionLen, MaxDescriptionLen, TextRegex, true); err != nil {
		return input, err
	}
	if err := validateTextMinMax("longDescription", input.LongDescription, 0, MaxLongDescLen, TextRegex, false); err != nil {
		return input, err
	}

	// URL uzunlukları
	if utf8.RuneCountInString(input.GithubURL) > MaxGithubURLLen {
		return input, fmt.Errorf("github_url_too_long")
	}
	if utf8.RuneCountInString(input.DemoURL) > MaxDemoURLLen {
		return input, fmt.Errorf("demo_url_too_long")
	}
	if utf8.RuneCountInString(input.ImageURL) > MaxImageURLLen {
		return input, fmt.Errorf("image_url_too_long")
	}

	// GitHub URL zorunlu + normalize
	if strings.TrimSpace(input.GithubURL) == "" {
		return input, fmt.Errorf("github_url_required")
	}
	normalizedRepo := normalizeGithubRepoURL(input.GithubURL)
	if normalizedRepo == "" {
		return input, fmt.Errorf("invalid_github_url")
	}
	input.GithubURL = normalizedRepo

	// Demo URL
	if input.DemoURL != "" &&
		!strings.HasPrefix(input.DemoURL, "http://") &&
		!strings.HasPrefix(input.DemoURL, "https://") {
		return input, fmt.Errorf("invalid_demo_url")
	}

	// Image URL
	if input.ImageURL != "" {
		normalized := normalizeGithubImageURL(input.ImageURL)
		if normalized == "" || !isValidGithubImageURL(normalized) {
			return input, fmt.Errorf("image_url_must_be_github")
		}
		input.ImageURL = normalized
	}

	// Kategori sayısı
	if len(input.CategoryIDs) > MaxCategories {
		return input, fmt.Errorf("too_many_categories")
	}

	return input, nil
}

// canChangeUsername — son username değişikliğinden bu yana yeterli süre geçti mi?
//
// lastChangedAt nil → hiç değişmemiş, izin ver.
// Aksi halde now - lastChangedAt >= UsernameChangeCooldown olmalı.
//
// Pure fonksiyon — now dışarıdan alınır (test edilebilirlik için).
func canChangeUsername(lastChangedAt *time.Time, now time.Time) bool {
	if lastChangedAt == nil {
		return true
	}
	return now.Sub(*lastChangedAt) >= UsernameChangeCooldown
}
