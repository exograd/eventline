CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE DOMAIN KSUID
  AS VARCHAR COLLATE "C" CHECK (length(VALUE) = 27);

CREATE DOMAIN CONNECTOR_NAME
  AS VARCHAR CHECK (length(VALUE) > 0);

CREATE DOMAIN EVENT_NAME
  AS VARCHAR CHECK (length(VALUE) > 0);

CREATE TYPE ACCOUNT_ROLE
  AS ENUM ('user', 'admin');

CREATE FUNCTION generate_ksuid()
  RETURNS KSUID
AS $$
DECLARE
  v_time TIMESTAMP;
  v_seconds NUMERIC;
  v_payload BYTEA;
  v_numeric NUMERIC;
  v_base62 TEXT;
  v_epoch NUMERIC = 1400000000;
  v_alphabet CHAR ARRAY[62] :=
    ARRAY['0', '1', '2', '3', '4', '5', '6', '7', '8', '9',
          'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J',
          'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T',
          'U', 'V', 'W', 'X', 'Y', 'Z',
          'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j',
          'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't',
          'u', 'v', 'w', 'x', 'y', 'z'];
  v_i INTEGER := 0;
BEGIN
  v_time := clock_timestamp();
  v_seconds := EXTRACT(EPOCH FROM v_time)::INT4 - v_epoch;
  v_payload := gen_random_bytes(16);

  v_numeric := v_seconds * pow(2::NUMERIC, 128);
  WHILE v_i < 16 LOOP
    v_i := v_i + 1;
    v_numeric := v_numeric
               + (get_byte(v_payload, v_i-1) * pow(2::NUMERIC, (16-v_i)*8));
  END LOOP;

  v_base62 := '';
  WHILE v_numeric <> 0 LOOP
    v_base62 := v_base62 || v_alphabet[mod(v_numeric, 62) + 1];
    v_numeric := div(v_numeric, 62);
  END LOOP;

  v_base62 := reverse(v_base62);
  v_base62 := lpad(v_base62, 27, '0');

  RETURN v_base62;
END
$$ LANGUAGE PLPGSQL;

CREATE TABLE projects
  (id KSUID PRIMARY KEY,
   name VARCHAR NOT NULL UNIQUE,
   creation_time TIMESTAMP NOT NULL,
   update_time TIMESTAMP NOT NULL);

CREATE TABLE project_settings
  (id KSUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
   code_header VARCHAR NOT NULL);

CREATE TABLE project_notification_settings
  (id KSUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
   on_successful_job BOOLEAN DEFAULT FALSE NOT NULL,
   on_first_successful_job BOOLEAN DEFAULT FALSE NOT NULL,
   on_failed_job BOOLEAN DEFAULT FALSE NOT NULL,
   on_aborted_job BOOLEAN DEFAULT FALSE NOT NULL,
   on_identity_refresh_error BOOLEAN DEFAULT FALSE NOT NULL,
   email_addresses VARCHAR[] NOT NULL);

CREATE INDEX project_notification_settings_id_idx
  ON project_notification_settings (id);

CREATE TABLE accounts
  (id KSUID PRIMARY KEY,
   creation_time TIMESTAMP NOT NULL,
   username VARCHAR NOT NULL UNIQUE,
   salt BYTEA NOT NULL,
   password_hash BYTEA NOT NULL,
   role ACCOUNT_ROLE NOT NULL,
   last_login_time TIMESTAMP,
   last_project_id KSUID REFERENCES projects (id),
   settings JSONB NOT NULL
     CHECK (jsonb_typeof(settings) = 'object'));

CREATE TABLE sessions
  (id KSUID PRIMARY KEY,
   account_id KSUID NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
   creation_time TIMESTAMP NOT NULL,
   update_time TIMESTAMP NOT NULL,
   data JSONB NOT NULL
     CHECK (jsonb_typeof(account_settings) = 'object'),
   account_role ACCOUNT_ROLE NOT NULL,
   account_settings JSONB NOT NULL
     CHECK (jsonb_typeof(account_settings) = 'object'));

CREATE INDEX sessions_account_id_idx
  ON sessions (account_id);

