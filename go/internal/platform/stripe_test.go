package platform

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"
)

func TestVerifyWebhookSignature(t *testing.T) {
	secret := "whsec_test_123"
	payload := []byte(`{"id":"evt_1","type":"checkout.session.completed"}`)
	now := time.Unix(1700000000, 0)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte("1700000000." + string(payload)))
	good := "t=1700000000,v1=" + hex.EncodeToString(mac.Sum(nil))

	if err := VerifyWebhookSignature(payload, good, secret, now); err != nil {
		t.Errorf("geçerli imza reddedildi: %v", err)
	}
	// çoklu v1: ilki yanlış, ikincisi doğru
	mac2 := hmac.New(sha256.New, []byte(secret))
	mac2.Write([]byte("1700000000." + string(payload)))
	multi := "t=1700000000,v1=deadbeef,v1=" + hex.EncodeToString(mac2.Sum(nil))
	if err := VerifyWebhookSignature(payload, multi, secret, now); err != nil {
		t.Errorf("çoklu imzada doğru olan bulunamadı: %v", err)
	}

	if err := VerifyWebhookSignature(payload, good, "yanlis", now); err == nil {
		t.Error("yanlış sırla imza geçmemeli")
	}
	if err := VerifyWebhookSignature([]byte(`{"tamamen":"farkli"}`), good, secret, now); err == nil {
		t.Error("gövde değişince imza geçmemeli")
	}
	// zaman toleransı (6 dk eski)
	macOld := hmac.New(sha256.New, []byte(secret))
	macOld.Write([]byte("1699999640." + string(payload)))
	oldSig := "t=1699999640,v1=" + hex.EncodeToString(macOld.Sum(nil))
	if err := VerifyWebhookSignature(payload, oldSig, secret, now); err == nil {
		t.Error("6 dk eski imza tolerans dışında olmalı")
	}
	if err := VerifyWebhookSignature(payload, "", secret, now); err == nil {
		t.Error("boş başlık geçmemeli")
	}
	if err := VerifyWebhookSignature(payload, good, "", now); err == nil {
		t.Error("boş sırla imza geçmemeli")
	}
}

func TestParseStripeEvent(t *testing.T) {
	// checkout.session.completed
	ev, err := ParseStripeEvent([]byte(`{
		"id":"evt_1","type":"checkout.session.completed",
		"data":{"object":{"customer":"cus_1","subscription":"sub_1",
			"customer_email":"okuyucu@test.com","metadata":{"email":"meta@test.com"}}}
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if ev.ID != "evt_1" || ev.Type != "checkout.session.completed" {
		t.Errorf("kimlik/tür yanlış: %+v", ev)
	}
	if ev.CustomerID != "cus_1" || ev.SubscriptionID != "sub_1" {
		t.Errorf("müşteri/abonelik yanlış: %+v", ev)
	}
	if ev.Email != "okuyucu@test.com" {
		t.Errorf("e-posta yanlış: %+v", ev)
	}

	// subscription.updated — id üst düzey object.id, süre üst düzeyde
	ev2, err := ParseStripeEvent([]byte(`{
		"id":"evt_2","type":"customer.subscription.updated",
		"data":{"object":{"id":"sub_2","customer":"cus_2","status":"active",
			"current_period_end":1700000500,"metadata":{"email":"x@test.com"}}}
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if ev2.SubscriptionID != "sub_2" || ev2.SubscriptionStat != "active" {
		t.Errorf("abonelik alanları yanlış: %+v", ev2)
	}
	if ev2.PeriodEnd.Unix() != 1700000500 {
		t.Errorf("dönem sonu yanlış: %+v", ev2)
	}

	// metadata e-postası yedeği
	ev3, _ := ParseStripeEvent([]byte(`{
		"id":"evt_3","type":"customer.subscription.deleted",
		"data":{"object":{"id":"sub_3","customer":"cus_3","status":"canceled","metadata":{"email":"y@test.com"}}}
	}`))
	if ev3.Email != "y@test.com" {
		t.Errorf("metadata e-postası okunmalı: %+v", ev3)
	}
}

func TestStripeConfigEnabled(t *testing.T) {
	if (StripeConfig{}).Enabled() {
		t.Error("boş yapılandırma kapalı olmalı")
	}
	if (StripeConfig{SecretKey: "sk"}).Enabled() {
		t.Error("price id'siz kapalı olmalı")
	}
	if !(StripeConfig{SecretKey: "sk", PriceID: "price_1"}).Enabled() {
		t.Error("anahtar+fiyat ile açık olmalı")
	}
}
