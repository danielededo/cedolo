package money

import "testing"

func TestParse(t *testing.T) {
	cases := []struct {
		in   string
		want int64
	}{
		{"0,00", 0},
		{"1,5", 150},
		{"12,30", 1230},
		{"1.234,56", 123456},
		{"1234,56", 123456},
		{"1.234.567,89", 123456789},
		{"-12,30", -1230},
		{"12,30-", -1230},
		{" 7 ", 700},
		{"€ 1.234,56", 123456},
		{"1.234,56€", 123456},
		{"EUR 12,30-", -1230},
	}
	for _, c := range cases {
		got, err := Parse(c.in)
		if err != nil || got != c.want {
			t.Errorf("Parse(%q) = %d, %v; want %d", c.in, got, err, c.want)
		}
	}
}

func TestParseRejects(t *testing.T) {
	for _, in := range []string{"", "abc", "1,234", "1.23,45", "-12,30-", "12.5", "1,2,3", "12,3a"} {
		if _, err := Parse(in); err == nil {
			t.Errorf("Parse(%q) succeeded, want error", in)
		}
	}
}

func TestIsAmount(t *testing.T) {
	for in, want := range map[string]bool{"1.234,56": true, "12,30-": true, "€1.234,56": true, "€": false, "173": false, "40,000": false, "RSSMRA": false} {
		if got := IsAmount(in); got != want {
			t.Errorf("IsAmount(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestFormatRoundTrip(t *testing.T) {
	for _, v := range []int64{0, 5, 99, 100, 123456, -1230, 123456789} {
		s := Format(v)
		got, err := Parse(s)
		if err != nil || got != v {
			t.Errorf("Parse(Format(%d)) = %d, %v (formatted %q)", v, got, err, s)
		}
	}
	if got := Format(123456789); got != "1.234.567,89" {
		t.Errorf("Format = %q", got)
	}
}
