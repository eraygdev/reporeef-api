-- ═══════════════════════════════════════════════════════════
-- 008 — Contributors sosyal katman
-- ═══════════════════════════════════════════════════════════
-- users.last_seen_at: katkıcılar arası aktiflik göstergesi için.
-- projects.whatsapp_invite_url: proje bazlı WhatsApp davet linki.
-- ═══════════════════════════════════════════════════════════

ALTER TABLE users ADD COLUMN last_seen_at TIMESTAMPTZ;
ALTER TABLE projects ADD COLUMN whatsapp_invite_url TEXT;