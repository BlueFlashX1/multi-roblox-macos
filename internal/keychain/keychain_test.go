package keychain

import "testing"

func TestQuoteToken(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{`plain`, `"plain"`},
		{`has space`, `"has space"`},
		{`quo"te`, `"quo\"te"`},
		{`back\slash`, `"back\\slash"`},
		{`both\"mixed`, `"both\\\"mixed"`},
		{`_|WARNING:-cookie-style-value_1234ABCD`, `"_|WARNING:-cookie-style-value_1234ABCD"`},
	}
	for _, c := range cases {
		if got := quoteToken(c.in); got != c.want {
			t.Errorf("quoteToken(%q) = %s, want %s", c.in, got, c.want)
		}
	}
}

func TestStoreRejectsNewlinesAndEmpty(t *testing.T) {
	if err := StoreGenericPassword("svc", "acct", "line1\nline2"); err == nil {
		t.Error("expected error for newline in secret")
	}
	if err := StoreGenericPassword("svc", "acct", ""); err == nil {
		t.Error("expected error for empty secret")
	}
}
