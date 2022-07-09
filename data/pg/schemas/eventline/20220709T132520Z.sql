ALTER TABLE subscriptions
  RENAME COLUMN last_update TO last_update_time;

ALTER TABLE subscriptions
  RENAME COLUMN next_update TO next_update_time;