CREATE INDEX sessions_data_project_id_idx
  ON sessions ((data->>'project_id'));

CREATE INDEX project_settings_id_idx
  ON project_settings (id);

CREATE TABLE api_keys
  (id KSUID PRIMARY KEY,
   account_id KSUID NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
   name VARCHAR NOT NULL,
   creation_time TIMESTAMP NOT NULL,
   last_use_time TIMESTAMP,
   key_hash BYTEA NOT NULL,

   UNIQUE (account_id, name));

CREATE INDEX api_keys_account_id_idx
  ON api_keys (account_id);

CREATE INDEX api_keys_key_hash_idx
  ON api_keys (key_hash);

CREATE DOMAIN IDENTITY_TYPE
  AS VARCHAR CHECK (length(VALUE) > 0);

CREATE TYPE IDENTITY_STATUS
  AS ENUM ('pending', 'ready', 'error');

CREATE TABLE identities
  (id KSUID PRIMARY KEY,
   project_id KSUID REFERENCES projects (id),
   name VARCHAR NOT NULL,
   status IDENTITY_STATUS NOT NULL,
   creation_time TIMESTAMP NOT NULL,
   update_time TIMESTAMP NOT NULL,
   last_use_time TIMESTAMP,
   connector CONNECTOR_NAME NOT NULL,
   type IDENTITY_TYPE NOT NULL,
   data BYTEA NOT NULL,
   error_message VARCHAR,
   refresh_time TIMESTAMP,

   UNIQUE (project_id, name));

CREATE INDEX identities_project_id_idx
  ON identities (project_id);

CREATE INDEX identities_refresh_time_idx
  ON identities (refresh_time);

CREATE TABLE jobs
  (id KSUID PRIMARY KEY,
   project_id KSUID REFERENCES projects (id),
   creation_time TIMESTAMP NOT NULL,
   update_time TIMESTAMP NOT NULL,
   disabled BOOLEAN NOT NULL,
   spec JSONB NOT NULL);

CREATE UNIQUE INDEX jobs_project_id_spec_name_idx
  ON jobs (project_id, (spec->>'name'));

CREATE INDEX jobs_spec_trigger_identity_idx
  ON jobs ((spec->'trigger'->>'identity'))
  WHERE spec->'trigger' IS NOT NULL;

CREATE INDEX jobs_spec_runner_identity_idx
  ON jobs ((spec->'runner'->>'identity'))
  WHERE spec->'runner' IS NOT NULL;

CREATE INDEX jobs_spec_identities_idx
  ON jobs USING GIN ((spec->'identities'));

CREATE TABLE favourite_jobs
  (account_id KSUID NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
   project_id KSUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
   job_id KSUID NOT NULL REFERENCES jobs (id) ON DELETE CASCADE,

   PRIMARY KEY (account_id, project_id, job_id));

CREATE TABLE events
  (id KSUID PRIMARY KEY,
   project_id KSUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
   job_id KSUID REFERENCES jobs (id) ON DELETE CASCADE,
   creation_time TIMESTAMP NOT NULL,
   event_time TIMESTAMP NOT NULL,
   connector CONNECTOR_NAME NOT NULL,
   name EVENT_NAME NOT NULL,
   data JSONB NOT NULL
     CHECK (jsonb_typeof(data) = 'object'),
   processed BOOLEAN NOT NULL,
   original_event_id KSUID REFERENCES events (id) ON DELETE SET NULL);

CREATE INDEX events_processed_idx
  ON events (processed);

CREATE INDEX events_job_id_idx
  ON events (job_id);

CREATE TYPE JOB_EXECUTION_STATUS AS ENUM
  ('created',
   'started',
   'aborted',
   'successful',
   'failed');

