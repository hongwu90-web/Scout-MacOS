-- Performance optimization indexes for list queries and unread counts

CREATE INDEX IF NOT EXISTS idx_items_user_unread_pubdate ON items(user_id, unread, pub_date DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_items_user_feed_unread ON items(user_id, feed_id, unread);
CREATE INDEX IF NOT EXISTS idx_snapshots_user_page_created ON page_snapshots(user_id, page_id, created_at DESC);
