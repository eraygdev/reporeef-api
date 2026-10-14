package app

import (
	"regexp"
	"time"
)

// ─────────────────────────────────────────────────────────
// API VERSION
// ─────────────────────────────────────────────────────────
const APIVersion = "v1"

// Profil limitleri
const (
	MinUsernameLen       = 3
	MaxUsernameLen       = 30
	MaxGithubUsernameLen = 39
	MinBioLen            = 0
	MaxBioLen            = 150
)

// UsernameChangeCooldown — username değiştirdikten sonra beklenmesi gereken süre.
// 48 saat ≈ haftada 3 değişiklik.
const UsernameChangeCooldown = 48 * time.Hour

// Proje limitleri
const (
	MinTitleLen        = 3
	MaxTitleLen        = 80
	MinDescriptionLen  = 20
	MaxDescriptionLen  = 150
	MaxLongDescLen     = 500
	MaxGithubURLLen    = 200
	MaxDemoURLLen      = 200
	MaxImageURLLen     = 300
	MaxImageSizeBytes  = 2 * 1024 * 1024 // 2 MB
	MaxCategories      = 5
	MaxProjectsPerUser = 10
)

// [YENİ] Katkıcı limitleri
const (
	DefaultContributorLimit = 10
)

// [YENİ] AllowedContributorLimits — create'te seçilebilecek değerler
var AllowedContributorLimits = []int{5, 10, 20, 50}

// Arama limitleri
const (
	MaxSearchLen = 100
)

// Request body limitleri — DoS koruması.
// En büyük payload (bio + longDesc + contributor message) toplamı <10KB.
// 1MB rahat bir üst sınır, bolca headroom bırakır.
const MaxRequestBodyBytes = 1 << 20 // 1 MB

// Sayfalama limitleri
const (
	MaxProjectsPerPage = 20
	MaxUsersPerSearch  = 5
)

// Karakter filtreleri (frontend validators.js ile uyumlu)
var (
	TitleRegex    = regexp.MustCompile(`^[a-zA-Z0-9çÇğĞıİöÖşŞüÜ_. ]*$`)
	UsernameRegex = regexp.MustCompile(`^[a-zA-Z0-9çÇğĞıİöÖşŞüÜ_.\-]*$`)
	TextRegex     = regexp.MustCompile(`^[a-zA-Z0-9çÇğĞıİöÖşŞüÜ.,!?;:'"()\[\]{}\-_/|@#$%&*+=~\s]*$`)
	BioRegex      = regexp.MustCompile(`^[a-zA-Z0-9çÇğĞıİöÖşŞüÜ.,!?;:'"()\[\]{}\-_/|@#$%&*+=~\s]*$`)
)

// ReservedUsernames — sistem veya gelecekteki route'lar için ayrılmış isimler.
// Kullanıcılar bu isimleri alamaz (case-insensitive).
var ReservedUsernames = map[string]bool{
	"admin": true, "root": true, "api": true, "www": true, "mail": true,
	"support": true, "help": true, "about": true, "settings": true,
	"login": true, "register": true, "logout": true, "profile": true,
	"dashboard": true, "project": true, "projects": true, "create-project": true,
	"onboarding": true, "search": true, "explore": true, "user": true,
	"users": true, "reporeef": true, "official": true, "system": true,
	"moderator": true, "security": true, "billing": true, "premium": true,
}

// [YENİ] isAllowedContributorLimit — değer izinli listede mi?
func isAllowedContributorLimit(limit int) bool {
	for _, allowed := range AllowedContributorLimits {
		if limit == allowed {
			return true
		}
	}
	return false
}
