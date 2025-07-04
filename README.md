# FLI - Flow Logs Insights

[![Go Report Card](https://goreportcard.com/badge/github.com/fractalops/fli)](https://goreportcard.com/report/github.com/fractalops/fli)
[![Go Version](https://img.shields.io/github/go-mod/go-version/fractalops/fli)](https://go.dev/)
[![Test Coverage](https://img.shields.io/badge/coverage-87.5%25-brightgreen)](https://github.com/fractalops/fli)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/fractalops/fli)](https://github.com/fractalops/fli/releases)
[![Contributors](https://img.shields.io/github/contributors/fractalops/fli)](https://github.com/fractalops/fli/graphs/contributors)

FLI is a command-line tool for querying and analyzing AWS VPC Flow Logs stored in CloudWatch Logs built in Go

## Features

- **Query AWS VPC Flow Logs** with natural language filters
- **Aggregate data** with count, sum, and group-by operations
- **Multiple output formats**: Table, CSV, and JSON
- **IP and ENI annotations**: WHOIS lookup, cloud provider IP detection, ENI tagging
- **Cross-platform**: Linux, macOS, and Windows binaries

## Quick Start

### Installation

**From Source**
```bash
git clone https://github.com/fractalops/fli.git
cd fli
make build
sudo make install
```

**Using Go**
```bash
go install github.com/fractalops/fli/cmd/fli@latest
```

### Configuration

1. **Set up AWS credentials**:
   ```bash
   aws configure
   ```

2. **Configure environment variables** (optional):
   ```bash
   # Set default log group
   export FLI_LOG_GROUP="/aws/vpc/flow-logs"
   ```

### Basic Usage

```bash
# Count all flows in the last 5 minutes
fli count --since 5m

# Show top 10 source IPs by traffic volume
fli sum bytes --by srcaddr --limit 10 --since 1h

# View raw flow logs with specific fields
fli raw srcaddr,dstaddr,dstport,action -s 30m

# Filter for specific traffic
fli raw -f "dstport=443 and action=ACCEPT" -s 1h

# Use different output format
fli count --by srcaddr -o json -s 1h

# Set version and timeout
fli raw -v 5 -t 30s -s 15m
```

## Common Commands

### Query Commands

```bash
# Raw data query
fli raw [fields] [flags]

# Count flows
fli count [fields] [flags]

# Sum numeric fields
fli sum <field> [flags]

# Average numeric fields
fli avg <field> [flags]

# Find minimum values
fli min <field> [flags]

# Find maximum values
fli max <field> [flags]
```

### Cache Commands

```bash
# Refresh ENI tags in the cache using AWS
fli cache refresh [--eni <eni-id>] [--all]

# List cached items
fli cache list

# Update cloud provider IP ranges
fli cache prefixes

# Delete the cache file
fli cache clean
```

## Common Flags

```bash
--log-group, -l    # CloudWatch Logs group to query
--since, -s        # Relative time range (e.g., 30m, 2h, 7d)
--filter, -f       # Filter expression
--by               # Group by fields (comma-separated)
--limit            # Limit number of results (default: 20)
--format, -o       # Output format: table, csv, json (default: table)
--version, -v      # Flow logs version: 2 or 5 (default: 2)
--timeout, -t      # Query timeout (e.g., 30s, 5m, 1h)
```

## Autocompletion

FLI provides intelligent autocompletion for commands, flags, fields, and filter expressions to enhance your productivity.

### Setup

#### Bash
```bash
# Generate bash completion script
fli completion bash > ~/.local/share/bash-completion/completions/fli

# Or add to your ~/.bashrc
echo 'source <(fli completion bash)' >> ~/.bashrc
source ~/.bashrc
```

#### Zsh
```bash
# Generate zsh completion script
fli completion zsh > ~/.zsh/completions/_fli

# Or add to your ~/.zshrc
echo 'source <(fli completion zsh)' >> ~/.zshrc
source ~/.zshrc
```

#### Fish
```bash
# Generate fish completion script
fli completion fish > ~/.config/fish/completions/fli.fish
```

#### PowerShell
```powershell
# Generate PowerShell completion script
fli completion powershell > fli.ps1

# Import the script
. .\fli.ps1
```

## Filtering Examples

```bash
# Filter by IP address
fli raw --filter "srcaddr=10.0.0.1"

# Filter by port
fli raw --filter "dstport=443"

# Filter by action
fli raw --filter "action=REJECT"

# Multiple conditions
fli raw -f "srcaddr=10.0.0.1 and dstport=443 and action=ACCEPT"

# CIDR blocks
fli count --by srcaddr -f "dstaddr=10.0.0.0/24"

# Port ranges
fli raw -f "dstport >= 80 and dstport <= 443"

# Protocol filtering
fli count --by dstport --filter "protocol=TCP"
```

## Output Formats

```bash
# Table format (default)
fli count --by srcaddr -o table

# CSV format
fli sum bytes --by srcaddr -o csv

# JSON format
fli raw --filter "action=REJECT" --format json
```

## Requirements

- **Go**: 1.20+ (for building from source)
- **AWS**: Account with VPC Flow Logs enabled
- **Permissions**: CloudWatch Logs read access
- **Platform**: Linux, macOS, or Windows

## AWS Permissions

Your IAM user/role needs the following permissions:
```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "logs:StartQuery",
                "logs:GetQueryResults",
                "logs:DescribeLogGroups"
            ],
            "Resource": "*"
        }
    ]
}
```

## Documentation

- [CLI Specification](docs/user/cli-specification.md)
- [Annotations System](docs/user/annotations.md)
- [Design Documents](docs/design/architecture.md)

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [AWS SDK for Go](https://github.com/aws/aws-sdk-go-v2) for AWS integration
- [Cobra](https://github.com/spf13/cobra) for the CLI framework
- [BBolt](https://github.com/etcd-io/bbolt) for embedded caching
- [WHOIS](https://github.com/likexian/whois) for IP address annotation
