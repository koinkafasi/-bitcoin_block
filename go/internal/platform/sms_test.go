package platform

import "testing"

func TestNormalizeGSM(t *testing.T) {
	cases := map[string]string{
		"+90 532 111 22 33": "905321112233",
		"00905321112233":    "905321112233",
		"0532 111 22 33":    "905321112233",
		"5321112233":        "905321112233",
		"905321112233":      "905321112233",
		"":                  "",
		"   ":               "",
	}
	for in, want := range cases {
		if got := normalizeGSM(in); got != want {
			t.Errorf("normalizeGSM(%q) = %q, beklenen %q", in, got, want)
		}
	}
}

func TestSMSConfigEnabled(t *testing.T) {
	if (SMSConfig{}).Enabled() {
		t.Error("boş yapılandırma kapalı olmalı")
	}
	if (SMSConfig{Provider: "netgsm", NetgsmUser: "u"}).Enabled() {
		t.Error("Netgsm şifresiz kapalı olmalı")
	}
	if !(SMSConfig{Provider: "netgsm", NetgsmUser: "u", NetgsmPassword: "p"}).Enabled() {
		t.Error("Netgsm kullanıcı+şifre ile açık olmalı")
	}
	if !(SMSConfig{Provider: "twilio", TwilioSID: "s", TwilioToken: "t", From: "+1555"}).Enabled() {
		t.Error("Twilio tam yapılandırma ile açık olmalı")
	}
	if (SMSConfig{Provider: "twilio", TwilioSID: "s", TwilioToken: "t"}).Enabled() {
		t.Error("Twilio gönderen numarasız kapalı olmalı")
	}
	if (SMSConfig{Provider: "bilinmeyen", NetgsmUser: "u", NetgsmPassword: "p"}).Enabled() {
		t.Error("bilinmeyen sağlayıcı kapalı olmalı")
	}
}
