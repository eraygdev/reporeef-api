-- ═══════════════════════════════════════════════════════════
-- 005 — GitHub username + Onboarding + Case-insensitive username
-- ═══════════════════════════════════════════════════════════

-- ─────────────────────────────────────────────────────────
-- 1. github_username kolonu
-- ─────────────────────────────────────────────────────────
-- GitHub login'i (kullanıcı tarafından değiştirilemez, her login'de senkronize edilir)
ALTER TABLE users ADD COLUMN github_username VARCHAR(39);
-- ─────────────────────────────────────────────────────────
-- 2. onboarded flag
-- ─────────────────────────────────────────────────────────
-- false → kullanıcı henüz kendi username'ini seçmedi
-- İlk login'de user_{github_id} geçici ismiyle gelir, onboarding sonrası true olur
ALTER TABLE users ADD COLUMN onboarded BOOLEAN NOT NULL DEFAULT false;

-- Mevcut kullanıcılar onboarding'i geçmiş sayılır
UPDATE users SET onboarded = true;

-- ─────────────────────────────────────────────────────────
-- 3. Case-insensitive username unique
-- ─────────────────────────────────────────────────────────
-- Mevcut "username UNIQUE" case-sensitive kalıyor;
-- ek olarak LOWER(username) için unique index ekliyoruz.
-- Böylece "Eray" ve "eray" aynı anda var olamaz.
CREATE UNIQUE INDEX idx_users_username_lower_unique ON users (LOWER(username));