CREATE TABLE job_executions
  (id KSUID PRIMARY KEY,
   project_id KSUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
   job_id KSUID NOT NULL REFERENCES jobs (id) ON DELETE CASCADE,
   job_spec JSONB NOT NULL
     CHECK (jsonb_typeof(job_spec) = 'object'),
   event_id KSUID REFERENCES events (id),
   parameters JSONB
     CHECK (jsonb_typeof(parameters) = 'object'),
   creation_time TIMESTAMP NOT NULL,
   update_time TIMESTAMP NOT NULL,
   scheduled_time TIMESTAMP NOT NULL,
   status JOB_EXECUTION_STATUS NOT NULL,
   start_time TIMESTAMP,
   end_time TIMESTAMP,
   refresh_time TIMESTAMP,
   expiration_time TIMESTAMP,
   failure_message VARCHAR);

CREATE INDEX job_executions_project_id_idx
  ON job_executions (project_id);

CREATE INDEX job_executions_job_id_idx
  ON job_executions (job_id);

CREATE INDEX job_executions_event_id_idx
  ON job_executions (event_id);

CREATE INDEX job_executions_scheduled_time_idx
  ON job_executions (scheduled_time);

CREATE INDEX job_executions_expiration_time_idx
  ON job_executions (expiration_time);

CREATE INDEX job_executions_status_idx
  ON job_executions (status);

CREATE TYPE STEP_EXECUTION_STATUS AS ENUM
  ('created',
   'started',
   'aborted',
   'successful',
   'failed');

CREATE TABLE step_executions
  (id KSUID PRIMARY KEY,
   project_id KSUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
   job_execution_id KSUID NOT NULL
     REFERENCES job_executions (id) ON DELETE CASCADE,
   position SMALLINT NOT NULL CHECK (position > 0),
   status STEP_EXECUTION_STATUS NOT NULL,
   start_time TIMESTAMP,
   end_time TIMESTAMP,
   failure_message VARCHAR,
   output VARCHAR);

CREATE INDEX step_executions_project_id_idx
  ON step_executions (project_id);

CREATE INDEX step_executions_job_execution_id_position_idx
  ON step_executions (job_execution_id, position);

CREATE TYPE SUBSCRIPTION_STATUS
  AS ENUM ('inactive', 'active', 'terminating');

CREATE SEQUENCE subscription_op;

CREATE TABLE subscriptions
  (id KSUID PRIMARY KEY,
   project_id KSUID REFERENCES projects (id),
   job_id KSUID REFERENCES jobs (id),
   identity_id KSUID REFERENCES identities (id),
   connector CONNECTOR_NAME NOT NULL,
   event EVENT_NAME NOT NULL,
   parameters JSONB NOT NULL
     CHECK (jsonb_typeof(parameters) = 'object'),
   creation_time TIMESTAMP NOT NULL,
   status SUBSCRIPTION_STATUS NOT NULL,
   update_delay INTEGER,
   last_update_time TIMESTAMP,
   next_update_time TIMESTAMP,
   op BIGINT DEFAULT nextval('subscription_op'));

CREATE INDEX subscriptions_job_id_idx
  ON subscriptions (job_id);

CREATE INDEX subscriptions_identity_id_idx
  ON subscriptions (identity_id);

CREATE INDEX subscriptions_status_idx
  ON subscriptions (status);

CREATE INDEX subscriptions_next_update_time_idx
  ON subscriptions (next_update_time);

CREATE INDEX subscriptions_op_idx
  ON subscriptions (op);

CREATE INDEX subscriptions_connector_event_idx
  ON subscriptions (connector, event);

CREATE TABLE c_time_subscriptions (
    id KSUID PRIMARY KEY REFERENCES subscriptions (id),
    last_tick TIMESTAMP,
    next_tick TIMESTAMP NOT NULL);

CREATE INDEX c_time_subscriptions_next_tick_idx
  ON c_time_subscriptions (next_tick);

CREATE TABLE c_github_subscriptions (
    id KSUID PRIMARY KEY REFERENCES subscriptions (id),
    organization VARCHAR NOT NULL,
    repository VARCHAR,
    hook_id INT8 NOT NULL);

CREATE INDEX c_github_subscriptions_organization_repository_idx
  ON c_github_subscriptions (organization, repository);

CREATE INDEX c_github_subscriptions_hook_id_idx
  ON c_github_subscriptions (hook_id);

CREATE INDEX events_event_time_idx
  ON events (event_time);

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
