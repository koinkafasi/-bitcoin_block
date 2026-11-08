package repo

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetSettingsTextSarmaliCozer: settings tablosunda değerler jsonb olarak
// tutulur; SetSetting geçerli JSON olmayan değerleri JSON string'e sarar.
// Tek sorguyla bölüm okuyan çağıranların tırnaklı değer görmemesi gerekir
// (youtube ayarları `"bitcosar"` yerine `bitcosar` okumalı).
func TestGetSettingsTextSarmaliCozer(t *testing.T) {
	db := &fakeDB{queryFn: func(string, []any) ([][]any, error) {
		return [][]any{
			{"kanal_id", []byte(`"UCPtM806ijpKatyIHH5w4kmg"`)},
			{"dakika", []byte(`15`)},
			{"aktif", []byte(`true`)},
		}, nil
	}}

	got, err := GetSettingsText(context.Background(), db, "youtube")
	require.NoError(t, err)
	assert.Equal(t, "UCPtM806ijpKatyIHH5w4kmg", got["kanal_id"])
	assert.Equal(t, "15", got["dakika"])
	assert.Equal(t, "true", got["aktif"])
}

// TestGetSettingsTextTekSorgu: anahtar başına sorgu YAPILMAZ (N+1 geri
// gelmesin). Bu, youtube.AyarlariOku'nun 10 sorgudan 1'e inmesinin temeli.
func TestGetSettingsTextTekSorgu(t *testing.T) {
	db := &fakeDB{queryFn: func(string, []any) ([][]any, error) {
		return [][]any{{"kanal_id", []byte(`"x"`)}}, nil
	}}
	_, err := GetSettingsText(context.Background(), db, "youtube")
	require.NoError(t, err)
	require.Len(t, db.Calls(), 1)
	assert.Equal(t, "query", db.Calls()[0].Kind)
}
