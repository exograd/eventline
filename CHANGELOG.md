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

### Bug fixes
- Remove initial blank line(s) in notification emails.
- Remove invalid expiration date for `github/oauth2` identities.
- Fix job pagination in evcli.

## v0.9.0
First public release.
