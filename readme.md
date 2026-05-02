[![Build and Release](https://github.com/sal-consultancy/checky-check/actions/workflows/build.yml/badge.svg)](https://github.com/sal-consultancy/checky-check/actions/workflows/build.yml)

# CheckyCheck

CheckyCheck runs remote host checks over SSH, central URL checks from the controller, and renders the latest results in a small web UI.

Run history is stored in SQLite at `history/checkycheck_history.db`.

## Configuration

Configuration is YAML-only.

- `-config config.yaml` loads one YAML file
- `-config config/` loads and merges all `.yaml` and `.yml` files in that directory tree
- `CHECKYCHECK_CONFIG_PATH` can provide the default config path for binary or container deployments
- duplicate names across merged files are rejected

Typical top-level sections:

- `identities`
- `host_defaults`
- `host_templates`
- `check_groups`
- `checks`
- `url_checks`
- `host_groups`
- `report`

## Variables

Checks can use placeholders such as `${sap.sid}` or `${url}`.

- host variables can be defined in `host_defaults`, `host_templates`, `host_groups`, and individual hosts
- check groups can contribute default `vars`
- environment variables can be referenced with `${env.NAME}`
- nested variables are supported

## Check Groups

Checks can be grouped and reused from:

- `host_defaults`
- `host_templates`
- `host_groups`
- individual hosts

Example:

```yaml
check_groups:
  sap_os:
    vars:
      thresholds:
        disk_root: "80"
    checks:
      - check_disk_root_usage
      - check_failed_systemd_units
```

## Example Structure

```text
config/
  identities.yaml
  host_defaults.yaml
  report.yaml
  check_groups.yaml
  checks/
    base.yaml
    sap.yaml
  host_templates/
    sap.yaml
  host_groups/
    production.yaml
  url_checks.yaml
```

Multiple checks may live in the same file.

## Identities

### SSH key

```yaml
identities:
  ssh_key:
    user: username
    key: keys/id_rsa
```

### SSH key with passphrase

```yaml
identities:
  ssh_key:
    user: username
    key: keys/id_rsa
    passphrase: ${env.CC_PASSPHRASE}
```

### Username and password

```yaml
identities:
  ssh_password:
    user: username
    password: ${env.CC_PASSWORD}
```

## Host Checks

Host checks live under `checks:` and run over SSH unless `local: true` is set.

### Command check

```yaml
checks:
  check_uptime:
    title: Uptime
    command: uptime | awk '{print $3}'
    fail_when: ">"
    fail_value: "90"
```

### Service check

```yaml
checks:
  check_firewall_running:
    title: Firewall
    service: ufw
    fail_when: "="
    fail_value: "0"
```

### Multiple fail values

```yaml
checks:
  check_example:
    title: Example
    command: echo "${status}"
    fail_when: "!="
    fail_value: ["200", "302"]
```

## URL Checks

Central website checks live under `url_checks:` and always run from the CheckyCheck controller.

```yaml
url_checks:
  check_laddio_url:
    title: Laddio.nl
    url: ${url}
    fail_when: "!="
    fail_value: ["200", "302"]
    timeout: 15s
    follow_redirects: false
    expected_contains: Laddio
    vars:
      url: https://laddio.nl
```

URL checks store extra runtime details in `results.json`, including:

- HTTP status code
- latency in milliseconds
- redirect location or final URL
- technical error type such as DNS, TLS, timeout, or request failure

## Check Options

| Parameter | Description | Default |
|---|---|---|
| `title` | Display name of the check | |
| `description` | Extra explanation shown in the UI | |
| `timeout` | Per-check timeout | `30s` |
| `local` | Run a command locally on the controller | `false` |
| `vars` | Variables available to the check | |
| `graph` | Graph settings for host-based report charts | |
| `url` | Native HTTP/HTTPS target for a URL check | |
| `follow_redirects` | Follow redirects for a URL check | `false` |
| `expected_contains` | Require response body text for a URL check | |

## Running

Generate or refresh `results.json`:

```sh
checkycheck.exe -config=/path/to/config -mode=check
```

Serve the UI:

```sh
checkycheck.exe -port=8071 -config=/path/to/config -mode=serve
```

## History

CheckyCheck stores lightweight history:

- `runs`: one summary row per full run or targeted rerun
- `events`: failures, recoveries, config errors, and targeted rerun results

Retention defaults:

- runs: 90 days
- events: 30 days

## Deployment Note

For production deployments, keep customer configuration outside this repository.

- mount the active config directory into the runtime environment
- mount SSH keys and other secrets separately
- pass sensitive values such as passphrases via environment variables or a secret manager

## Development

Run tests:

```sh
go test ./...
```

Build the frontend:

```sh
cd frontend
npm run build
```

### Local auth proxy

To test `auth.mode: proxy` locally without Keycloak or oauth2-proxy, run the small header-injecting reverse proxy:

```sh
go run ./cmd/local-auth-proxy -upstream http://127.0.0.1:8070 -listen :8080
```

Then open:

```text
http://127.0.0.1:8080/__auth__
```

The chooser page lets you switch between:

- `unauthenticated`
- `viewer`
- `operator`
- `admin`

The proxy stores the selected role in a local cookie and injects the corresponding auth headers on proxied requests.
