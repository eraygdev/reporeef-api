-- ═══════════════════════════════════════════════════════════
-- 006 — Username Change Cooldown
-- ═══════════════════════════════════════════════════════════
-- Kullanıcıların çok sık username değiştirmesini engellemek için
-- son değişiklik zamanını takip eder. 48 saat cooldown uygulanır.
-- ═══════════════════════════════════════════════════════════

ALTER TABLE users ADD COLUMN username_changed_at TIMESTAMPTZ;