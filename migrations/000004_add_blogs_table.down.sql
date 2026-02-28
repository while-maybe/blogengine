DROP TRIGGER IF EXISTS trg_blogs_updated_at;

DROP INDEX IF EXISTS idx_blogs_created_at;
DROP INDEX IF EXISTS idx_blogs_slug;
DROP INDEX IF EXISTS idx_blogs_slug_active;

DROP TABLE IF EXISTS blogs;