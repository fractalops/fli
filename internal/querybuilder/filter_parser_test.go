package querybuilder

import (
	"strings"
	"testing"
)

func TestParseFilter(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Expr
		wantErr bool
	}{
		{
			name:  "simple equals",
			input: "srcaddr = '10.0.0.1'",
			want:  &Eq{Field: "srcaddr", Value: "10.0.0.1"},
		},
		{
			name:  "simple not equals",
			input: "srcaddr != '10.0.0.1'",
			want:  &Neq{Field: "srcaddr", Value: "10.0.0.1"},
		},
		{
			name:  "simple like",
			input: "srcaddr like '10.0'",
			want:  &Like{Field: "srcaddr", Value: "10.0"},
		},
		{
			name:  "simple not like",
			input: "srcaddr not like '10.0'",
			want:  &NotLike{Field: "srcaddr", Value: "10.0"},
		},
		{
			name:  "and expression",
			input: "srcaddr = '10.0.0.1' and dstport = '443'",
			want:  &And{&Eq{Field: "srcaddr", Value: "10.0.0.1"}, &Eq{Field: "dstport", Value: 443}},
		},
		{
			name:  "or expression",
			input: "dstport = '80' or dstport = '443'",
			want:  &Or{&Eq{Field: "dstport", Value: 80}, &Eq{Field: "dstport", Value: 443}},
		},
		{
			name:  "complex expression",
			input: "srcaddr like '10.0' and (dstport = '80' or dstport = '443')",
			want:  &And{&Like{Field: "srcaddr", Value: "10.0"}, &Or{&Eq{Field: "dstport", Value: 80}, &Eq{Field: "dstport", Value: 443}}},
		},
		{
			name:  "ip subnet",
			input: "srcaddr = '10.0.0.0/24'",
			want:  &IsIpv4InSubnet{Field: "srcaddr", Value: "10.0.0.0/24"},
		},
		{
			name:  "ip subnet not equals",
			input: "srcaddr != '10.0.0.0/24'",
			want:  &NotExpr{Expr: &IsIpv4InSubnet{Field: "srcaddr", Value: "10.0.0.0/24"}},
		},
		{
			name:    "invalid expression",
			input:   "srcaddr 10.0.0.1",
			wantErr: true,
		},
		{
			name:    "unsupported operator",
			input:   "action > 'ACCEPT'",
			wantErr: true,
		},
		{
			name:  "valid port",
			input: "dstport = '443'",
			want:  &Eq{Field: "dstport", Value: 443},
		},
		{
			name:    "port out of range",
			input:   "srcport = 70000",
			wantErr: true,
		},
		{
			name:    "invalid port value",
			input:   "dstport = 'https'",
			wantErr: true,
		},
		{
			name:  "bytes greater than",
			input: "bytes > 1000",
			want:  &Gt{Field: "bytes", Value: 1000},
		},
		{
			name:  "bytes less than",
			input: "bytes < 5000",
			want:  &Lt{Field: "bytes", Value: 5000},
		},
		{
			name:  "bytes greater than or equal",
			input: "bytes >= 1000",
			want:  &Gte{Field: "bytes", Value: 1000},
		},
		{
			name:  "bytes less than or equal",
			input: "bytes <= 5000",
			want:  &Lte{Field: "bytes", Value: 5000},
		},
		{
			name:  "packets greater than",
			input: "packets > 10",
			want:  &Gt{Field: "packets", Value: 10},
		},
		{
			name:  "duration greater than",
			input: "duration > 300",
			want:  &Gt{Field: "duration", Value: 300},
		},
		{
			name:  "protocol greater than",
			input: "protocol > 6",
			want:  &Gt{Field: "protocol", Value: 6},
		},
		{
			name:  "port greater than",
			input: "srcport > 1024",
			want:  &Gt{Field: "srcport", Value: 1024},
		},
		{
			name:  "port less than",
			input: "dstport < 1024",
			want:  &Lt{Field: "dstport", Value: 1024},
		},
		{
			name:  "port greater than or equal",
			input: "srcport >= 80",
			want:  &Gte{Field: "srcport", Value: 80},
		},
		{
			name:  "port less than or equal",
			input: "dstport <= 443",
			want:  &Lte{Field: "dstport", Value: 443},
		},
		{
			name:  "duration greater than with schema",
			input: "duration > 1000",
			want:  &Gt{Field: "end - start", Value: 1000},
		},
		{
			name:  "duration less than with schema",
			input: "duration < 300",
			want:  &Lt{Field: "end - start", Value: 300},
		},
		{
			name:  "duration greater than or equal with schema",
			input: "duration >= 500",
			want:  &Gte{Field: "end - start", Value: 500},
		},
		{
			name:  "duration equals with schema",
			input: "duration = 1000",
			want:  &Eq{Field: "end - start", Value: 1000},
		},
		{
			name:  "protocol equals numeric",
			input: "protocol = 6",
			want:  &Eq{Field: "protocol", Value: 6},
		},
		{
			name:  "protocol equals TCP",
			input: "protocol = TCP",
			want:  &Eq{Field: "protocol", Value: 6},
		},
		{
			name:  "protocol equals tcp (lowercase)",
			input: "protocol = tcp",
			want:  &Eq{Field: "protocol", Value: 6},
		},
		{
			name:  "protocol equals UDP",
			input: "protocol = UDP",
			want:  &Eq{Field: "protocol", Value: 17},
		},
		{
			name:  "protocol equals udp (lowercase)",
			input: "protocol = udp",
			want:  &Eq{Field: "protocol", Value: 17},
		},
		{
			name:  "protocol equals ICMP",
			input: "protocol = ICMP",
			want:  &Eq{Field: "protocol", Value: 1},
		},
		{
			name:  "protocol greater than numeric",
			input: "protocol > 6",
			want:  &Gt{Field: "protocol", Value: 6},
		},
		{
			name:  "protocol greater than TCP",
			input: "protocol > TCP",
			want:  &Gt{Field: "protocol", Value: 6},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got Expr
			var err error

			// Use schema-aware parser for tests that need computed field support
			if strings.Contains(tt.name, "with schema") {
				schema := &VPCFlowLogsSchema{}
				got, err = ParseFilterWithSchema(tt.input, schema)
			} else {
				got, err = ParseFilter(tt.input)
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFilter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !exprsEqual(got, tt.want) {
				t.Errorf("ParseFilter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func exprsEqual(a, b Expr) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.String() == b.String()
}

// Helper functions for cleaner tests
func assertEq(t *testing.T, expr Expr, field string, value any) {
	t.Helper()
	eq, ok := expr.(*Eq)
	if !ok {
		t.Fatalf("expected *Eq, got %T", expr)
	}
	if eq.Field != field {
		t.Errorf("field = %q, want %q", eq.Field, field)
	}
	if eq.Value != value {
		t.Errorf("value = %v, want %v", eq.Value, value)
	}
}

func assertGt(t *testing.T, expr Expr, field string, value any) {
	t.Helper()
	gt, ok := expr.(*Gt)
	if !ok {
		t.Fatalf("expected *Gt, got %T", expr)
	}
	if gt.Field != field {
		t.Errorf("field = %q, want %q", gt.Field, field)
	}
	if gt.Value != value {
		t.Errorf("value = %v, want %v", gt.Value, value)
	}
}

func assertError(t *testing.T, err error, wantErr bool) {
	t.Helper()
	if (err != nil) != wantErr {
		t.Errorf("error = %v, wantErr %v", err, wantErr)
	}
}

func TestParsePortFieldExpr(t *testing.T) {
	t.Run("valid port", func(t *testing.T) {
		expr, err := parsePortFieldExpr("srcport", "=", "80")
		assertError(t, err, false)
		assertEq(t, expr, "srcport", 80)
	})
	t.Run("invalid port", func(t *testing.T) {
		_, err := parsePortFieldExpr("srcport", "=", "notaport")
		assertError(t, err, true)
	})
	t.Run("out of range port", func(t *testing.T) {
		_, err := parsePortFieldExpr("srcport", "=", "70000")
		assertError(t, err, true)
	})
}

func TestParseNumericFieldExpr(t *testing.T) {
	t.Run("valid int", func(t *testing.T) {
		expr, err := parseNumericFieldExpr("bytes", ">", "1000")
		assertError(t, err, false)
		assertGt(t, expr, "bytes", 1000)
	})
	t.Run("invalid numeric", func(t *testing.T) {
		_, err := parseNumericFieldExpr("bytes", ">", "notanumber")
		assertError(t, err, true)
	})
}

func TestParseProtocolFieldExpr(t *testing.T) {
	t.Run("numeric protocol", func(t *testing.T) {
		expr, err := parseProtocolFieldExpr("protocol", "=", "6")
		assertError(t, err, false)
		assertEq(t, expr, "protocol", 6)
	})
	t.Run("acronym protocol", func(t *testing.T) {
		expr, err := parseProtocolFieldExpr("protocol", "=", "TCP")
		assertError(t, err, false)
		assertEq(t, expr, "protocol", 6)
	})
	t.Run("unknown protocol", func(t *testing.T) {
		expr, err := parseProtocolFieldExpr("protocol", "=", "customproto")
		assertError(t, err, false)
		assertEq(t, expr, "protocol", "customproto")
	})
}

func TestIsValidIPPrefix(t *testing.T) {
	cases := []struct {
		input string
		want  bool
	}{
		{"10.0.0", true},
		{"10.0.0.0", true},
		{"10.0.0.0.0", false},
		{"10.0..0", false},
		{"10.0.0.256", false},
		{"10.0.0.a", false},
	}
	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			if got := isValidIPPrefix(c.input); got != c.want {
				t.Errorf("isValidIPPrefix(%q) = %v, want %v", c.input, got, c.want)
			}
		})
	}
}

func TestValidateFilter(t *testing.T) {
	schema := &VPCFlowLogsSchema{}
	t.Run("valid filter", func(t *testing.T) {
		err := ValidateFilter(&Eq{Field: "srcaddr", Value: "10.0.0.1"}, schema, 2)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	t.Run("invalid field", func(t *testing.T) {
		err := ValidateFilter(&Eq{Field: "nonexistent_field", Value: 1}, schema, 2)
		if err == nil {
			t.Error("expected error for invalid field")
		}
	})
	t.Run("invalid version", func(t *testing.T) {
		err := ValidateFilter(&Eq{Field: "srcaddr", Value: "10.0.0.1"}, schema, 999)
		if err == nil {
			t.Error("expected error for invalid version")
		}
	})
}
