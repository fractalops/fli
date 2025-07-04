package querybuilder

import (
	"testing"
)

func TestEq(t *testing.T) {
	tests := []struct {
		name     string
		operator Eq
		want     string
	}{
		{
			name:     "string value",
			operator: Eq{Field: "ip", Value: "10.0.0.1"},
			want:     "ip = '10.0.0.1'",
		},
		{
			name:     "integer value",
			operator: Eq{Field: "port", Value: 80},
			want:     "port = 80",
		},
		{
			name:     "string with quotes",
			operator: Eq{Field: "ip", Value: "10.0'0.1"},
			want:     "ip = '10.0\\'0.1'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.operator.String(); got != tt.want {
				t.Errorf("Eq.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNeq(t *testing.T) {
	tests := []struct {
		name     string
		operator Neq
		want     string
	}{
		{
			name:     "string value",
			operator: Neq{Field: "ip", Value: "10.0.0.1"},
			want:     "ip != '10.0.0.1'",
		},
		{
			name:     "integer value",
			operator: Neq{Field: "port", Value: 80},
			want:     "port != 80",
		},
		{
			name:     "string with quotes",
			operator: Neq{Field: "ip", Value: "10.0'0.1"},
			want:     "ip != '10.0\\'0.1'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.operator.String(); got != tt.want {
				t.Errorf("Neq.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGt(t *testing.T) {
	tests := []struct {
		name     string
		operator Gt
		want     string
	}{
		{
			name:     "number value",
			operator: Gt{Field: "port", Value: 80},
			want:     "port > 80",
		},
		{
			name:     "string value",
			operator: Gt{Field: "timestamp", Value: "2024-01-01"},
			want:     "timestamp > '2024-01-01'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.operator.String(); got != tt.want {
				t.Errorf("Gt.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLt(t *testing.T) {
	tests := []struct {
		name     string
		operator Lt
		want     string
	}{
		{
			name:     "number value",
			operator: Lt{Field: "port", Value: 80},
			want:     "port < 80",
		},
		{
			name:     "string value",
			operator: Lt{Field: "timestamp", Value: "2024-01-01"},
			want:     "timestamp < '2024-01-01'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.operator.String(); got != tt.want {
				t.Errorf("Lt.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLike(t *testing.T) {
	tests := []struct {
		name     string
		operator *Like
		want     string
	}{
		{
			name:     "simple pattern",
			operator: &Like{Field: "ip", Value: "10.0"},
			want:     "ip like '10.0'",
		},
		{
			name:     "pattern with quotes",
			operator: &Like{Field: "ip", Value: "10.0"},
			want:     "ip like '10.0'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.operator.String(); got != tt.want {
				t.Errorf("Like.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNotLike(t *testing.T) {
	tests := []struct {
		name     string
		operator *NotLike
		want     string
	}{
		{
			name:     "simple pattern",
			operator: &NotLike{Field: "ip", Value: "10.0"},
			want:     "ip not like '10.0'",
		},
		{
			name:     "pattern with quotes",
			operator: &NotLike{Field: "ip", Value: "10.0"},
			want:     "ip not like '10.0'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.operator.String(); got != tt.want {
				t.Errorf("NotLike.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAnd(t *testing.T) {
	tests := []struct {
		name     string
		operator And
		want     string
	}{
		{
			name:     "empty",
			operator: And{},
			want:     "",
		},
		{
			name: "single expression",
			operator: And{
				&Eq{Field: "ip", Value: "10.0.0.1"},
			},
			want: "ip = '10.0.0.1'",
		},
		{
			name: "multiple expressions",
			operator: And{
				&Eq{Field: "ip", Value: "10.0.0.1"},
				&Eq{Field: "port", Value: 80},
			},
			want: "ip = '10.0.0.1' and port = 80",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.operator.String(); got != tt.want {
				t.Errorf("And.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOr(t *testing.T) {
	tests := []struct {
		name     string
		operator Or
		want     string
	}{
		{
			name:     "empty",
			operator: Or{},
			want:     "",
		},
		{
			name: "single expression",
			operator: Or{
				&Eq{Field: "ip", Value: "10.0.0.1"},
			},
			want: "(ip = '10.0.0.1')",
		},
		{
			name: "multiple expressions",
			operator: Or{
				&Eq{Field: "ip", Value: "10.0.0.1"},
				&Eq{Field: "ip", Value: "192.168.0.1"},
			},
			want: "(ip = '10.0.0.1' or ip = '192.168.0.1')",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.operator.String(); got != tt.want {
				t.Errorf("Or.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetFieldAndGetValue(t *testing.T) {
	cases := []struct {
		expr  FieldValueExpr
		field string
		value any
		name  string
	}{
		{Eq{Field: "foo", Value: 42}, "foo", 42, "Eq"},
		{Neq{Field: "bar", Value: "baz"}, "bar", "baz", "Neq"},
		{Gt{Field: "gt", Value: 1}, "gt", 1, "Gt"},
		{Lt{Field: "lt", Value: 2}, "lt", 2, "Lt"},
		{Like{Field: "like", Value: "abc"}, "like", "abc", "Like"},
		{NotLike{Field: "notlike", Value: "def"}, "notlike", "def", "NotLike"},
		{Gte{Field: "gte", Value: 3}, "gte", 3, "Gte"},
		{Lte{Field: "lte", Value: 4}, "lte", 4, "Lte"},
		{IsIpv4InSubnet{Field: "ip", Value: "10.0.0.0/24"}, "ip", "10.0.0.0/24", "IsIpv4InSubnet"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.expr.GetField(); got != c.field {
				t.Errorf("GetField() = %v, want %v", got, c.field)
			}
			if got := c.expr.GetValue(); got != c.value {
				t.Errorf("GetValue() = %v, want %v", got, c.value)
			}
		})
	}
}

func TestIsComputedField(t *testing.T) {
	cases := []struct {
		input string
		want  bool
	}{
		{"foo", false},
		{"foo - bar", true},
		{"foo/bar", true},
		{"foo*bar", true},
		{"foo+bar", true},
		{"foo (bar)", true},
		{"foo)bar", true},
		{"foo bar", true},
	}
	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			if got := isComputedField(c.input); got != c.want {
				t.Errorf("isComputedField(%q) = %v, want %v", c.input, got, c.want)
			}
		})
	}
}
