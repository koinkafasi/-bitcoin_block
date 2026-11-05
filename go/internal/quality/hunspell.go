package quality

import (
	"bufio"
	"bytes"
	"context"
	"os/exec"
	"sort"
	"strings"
	"time"
)

// Bu dosya isteğe bağlı sözlük tabanlı yazım denetimidir.
//
// Kural tabanlı denetim (imla.go) her ortamda çalışır ama sözlük gerektiren
// hataları (ör. "yanlış yazılmış sözcük") yakalayamaz. Sistemde hunspell ve
// ilgili sözlük varsa bu katman devreye girer; yoksa sessizce atlanır.
//
// Tasarım: hunspell'e metin stdin'den verilir, `-l` (liste) çıktısı hatalı
// sözcükleri döndürür. Dış çağrı 2 saniye ile sınırlanır; yavaş/eksik kurulum
// editörü bekletmez.

// HunspellBulgu sözlükte bulunamayan tek sözcüktür.
type HunspellBulgu struct {
	Word  string `json:"word"`
	Count int    `json:"count"`
}

// ImlaDenetleyici sözlük tabanlı denetleyicidir.
type ImlaDenetleyici struct {
	// Bin hunspell çalıştırılabilir dosyası (varsayılan "hunspell").
	Bin string
	// Dict "-d" ile verilecek sözlük adı (ör. "tr_TR"). Boşsa hunspell
	// varsayılanı kullanılır.
	Dict string
	// Timeout dış çağrı süresi (varsayılan 2 sn).
	Timeout time.Duration
}

// YeniImlaDenetleyici denetleyiciyi kurar. bin boşsa "hunspell".
func YeniImlaDenetleyici(bin, dict string) *ImlaDenetleyici {
	if strings.TrimSpace(bin) == "" {
		bin = "hunspell"
	}
	return &ImlaDenetleyici{Bin: bin, Dict: strings.TrimSpace(dict), Timeout: 2 * time.Second}
}

// Kullanilabilir hunspell ve sözlük gerçekten çalışıyor mu? Kısa bir deneme
// yapar; başarısızsa false döner (çağıran kural tabanlı denetimle devam eder).
func (d *ImlaDenetleyici) Kullanilabilir(ctx context.Context) bool {
	if d == nil {
		return false
	}
	if _, err := exec.LookPath(d.Bin); err != nil {
		return false
	}
	out, err := d.run(ctx, []string{"-l"}, "merhaba dunya")
	if err != nil {
		return false
	}
	// Hunspell desteklenmeyen dilde hata mesajı basar veya her sözcüğü
	// hatalı sayar. Sağlıklı bir sözlükte "merhaba" hatalı olmamalı.
	lines := nonEmptyLines(out)
	for _, l := range lines {
		if strings.EqualFold(l, "merhaba") {
			return false
		}
	}
	return true
}

// Denetle metni hunspell ile denetler. Hunspell yoksa/başarısızsa (nil, nil)
// döner: bu bir hata değil, "bu katman kapalı" demektir.
func (d *ImlaDenetleyici) Denetle(ctx context.Context, text string) ([]HunspellBulgu, error) {
	if d == nil || strings.TrimSpace(text) == "" {
		return nil, nil
	}
	args := []string{"-l"}
	if d.Dict != "" {
		args = append(args, "-d", d.Dict)
	}
	out, err := d.run(ctx, args, text)
	if err != nil {
		return nil, nil // sessizce kural tabanlı denetime düş
	}
	counts := map[string]int{}
	var order []string
	for _, line := range nonEmptyLines(out) {
		w := strings.TrimSpace(line)
		if w == "" || strings.ContainsAny(w, " \t") {
			continue
		}
		// Hunspell desteklenmeyen sözlükte "Can't open affix" gibi satırlar
		// basar; bunları bulgu sayma.
		if looksLikeHunspellError(w) {
			continue
		}
		if counts[w] == 0 {
			order = append(order, w)
		}
		counts[w]++
	}
	if len(counts) == 0 {
		return nil, nil
	}
	out2 := make([]HunspellBulgu, 0, len(order))
	for _, w := range order {
		out2 = append(out2, HunspellBulgu{Word: w, Count: counts[w]})
	}
	sort.SliceStable(out2, func(i, j int) bool {
		if out2[i].Count != out2[j].Count {
			return out2[i].Count > out2[j].Count
		}
		return out2[i].Word < out2[j].Word
	})
	return out2, nil
}

func (d *ImlaDenetleyici) run(ctx context.Context, args []string, stdin string) (string, error) {
	timeout := d.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(cctx, d.Bin, args...)
	cmd.Stdin = strings.NewReader(stdin)
	var out bytes.Buffer
	cmd.Stdout = &out
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return out.String(), nil
}

func nonEmptyLines(s string) []string {
	var out []string
	sc := bufio.NewScanner(strings.NewReader(s))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

// looksLikeHunspellError hunspell'in kurulum/hata çıktısını tanır.
func looksLikeHunspellError(s string) bool {
	low := strings.ToLower(s)
	for _, needle := range []string{"can't open", "cannot open", "error", "not found", "no dictionary"} {
		if strings.Contains(low, needle) {
			return true
		}
	}
	return false
}

// HecelemeBulguHazirla hunspell bulgularını ImlaBulgu biçimine çevirir;
// böylece panel ve puanlama tek biçim kullanır.
func HecelemeBulguHazirla(bulgular []HunspellBulgu) []ImlaBulgu {
	if len(bulgular) == 0 {
		return nil
	}
	words := make([]string, 0, len(bulgular))
	for _, b := range bulgular {
		words = append(words, b.Word)
	}
	if len(words) > 8 {
		words = append(words[:8], "…")
	}
	return []ImlaBulgu{{
		Code:    "imla_sozluk",
		Count:   len(bulgular),
		Sample:  strings.Join(words, ", "),
		Message: "Sözlükte bulunamayan sözcükler: " + strings.Join(words, ", "),
	}}
}
