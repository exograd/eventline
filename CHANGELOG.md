# Changelog
## v1.0.0
_Work in progress._

### Misc
- Add index to prevent potential performance issue during project deletion.

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

### Bug fixes
- Fix job pagination in evcli.
- Fix the Docker image so that evcli can be executed inside.

### Misc
- Use the default monospace font of the web browser instead of serving a half
  megabyte file.
- Disable color for logging if the error output stream is not a character
  device.
- Improve validation of the configuration file.

## v0.9.1
### Bug fixes
- Remove initial blank line(s) in notification emails.
- Remove invalid expiration date for `github/oauth2` identities.
- Fix the abort and restart buttons on job execution pages.

### Misc
- Use a stable tag in the docker-compose setup.

## v0.9.0
First public release.
