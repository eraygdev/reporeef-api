DROP INDEX IF EXISTS idx_users_username_lower_unique;

ALTER TABLE users DROP COLUMN IF EXISTS onboarded;
ALTER TABLE users DROP COLUMN IF EXISTS github_username;