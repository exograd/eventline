# Changelog
## v1.0.0
_Work in progress._

### Features
- Add pagination support for the `list-projects` evcli command.
- Add a `--wait` option to the `execute-job` evcli command which monitors
  execution, print status changes and wait for execution to finish before
  exiting.
- Add a `--fail` option to the `execute-job` evcli command to exit with status
  1 if job execution fails.
- Render [ANSI escape
  sequences](https://en.wikipedia.org/wiki/ANSI_escape_code) in execution
  output data on the web interface.
- `github/oauth2` identities can now be used as identity for the `docker`
  runner.
- Add support for deletion of old job executions based on the `job_retention`
  setting and the `retention` job field.
- Add support for deletion of old sessions based on the `session_retention`
  setting.
- Inject job parameters as files in `$EVENTLINE_DIR/parameters`.
- Add a notification setting to allow a specific list of email address
  domains.
- Add the `job-execution-watcher` worker to detect and stop jobs which have
  timed out, i.e. executions which have not been refreshed for some time. Job
  executions are now refreshed regularly (the interval is controlled by the
  `job_execution_refresh_interval` setting). The timeout duration is
  controlled by the `job_execution_timeout` setting.
- Add a `--validate-cfg` command line flag to exit after configuration
  validation but before starting the service.
- Add an `allowed_runners` setting to provide a list of the runners allowed in
  submitted jobs.

### Bug fixes
- Fix job pagination in evcli.
- Fix the Docker image so that evcli can be executed inside.
- Always provide `EVENTLINE_DIR` as an absolute path.
- Fix incorrect validation of the `tls` field in http server configuration.
- Fix immediate session deletion issue when session retention is not
  configured or equal to zero.
- Fix incorrect validation of the influxdb client configuration.
- Fix initialization so that the program exits when HTTP server initialization
  fails.
- Fix configuration validation so that it fails when there is no encryption
  key.
- Fix file permissions when executing in a container as a non-root user.
- Add missing dependencies to the Systemd unit file.

### Misc
- Use the default monospace font of the web browser instead of serving a half
  megabyte file.
- Disable color for logging if the error output stream is not a character
  device.
- Improve validation of the configuration file.
- Add index to prevent potential performance issue during project deletion.
- Rename settings:
  - `max_parallel_jobs` to `max_parallel_job_executions`.
  - `job_retention` to `job_execution_retention`.

## v0.9.1
### Bug fixes
- Remove initial blank line(s) in notification emails.
- Remove invalid expiration date for `github/oauth2` identities.
- Fix the abort and restart buttons on job execution pages.

### Misc
- Use a stable tag in the docker-compose setup.

## v0.9.0
First public release.
