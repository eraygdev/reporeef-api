-- ═══════════════════════════════════════════════════════════
-- 007 — projects.updated_at trigger
-- ═══════════════════════════════════════════════════════════
-- projects tablosunda updated_at kolonunu otomatik güncelle.
-- update_updated_at_column() fonksiyonu 001'de tanımlı.
-- Böylece handler'da elle NOW() yazmaya gerek kalmaz,
-- project_contributors ile aynı davranış.
-- ═══════════════════════════════════════════════════════════

DROP TRIGGER IF EXISTS set_projects_updated_at ON projects;
CREATE TRIGGER set_projects_updated_at
    BEFORE UPDATE ON projects
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();