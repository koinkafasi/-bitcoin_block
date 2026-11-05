// Package repo veritabanı erişim katmanıdır (pgxpool tabanlı).
package repo

import "errors"

// ErrNotFound kayıt bulunamadığında döner.
var ErrNotFound = errors.New("kayıt bulunamadı")

// ErrDuplicate benzersiz kısıt ihlali (ör. aynı adlı etiket) — Faz 41.
var ErrDuplicate = errors.New("kayıt zaten var")

// ErrConflict iyimser kilitleme çakışması (içerik başka biri tarafından
// değiştirildi) — AI editör / sürüm kontrolü.
var ErrConflict = errors.New("içerik sürümü değişti")

// ErrMediaInUse medya bir içerikte (kapak/manşet/sosyal) kullanıldığı için
// silinemediğinde döner (Faz 29). Eskiden DeleteMedia bu durumda hata
// vermiyor; handler diskteki dosyayı silip DB kaydını bırakıyordu.
var ErrMediaInUse = errors.New("medya kullanımda")
