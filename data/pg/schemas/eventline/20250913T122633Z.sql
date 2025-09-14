CREATE OR REPLACE FUNCTION ksuid_to_uuid_v7(p_ksuid KSUID)
  RETURNS UUID
AS $$
DECLARE
  v_alphabet CONSTANT VARCHAR :=
    '0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz';
  v_epoch BIGINT = 1400000000;

  v_ksuid_i NUMERIC;
  v_ksuid BYTEA;
  v_ksuid_ts BIGINT;
  v_ts BIGINT;
  v_random BYTEA;
  v_uuid BYTEA;

  v_char CHARACTER;
  v_char_value INTEGER;
  v_i INTEGER;
BEGIN
  -- KSUID:
  --    32 bit second timestamp (since UNIX timestamp 1'400'000'000)
  --   128 bit random data
  -- UUID v7:
  --    48 bit millisecond timestamp
  --    74 bit random data
  --     6 bit version information
  --   (random data and version information fields are interleaved)
  --
  -- We keep the same timestamp with a zero millisecond part and truncate random
  -- data from 128 to 74 bit to preserve ordering as much as possible.

  -- Decode the KSUID (27 characters to 20 bytes)
  v_ksuid_i := 0;
  FOR v_i IN 1..27 LOOP
    v_char = substring(p_ksuid FROM v_i FOR 1);
    v_char_value := position(v_char IN v_alphabet) - 1;
    IF v_char_value < 0 THEN
        raise EXCEPTION 'invalid KSUID character %', v_char;
    END IF;
    v_ksuid_i := v_ksuid_i * 62 + v_char_value;
  END LOOP;

  v_ksuid = '\x0000000000000000000000000000000000000000'::BYTEA; -- 20 bytes
  FOR v_i IN REVERSE 19..0 LOOP
    v_ksuid := set_byte(v_ksuid, v_i, mod(v_ksuid_i, 256)::INTEGER);
    v_ksuid_i := div(v_ksuid_i, 256);
  END LOOP;

  -- Extract the timestamp and random data
  v_ksuid_ts := (get_byte(v_ksuid, 0)::BIGINT << 24) +
                (get_byte(v_ksuid, 1)::BIGINT << 16) +
                (get_byte(v_ksuid, 2)::BIGINT <<  8) +
                 get_byte(v_ksuid, 3);
  v_ts = (v_ksuid_ts + v_epoch) * 1000;

  v_random = substring(v_ksuid FROM 5 FOR 16);

  -- Initialize the UUID timestamp
  v_uuid := '\x00000000000000000000000000000000'::BYTEA; -- 16 bytes
  v_uuid := set_byte(v_uuid, 0, ((v_ts & 0xff0000000000) >> 40)::INTEGER);
  v_uuid := set_byte(v_uuid, 1, ((v_ts & 0x00ff00000000) >> 32)::INTEGER);
  v_uuid := set_byte(v_uuid, 2, ((v_ts & 0x0000ff000000) >> 24)::INTEGER);
  v_uuid := set_byte(v_uuid, 3, ((v_ts & 0x000000ff0000) >> 16)::INTEGER);
  v_uuid := set_byte(v_uuid, 4, ((v_ts & 0x00000000ff00) >>  8)::INTEGER);
  v_uuid := set_byte(v_uuid, 5,  (v_ts & 0x0000000000ff)::INTEGER);

  -- Set the version
  v_uuid :=
    set_byte(v_uuid, 6, ((get_byte(v_random, 2) & 0x0f) | 0x70));

  -- Copy random data (
  FOR v_i IN 0..7 LOOP
    v_uuid :=
      set_byte(v_uuid, v_i + 7, get_byte(v_random, v_i + 3));
  END LOOP;

  -- Set the variant bit in the middle of random data
  v_uuid :=
    set_byte(v_uuid, 8, ((get_byte(v_random, 8) & 0x3f) | 0x80));

  RETURN encode(v_uuid, 'hex')::UUID;
END
$$ LANGUAGE PLPGSQL IMMUTABLE STRICT;



ALTER TABLE accounts
  DROP CONSTRAINT accounts_last_project_id_fkey;

ALTER TABLE api_keys
  DROP CONSTRAINT api_keys_account_id_fkey;

ALTER TABLE c_github_subscriptions
  DROP CONSTRAINT c_github_subscriptions_id_fkey;

ALTER TABLE c_time_subscriptions
  DROP CONSTRAINT c_time_subscriptions_id_fkey;

ALTER TABLE events
  DROP CONSTRAINT events_job_id_fkey,
  DROP CONSTRAINT events_original_event_id_fkey,
  DROP CONSTRAINT events_project_id_fkey;

ALTER TABLE favourite_jobs
  DROP CONSTRAINT favourite_jobs_account_id_fkey,
  DROP CONSTRAINT favourite_jobs_job_id_fkey,
  DROP CONSTRAINT favourite_jobs_project_id_fkey;

ALTER TABLE identities
  DROP CONSTRAINT identities_project_id_fkey;

ALTER TABLE job_executions
  DROP CONSTRAINT job_executions_event_id_fkey,
  DROP CONSTRAINT job_executions_job_id_fkey,
  DROP CONSTRAINT job_executions_project_id_fkey;

ALTER TABLE jobs
  DROP CONSTRAINT jobs_project_id_fkey;

ALTER TABLE notifications
  DROP CONSTRAINT notifications_project_id_fkey;

ALTER TABLE project_notification_settings
  DROP CONSTRAINT project_notification_settings_id_fkey;

ALTER TABLE project_settings
  DROP CONSTRAINT project_settings_id_fkey;

ALTER TABLE sessions
  DROP CONSTRAINT sessions_account_id_fkey;

ALTER TABLE step_executions
  DROP CONSTRAINT step_executions_job_execution_id_fkey,
  DROP CONSTRAINT step_executions_project_id_fkey;

ALTER TABLE subscriptions
  DROP CONSTRAINT subscriptions_identity_id_fkey,
  DROP CONSTRAINT subscriptions_job_id_fkey,
  DROP CONSTRAINT subscriptions_project_id_fkey;



ALTER TABLE accounts
  ALTER COLUMN id TYPE UUID
    USING ksuid_to_uuid_v7(id),
  ALTER COLUMN last_project_id TYPE UUID
    USING ksuid_to_uuid_v7(last_project_id);

ALTER TABLE api_keys
  ALTER COLUMN id TYPE UUID
    USING ksuid_to_uuid_v7(id),
  ALTER COLUMN account_id TYPE UUID
    USING ksuid_to_uuid_v7(account_id);

ALTER TABLE c_github_subscriptions
  ALTER COLUMN id TYPE UUID
    USING ksuid_to_uuid_v7(id);

ALTER TABLE c_time_subscriptions
  ALTER COLUMN id TYPE UUID
    USING ksuid_to_uuid_v7(id);

ALTER TABLE events
  ALTER COLUMN id TYPE UUID
    USING ksuid_to_uuid_v7(id),
  ALTER COLUMN project_id TYPE UUID
    USING ksuid_to_uuid_v7(project_id),
  ALTER COLUMN job_id TYPE UUID
    USING ksuid_to_uuid_v7(job_id),
  ALTER COLUMN original_event_id TYPE UUID
    USING ksuid_to_uuid_v7(original_event_id);

ALTER TABLE favourite_jobs
  ALTER COLUMN account_id TYPE UUID
    USING ksuid_to_uuid_v7(account_id),
  ALTER COLUMN project_id TYPE UUID
    USING ksuid_to_uuid_v7(project_id),
  ALTER COLUMN job_id TYPE UUID
    USING ksuid_to_uuid_v7(job_id);

ALTER TABLE identities
  ALTER COLUMN id TYPE UUID
    USING ksuid_to_uuid_v7(id),
  ALTER COLUMN project_id TYPE UUID
    USING ksuid_to_uuid_v7(project_id);

ALTER TABLE job_executions
  ALTER COLUMN id TYPE UUID
    USING ksuid_to_uuid_v7(id),
  ALTER COLUMN project_id TYPE UUID
    USING ksuid_to_uuid_v7(project_id),
  ALTER COLUMN job_id TYPE UUID
    USING ksuid_to_uuid_v7(job_id),
  ALTER COLUMN event_id TYPE UUID
    USING ksuid_to_uuid_v7(event_id);

ALTER TABLE jobs
  ALTER COLUMN id TYPE UUID
    USING ksuid_to_uuid_v7(id),
  ALTER COLUMN project_id TYPE UUID
    USING ksuid_to_uuid_v7(project_id);

ALTER TABLE notifications
  ALTER COLUMN id TYPE UUID
    USING ksuid_to_uuid_v7(id),
  ALTER COLUMN project_id TYPE UUID
    USING ksuid_to_uuid_v7(project_id);

ALTER TABLE project_notification_settings
  ALTER COLUMN id TYPE UUID
    USING ksuid_to_uuid_v7(id);

ALTER TABLE project_settings
  ALTER COLUMN id TYPE UUID
    USING ksuid_to_uuid_v7(id);

ALTER TABLE projects
  ALTER COLUMN id TYPE UUID
    USING ksuid_to_uuid_v7(id);

ALTER TABLE sessions
  ALTER COLUMN id TYPE UUID
    USING ksuid_to_uuid_v7(id),
  ALTER COLUMN account_id TYPE UUID
    USING ksuid_to_uuid_v7(account_id);

ALTER TABLE step_executions
  ALTER COLUMN id TYPE UUID
    USING ksuid_to_uuid_v7(id),
  ALTER COLUMN project_id TYPE UUID
    USING ksuid_to_uuid_v7(project_id),
  ALTER COLUMN job_execution_id TYPE UUID
    USING ksuid_to_uuid_v7(job_execution_id);

ALTER TABLE subscriptions
  ALTER COLUMN id TYPE UUID
    USING ksuid_to_uuid_v7(id),
  ALTER COLUMN project_id TYPE UUID
    USING ksuid_to_uuid_v7(project_id),
  ALTER COLUMN job_id TYPE UUID
    USING ksuid_to_uuid_v7(job_id),
  ALTER COLUMN identity_id TYPE UUID
    USING ksuid_to_uuid_v7(identity_id);



ALTER TABLE accounts
  ADD CONSTRAINT accounts_last_project_id_fkey
    FOREIGN KEY (last_project_id) REFERENCES projects (id);

ALTER TABLE api_keys
  ADD CONSTRAINT api_keys_account_id_fkey
    FOREIGN KEY (account_id) REFERENCES accounts (id) ON DELETE CASCADE;

ALTER TABLE c_github_subscriptions
  ADD CONSTRAINT c_github_subscriptions_id_fkey
    FOREIGN KEY (id) REFERENCES subscriptions (id);

ALTER TABLE c_time_subscriptions
  ADD CONSTRAINT c_time_subscriptions_id_fkey
    FOREIGN KEY (id) REFERENCES subscriptions (id);

ALTER TABLE events
  ADD CONSTRAINT events_job_id_fkey
    FOREIGN KEY (job_id) REFERENCES jobs (id) ON DELETE CASCADE,
  ADD CONSTRAINT events_original_event_id_fkey
    FOREIGN KEY (original_event_id) REFERENCES events (id) ON DELETE SET NULL,
  ADD CONSTRAINT events_project_id_fkey
    FOREIGN KEY (project_id) REFERENCES projects (id) ON DELETE CASCADE;

ALTER TABLE favourite_jobs
  ADD CONSTRAINT favourite_jobs_account_id_fkey
    FOREIGN KEY (account_id) REFERENCES accounts (id) ON DELETE CASCADE,
  ADD CONSTRAINT favourite_jobs_job_id_fkey
    FOREIGN KEY (job_id) REFERENCES jobs (id) ON DELETE CASCADE,
  ADD CONSTRAINT favourite_jobs_project_id_fkey
    FOREIGN KEY (project_id) REFERENCES projects (id) ON DELETE CASCADE;

ALTER TABLE identities
  ADD CONSTRAINT identities_project_id_fkey
    FOREIGN KEY (project_id) REFERENCES projects (id);

ALTER TABLE job_executions
  ADD CONSTRAINT job_executions_event_id_fkey
    FOREIGN KEY (event_id) REFERENCES events (id),
  ADD CONSTRAINT job_executions_job_id_fkey
    FOREIGN KEY (job_id) REFERENCES jobs (id) ON DELETE CASCADE,
  ADD CONSTRAINT job_executions_project_id_fkey
    FOREIGN KEY (project_id) REFERENCES projects (id) ON DELETE CASCADE;

ALTER TABLE jobs
  ADD CONSTRAINT jobs_project_id_fkey
    FOREIGN KEY (project_id) REFERENCES projects (id);

ALTER TABLE notifications
  ADD CONSTRAINT notifications_project_id_fkey
    FOREIGN KEY (project_id) REFERENCES projects (id) ON DELETE CASCADE;

ALTER TABLE project_notification_settings
  ADD CONSTRAINT project_notification_settings_id_fkey
    FOREIGN KEY (id) REFERENCES projects (id) ON DELETE CASCADE;

ALTER TABLE project_settings
  ADD CONSTRAINT project_settings_id_fkey
    FOREIGN KEY (id) REFERENCES projects (id) ON DELETE CASCADE;

ALTER TABLE sessions
  ADD CONSTRAINT sessions_account_id_fkey
    FOREIGN KEY (account_id) REFERENCES accounts (id) ON DELETE CASCADE;

ALTER TABLE step_executions
  ADD CONSTRAINT step_executions_job_execution_id_fkey
    FOREIGN KEY (job_execution_id)
      REFERENCES job_executions (id) ON DELETE CASCADE,
  ADD CONSTRAINT step_executions_project_id_fkey
    FOREIGN KEY (project_id) REFERENCES projects (id) ON DELETE CASCADE;

ALTER TABLE subscriptions
  ADD CONSTRAINT subscriptions_identity_id_fkey
    FOREIGN KEY (identity_id) REFERENCES identities (id),
  ADD CONSTRAINT subscriptions_job_id_fkey
    FOREIGN KEY (job_id) REFERENCES jobs (id),
  ADD CONSTRAINT subscriptions_project_id_fkey
    FOREIGN KEY (project_id) REFERENCES projects (id);



DROP FUNCTION ksuid_to_uuid_v7;
DROP FUNCTION generate_ksuid;
DROP DOMAIN KSUID;
