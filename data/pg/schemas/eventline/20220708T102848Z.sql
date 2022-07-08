ALTER TABLE project_notification_settings
  DROP COLUMN recipient_account_ids,
  ADD COLUMN email_addresses VARCHAR[] NOT NULL DEFAULT ARRAY[]::VARCHAR[];
