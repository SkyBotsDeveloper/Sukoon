CREATE UNIQUE INDEX IF NOT EXISTS bot_instances_one_active_clone_per_owner
ON bot_instances (created_by_user_id)
WHERE created_by_user_id <> 0 AND is_primary = FALSE AND status = 'active';
