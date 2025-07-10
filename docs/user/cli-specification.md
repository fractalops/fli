# CLI Specification

This document defines the **contract** between the CLI parser and the query-builder:
1. **Grammar / syntax** of every CLI invocation
2. **How each verb is rewritten** into the core CloudWatch Logs Insights primitives
3. **Rules the builder follows** (parse clause, filters, grouping, limits, etc.)

---

## 1  Grammar (EBNF-style)

```ebnf
command        = "fli" , verb , target , options ;

verb           = "count" | "sum" | "avg" | "min" | "max" | "raw" ;

target         = identifier                // e.g. dstaddr,srcaddr, bytes
               | field-name                // any flow-log field or computed alias
               ;

options        = { option } ;

option         = "by" , field-name
               | "--filter" , quote , filter-expr , quote
               | "--log-group" , name
               | "--since" , duration
               | "--limit" , integer
               | "--dry-run"
               | "--format" , ("table" | "json" | "csv")
               | "--version", integer
               | "--debug"
               | "--color"
               | "--no-ptr"
               | "--proto-names"
               | "--save-enis"
               | "--save-ips"
               | "--timeout" , duration

               ;

filter-expr    = <builder's mini-DSL, e.g. srcport=443 and action="REJECT">
field-name     = letter , { letter | digit | "-" | "_" } ;
identifier     = same as field-name ;
```

### Parsing rules

* `--by` supersedes the noun if the noun is not itself a field.
* `--filter` is inserted *after* the parse clause and *before* the stats line.
* `--limit` always goes last, after any `sort`.

---

## 2  Verb-to-Insights mapping

### 2.1 count

| Pattern                     | Generated **stats** line             | Sort default              |
| --------------------------- | ------------------------------------ | ------------------------- |
| `count flows`               | `stats count(*) as flows`            | none                      |
| `count <field>`             | `stats count(*) as flows by <field>` | `flows desc`              |

*(Everything else is syntactic sugar on these three forms.)*

---

### 2.2 sum | avg | min | max

For any verb **v ∈ {sum, avg, min, max}** and field **f**:

```
stats v(f) as v_f
```

*If `--by x` is present:*

```
stats v(f) as v_f by x
sort v_f desc
```

---

## 3  Automatic builder logic

1. **Parse clause**

```insights
parse @message "* * * * * * * * * * * * * *" as version, account_id, interface_id, srcaddr, dstaddr, srcport, dstport, protocol, packets, bytes, start, end, action, log_status
```

   *Fields referenced in the target / filter / group list are **added** if missing.*

2. **Filter insertion**
   All `--filter` expressions are joined with **AND** and injected after the parse.

3. **Stats + sort + limit**
   *See tables above.*
   If the user sets `--limit` it replaces the default

4. **Dry-run**
   *Stop after rendering the full query string.*

5. **Live run**
   *Call the Runner; poll until `status=Complete`; format with the chosen formatter.*

---

### Example end-to-end (expanded)

Command

```bash
fli count dstport --filter 'interface_id="eni-01abc"' \
                  --since 1h --limit 5 --dry-run
```

Resolves to

```insights
<parse expression>
| filter interface_id = "eni-01abc"
| stats count(*) as flows by dstport
| sort flows desc
| limit 5
```

---

## 4  Flag Reference

### Common Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--log-group`, `-l` | string | - | CloudWatch Logs group name |
| `--since` | duration | 5m | Time window to look back |
| `--limit` | int | 20 | Maximum number of results |
| `--format` | string | table | Output format (table, csv, json) |
| `--filter` | string | - | Filter expression |
| `--by` | string | - | Group by field(s) |
| `--dry-run` | bool | false | Show query without executing |
| `--debug` | bool | false | Enable debug output |
| `--color` | bool | true | Colorize output |
| `--no-ptr` | bool | true | Remove @ptr fields |
| `--proto-names` | bool | true | Use protocol names |
| `--version` | int | 2 | VPC Flow Logs version |
| `--timeout` | duration | 5m | Query timeout |

### Cache Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--save-enis` | bool | false | Save ENIs to cache |
| `--save-ips` | bool | false | Save public IPs to cache |
| `--cache` | string | ~/.fli/cache/anno.db | Path to cache file |
| `--verbose` | bool | false | Enable verbose output |
| `--eni` | []string | - | ENI IDs to refresh (for refresh command) |
| `--all` | bool | false | Refresh all ENIs (for refresh command) |



---

## 5  Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `FLI_LOG_GROUP` | Default log group | - |

---

## 6  Cache Commands

FLI provides several cache-related commands:

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
