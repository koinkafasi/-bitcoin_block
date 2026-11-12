package httpapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeAIString(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"json dizi ilk öğe", `["Türkçe başlık","ikinci başlık"]`, "Türkçe başlık"},
		{"json dizi boş ilk öğe", `["", "Dolu başlık"]`, "Dolu başlık"},
		{"tırnaklı düz metin", `"Türkçe spot metni"`, "Türkçe spot metni"},
		{"tırnaklı kaçışlı metin", `"T\u00fcrk\u00e7e ka\u00e7\u0131\u015fl\u0131"`, "Türkçe kaçışlı"},
		{"tırnaksız kaçışlı metin", `T\u00fcrk\u00e7e d\u00fcz metin`, "Türkçe düz metin"},
		{"json kod bloğu", "```json\n[\"Başlık şğüöçİı\"]\n```", "Başlık şğüöçİı"},
		{"sade kod bloğu", "```\n\"Sade tırnak\"\n```", "Sade tırnak"},
		{"kod bloğu + kaçış", "```json\n\"\u0130stanbul'da su baskını\"\n```", "İstanbul'da su baskını"},
		{"liste değil düz metin", "Düz metin — özet", "Düz metin — özet"},
		{"boş dizi", "[ ]", "[ ]"},
		{"baştaki/sondaki boşluk", "  Merhaba dünya  ", "Merhaba dünya"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, normalizeAIString(tc.in))
		})
	}
}

// TestStripCodeFence, tırnak/sarmal olmayan metnin aynen kaldığını doğrular.
func TestStripCodeFence(t *testing.T) {
	assert.Equal(t, "düz metin", stripCodeFence("düz metin"))
	assert.Equal(t, "içerik", stripCodeFence("```\niçerik\n```"))
	assert.Equal(t, "içerik", stripCodeFence("```text\r\niçerik\r\n```"))
}
