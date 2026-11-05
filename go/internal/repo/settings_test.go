package repo

import (
	"bytes"
	"testing"
)

func TestSettingJSONSarma(t *testing.T) {
	// Geçerli JSON olduğu gibi kalmalı.
	for _, v := range []string{`0`, `true`, `{"a":1}`, `[1,2]`, `"quoted"`, `null`} {
		if got := string(settingJSON([]byte(v))); got != v {
			t.Errorf("settingJSON(%q) = %q, değişmemeli", v, got)
		}
	}
	// Geçersiz JSON sarılmalı.
	for _, v := range []string{`09:41`, `netgsm`, `Ad|URL`, `abc"def`, ``} {
		got := settingJSON([]byte(v))
		if bytes.HasPrefix(got, []byte(`"`)) && bytes.HasSuffix(got, []byte(`"`)) {
			continue // JSON string sarmalı
		}
		t.Errorf("settingJSON(%q) = %q, JSON string sarmalı beklenirdi", v, got)
	}
}

func TestSettingTextGeriCozum(t *testing.T) {
	cases := map[string]string{
		`"09:41"`:     `09:41`,
		`"netgsm"`:    `netgsm`,
		`""`:          ``,
		`0`:           `0`,
		`true`:        `true`,
		`{"a":1}`:     `{"a":1}`,
		`[1,2]`:       `[1,2]`,
		`yarim "json`: `yarim "json`,
	}
	for in, want := range cases {
		if got := string(settingText([]byte(in))); got != want {
			t.Errorf("settingText(%q) = %q, beklenen %q", in, got, want)
		}
	}
}

func TestSettingRoundTrip(t *testing.T) {
	for _, v := range []string{`09:41`, `netgsm`, `THEBLOCK`, `Ad|URL satiri`, `+90 555`, `{"tr":"ana"}`} {
		got := string(settingText(settingJSON([]byte(v))))
		if got != v {
			t.Errorf("round-trip(%q) = %q", v, got)
		}
	}
}
