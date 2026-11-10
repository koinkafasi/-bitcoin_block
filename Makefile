SHELL := /bin/bash
.DEFAULT_GOAL := yardim

yardim: ## Bu listeyi göster
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) \
	  | awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

guvenlik-tara: ## Güvenlik kapıları (CI ile birebir aynı; kırmızıysa çıkış 1)
	@bash deploy/scripts/guvenlik-tara.sh

guvenlik-rapor: ## Güvenlik taraması, yalnız rapor (kırmaz)
	@bash deploy/scripts/guvenlik-tara.sh --rapor

guvenlik-imaj: ## Kap imajlarını derle ve tara
	@bash deploy/scripts/guvenlik-tara.sh --imaj

performans: ## Lighthouse bütçesi (perf/SEO/a11y/best-practices)
	@bash deploy/scripts/lighthouse-kapi.sh

performans-rapor: ## Lighthouse, yalnız rapor (kırmaz)
	@bash deploy/scripts/lighthouse-kapi.sh --rapor

derle: ## Site binary'sini derle
	@cd go && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/site ./cmd/site

templ: ## templ şablonlarını üret
	@go run github.com/a-h/templ/cmd/templ@v0.3.1070 generate

tunnel-durum: ## demo.bitcosar.com tünelinin durumu
	@systemctl --user --no-pager --lines=0 status bitcosar-tunnel.service || true
	@curl -sS -o /dev/null -w 'https://demo.bitcosar.com → HTTP %{http_code}\n' \
	  --max-time 15 https://demo.bitcosar.com/ || true

# ---- SonarQube (statik analiz) ----
# Yerel SonarQube (Docker, :9100). Kurulum bir kez: `sonar-kur`.
sonar-kur: ## SonarQube'u kur/başlat (ilk kez veya makine açılışında)
	@bash deploy/scripts/sonar-kur.sh

sonar-tara: ## Kapsamı üret + SonarQube taraması
	@bash deploy/scripts/sonar-tara.sh

sonar-durum: ## Kalite kapısı + açık bulgular özeti
	@bash deploy/scripts/sonar-durum.sh

.PHONY: yardim guvenlik-tara guvenlik-rapor guvenlik-imaj performans \
        performans-rapor derle templ tunnel-durum \
        sonar-kur sonar-tara sonar-durum
