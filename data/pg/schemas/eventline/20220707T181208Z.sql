CREATE TABLE notifications
  (id KSUID PRIMARY KEY,
   project_id KSUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
   recipients VARCHAR[] NOT NULL,
   message TEXT NOT NULL,
   next_delivery_time TIMESTAMP NOT NULL,
   delivery_delay INT NOT NULL);

CREATE INDEX notifications_project_id_idx
  ON notifications (project_id);

CREATE INDEX notifications_next_delivery_time_idx
  ON notifications (next_delivery_time);
