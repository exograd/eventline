CREATE INDEX jobs_spec_runner_identity_idx
  ON jobs ((spec->'runner'->>'identity'))
  WHERE spec->'runner' IS NOT NULL;
