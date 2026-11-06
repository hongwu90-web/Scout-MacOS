-- 1. Add read_at timestamp column to items (0 = unread / not set)
ALTER TABLE items ADD COLUMN read_at INTEGER NOT NULL DEFAULT 0;

-- Backfill legacy read items with created_at so they have a baseline read_at
UPDATE items SET read_at = created_at WHERE unread = 0 AND read_at = 0;

-- 2. Optimize FTS5 update trigger so it only fires when title or content changes (not on read/unread status updates)
DROP TRIGGER IF EXISTS items_fts_items_au;

CREATE TRIGGER IF NOT EXISTS items_fts_items_au AFTER UPDATE OF title, content ON items BEGIN
	DELETE FROM items_fts WHERE rowid = old.id;
	INSERT INTO items_fts(rowid, title, content)
	VALUES (new.id, new.title, new.content);
END;

-- 3. Replace idx_items_user_feed_unread with covering index including ordering
DROP INDEX IF EXISTS idx_items_user_feed_unread;
CREATE INDEX IF NOT EXISTS idx_items_user_feed_unread_pubdate ON items(user_id, feed_id, unread, pub_date DESC, id DESC);

-- 4. Index for user_id + pub_date ordering (eliminates temp B-tree on "All" feed view)
CREATE INDEX IF NOT EXISTS idx_items_user_pubdate ON items(user_id, pub_date DESC, id DESC);

-- 5. Index for expiring read items quickly by read_at
CREATE INDEX IF NOT EXISTS idx_items_user_unread_read_at ON items(user_id, unread, read_at);
