# Query Builder Package

The querybuilder package provides a modern, type-safe builder for constructing CloudWatch Logs Insights queries for AWS VPC Flow Logs. It supports all major VPC Flow Log versions and provides compile-time safety for fields, filters, and aggregation verbs.

## File Structure

- `builder_core.go` - Core query building functionality
- `builder_options.go` - Options for configuring the builder
- `expressions.go` - Expression types and interfaces
- `filter_lexer.go` - Tokenization and field type handling
- `filter_parser.go` - Parsing filter expressions
- `schema.go` - Schema interface definition
- `vpc_flow_logs_schema.go` - VPC Flow Logs specific implementation
- `verbs.go` - Query verb definitions
- `verb_string.go` - Auto-generated String() method for Verb type

## Key Components

### Builder

The `Builder` is the main entry point for constructing queries. It's created with the `New` function and configured with options:

```go
builder, err := querybuilder.New(
    schema,
    querybuilder.WithVerb(querybuilder.VerbCount),
    querybuilder.WithFields("srcaddr"),
    querybuilder.WithFilter(querybuilder.Eq{Field: "dstport", Value: 443}),
    querybuilder.WithLimit(10),
)
```

### Expressions

The package provides a rich set of expression types for building filters:

- `Eq` - Equality comparison
- `Neq` - Non-equality comparison
- `Gt` - Greater than comparison
- `Lt` - Less than comparison
- `Gte` - Greater than or equal comparison
- `Lte` - Less than or equal comparison
- `Like` - Pattern matching
- `NotLike` - Negative pattern matching
- `And` - Conjunction of expressions
- `Or` - Disjunction of expressions
- `NotExpr` - Logical NOT operation
- `IsIpv4InSubnet` - CIDR block membership check

### Schema

The `Schema` interface defines the contract for a specific data source's query dialect. The package includes a `VPCFlowLogsSchema` implementation for VPC Flow Logs.

### Filter Parser

The filter parser converts string filter expressions into expression trees:

```go
expr, err := querybuilder.ParseFilter("srcaddr = '10.0.0.1' and dstport = 443")
```

## Usage Examples

See the [examples_test.go](examples_test.go) file for comprehensive usage examples.
