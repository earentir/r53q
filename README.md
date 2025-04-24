# r53q: Tiny AWS Route53 CLI

A lightweight command-line tool to interact with AWS Route53. Written in Go, it supports listing hosted zones, listing DNS records, querying individual zone IDs or names, and displaying record counts. Configuration (credentials & region) is automatically loaded from a JSON file, environment variables, or generated if missing.

GPL v2 Licensed.

## Features
- **List hosted zones**      : `r53q list zones`
- **List DNS records**       : `r53q list records <zone-id|domain>`
- **Query a zone**           : `r53q zone <zone-id|domain>`
- **Get record count**       : `r53q zone <zone-id|domain> count`
- **Version info**           : `r53q --version` (also prints config source)

## Installation

1. **Clone the repository**

   ```bash
   git clone https://github.com/youruser/r53q.git
   cd r53q
   ```

2. **Ensure Go is installed** (>=1.16)

3. **Make the build script executable**

   ```bash
   chmod +x build.sh
   ```

4. **Build the binary**

   ```bash
   ./build.sh 1.0.0
   ```

   This produces `r53q` in the project root, embedding version, commit hash, and build date.

## Usage

```bash
# Show version and config source
./r53q --version
# Example output:
# r53q 1.0.0 (commit ab12cd3, built 2025-04-24T12:34:56Z)
# Config: /home/alice/.config/r53q.json

# List hosted zones
./r53q list zones

# List records in a zone (by ID or domain)
./r53q list records ear.pm
./r53q list records Z123ABCDEF

# Query a zone (ID or name)
./r53q zone ear.pm       # prints the zone ID
./r53q zone Z123ABCDEF   # prints the zone name

# Get record count
./r53q zone ear.pm count
./r53q zone Z123ABCDEF count
```

## Configuration

r53q will look for credentials and region in this order:

1. **Configuration file** `r53q.json` located:
   - Next to the executable
   - `$HOME/.config/r53q.json`
   - `/etc/r53q.json`

2. **Environment variables** (AWS CLI standard):
   - `AWS_ACCESS_KEY_ID`
   - `AWS_SECRET_ACCESS_KEY`
   - `AWS_REGION` or `AWS_DEFAULT_REGION`

3. **Generate empty config** if neither file nor env-vars exist:
   - Creates `r53q.json` in the current directory with empty values.
   - Prompts user to populate the file before running other commands.

## Build Script (`build.sh`)

```bash
#!/usr/bin/env bash
set -euo pipefail

# Usage: ./build.sh <version>
APP_VERSION=$1
GIT_COMMIT=$(git rev-parse --short HEAD)
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

go build -ldflags "\
  -X 'main.appVersion=${APP_VERSION}' \
  -X 'main.gitCommit=${GIT_COMMIT}' \
  -X 'main.buildDate=${BUILD_DATE}'" \
  -o r53q main.go
```

## Contributing

Contributions are always welcome!
All contributions are required to follow the https://google.github.io/styleguide/go/


## Authors

- [@earentir](https://www.github.com/earentir)


## License

I will always follow the Linux Kernel License as primary, if you require any other OPEN license please let me know and I will try to accomodate it.

[![License](https://img.shields.io/github/license/earentir/gitearelease)](https://opensource.org/license/gpl-2-0)
