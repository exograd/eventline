# Eventline
[Eventline](https://eventline.net) is an open source job scheduling platform
developed by [Exograd](http://exograd.com).

Eventline makes it easy to control all your automation in the same place.
Small recurrent tasks, long processing jobs, integration scripts, everything
runs in Eventline.

## Runners
Eventline lets you execute jobs the way you want:

| Runner       | Description                        | Availability  |
|--------------|------------------------------------|---------------|
| `local`      | Local execution.                   | Eventline     |
| `docker`     | Execution in a Docker container.   | Eventline     |
| `ssh`        | Remote execution.                  | Eventline     |
| `kubernetes` | Execution in a Kubernetes cluster. | Eventline Pro |

## Connectors
Connectors include support for various identities, which are used to store
credentials, and for events used to trigger jobs.

Eventline supports multiple connectors, and we intend to add a lot more.

| Connector    | Description                   | Availability  |
|--------------|-------------------------------|---------------|
| `eventline`  | Eventline identities.         | Eventline     |
| `generic`    | Various generic identities.   | Eventline     |
| `time`       | Recurring events.             | Eventline     |
| `dockerhub`  | DockerHub identities.         | Eventline     |
| `postgresql` | PostgreSQL identities.        | Eventline     |
| `github`     | GitHub identities and events. | Eventline     |
| `slack`      | Slack identities.             | Eventline Pro |

## Example
Eventline makes it trivial to write various kinds of jobs. For example:

```yaml
---
name: "export-clients"
trigger:
  event: "time/tick"
  parameters:
    daily:
      hour: 23
identities:
  - "pg-export"
  - "aws-s3"
environment:
  PG_HOST: "pg.example.com"
  S3_URI: "https://s3.eu-west-3.amazonaws.com/clients-data"
steps:
  - label: "export the database"
    code: "pg_dump -h $PG_HOST clients > clients.pgdump"
  - label: "upload data to s3"
    code: |
      key=$(date -u +%FT%TZ).pgdump
      aws s3 cp clients.pgdump $S3_URI/$key
```

Once defined, simply deploy it using the evcli command line program:

```
evcli deploy-job export-clients.yaml
```

## Running Eventline
### Docker
The simplest way to run Eventline is to use the [Docker
Compose](misc/docker-compose.yaml) setup, which pulls the official
`exograd/eventline` Docker image.

Refer to the documentation for a [quick start
guide](https://www.exograd.com/doc/eventline/handbook.html#_getting_started).

### FreeBSD
Eventline is packaged for FreeBSD.

```sh
mkdir -p /usr/local/etc/pkg/repos
curl -sSfL -o /usr/local/etc/pkg/repos/exograd-public.conf \
     https://pkg.exograd.com/public/freebsd/exograd.conf
pkg update
pkg install eventline
```

Refer to the documentation for [more
information](https://www.exograd.com/doc/eventline/handbook.html#freebsd-package).

### Ubuntu
Eventline is packaged for Ubuntu:

```sh
curl -sSfL -o /etc/apt/sources.list.d/exograd-public.list \
     https://pkg.exograd.com/public/ubuntu/exograd.list
apt-get update
apt-get install eventline
```

Refer to the documentation for [more
information](https://www.exograd.com/doc/eventline/handbook.html#ubuntu-package).

## Documentation
The Eventline handbook is available [in the
repository](doc/handbook/handbook.adoc), on the [Exograd
website](https://www.exograd.com/doc/eventline/handbook.html) and in each
GitHub release.

## Commercial use
We also provide Eventline Pro with multiple extensions and commercial support.

Exograd is a small bootstrapped company; by using Eventline Pro, you help us
secure the future of the open source version.

[Contact us](mailto:contact@exograd.com) at any time for questions, we would
love to help you!

## Contact
Feel free to open a GitHub issue if you find a bug. You can also use
GitHub Discussions for questions, ideas or suggestions.

Eventline Pro users also get access to private support by email.
