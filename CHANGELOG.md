# Changelog
## v1.1.0
_Work in progress._

### Breaking changes
- In the logger configuration, the `backend` member is replaced by either
  `terminal_backend` or `json_backend` depending on backend type configured.
  For example:
  ```yaml
  logger:
    backend_type: "terminal"
    backend:
      color: true
  ```
  Is now:
  ```yaml
  logger:
    backend_type: "terminal"
    terminal_backend:
      color: true
  ```

### Features
- Add the possibility to rename jobs. Since jobs are primarily identified by
  their name, deploying a job with a different name creates a new job instead
  of renaming the existing one. This new feature makes it possible to rename a
  job, for example to archive it.
- Add a `/jobs/id/:id/rename` API route.
- Add a `rename-job` evcli command.

## v1.0.8
### Bug fixes
- Fix login for users who have no current project when the "main" project has
  been deleted.
- Fix invalid docker `ContainerWait` parameter for recent Docker versions.

## v1.0.7
### Misc
- Truncate the output of all step executions on the web interface. The current
  limit is hardcoded to 1MB. Very large outputs cause performance issues both
  on the server and in the web browser.
- Add buffering to runner output capture to avoid overloading the database for
  jobs producing massive amounts of data. See the associated commit for more
  information.

## v1.0.6
### Bug fixes
- Fix validation for the subscription parameters of the `time` connector.

## v1.0.5
### Bug fixes
- Fix repository URI for evcli update checks.

## v1.0.4
### Bug fixes
- Fix subscription handling during the update of a job whose trigger is being
  removed.
- Fix event processing when the job has been modified and does not have a
  trigger anymore.

## v1.0.3
### Bug fixes
- Fix the leak of the client stream socket in the Docker runner.

## v1.0.2
### Bug fixes
- Fix terminal rendering so that text using black foreground remains visible.

## v1.0.1
Thanks to Adyxax for his help!

### Bug fixes
- Fix missing argument in error message in evcli.
- Recreate the subscription if the identity has changed during a job update.
- Ensure terminal restoration on login error in evcli.
- Fix ssh runner termination when the connection was never established.
- Fix support for ssh jobs with multiple users on the same host.

## v1.0.0
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
- Add a `generic/gpg_key` identity to store GPG keys.
- Add a new `ssh` runner to execute remote jobs.
- Add support for mount points to the `docker` runner.

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
- Fix filter serialization.
- Update the last use time of all identities injected in the job during
  execution.
- Fix decoding of validation errors with no data in evcli.
- Interrupt execution on job abortion.

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
- Do not create events for periodic timer ticks which occurred when the server
  was down.
- The job id is now mandatory in event objects.
- Store the file containing the last build id check date in `$HOME/.evcli` for
  consistency.
- Set `HOME` for the local runner.

## v0.9.1
### Bug fixes
- Remove initial blank line(s) in notification emails.
- Remove invalid expiration date for `github/oauth2` identities.
- Fix the abort and restart buttons on job execution pages.

### Misc
- Use a stable tag in the docker-compose setup.

## v0.9.0
First public release.
