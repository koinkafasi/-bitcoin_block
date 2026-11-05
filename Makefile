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

tunnel-durum: ## demo.bitcosar.com tünelinin durumu
	@systemctl --user --no-pager --lines=0 status bitcosar-tunnel.service || true
	@curl -sS -o /dev/null -w 'https://demo.bitcosar.com → HTTP %{http_code}\n' \
	  --max-time 15 https://demo.bitcosar.com/ || true

.PHONY: yardim guvenlik-tara guvenlik-rapor guvenlik-imaj tunnel-durum
