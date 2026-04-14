ALTER TABLE filters
    ADD COLUMN IF NOT EXISTS response_media_type TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS response_media_file_id TEXT NOT NULL DEFAULT '';
