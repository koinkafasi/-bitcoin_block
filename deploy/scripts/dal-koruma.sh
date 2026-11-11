#!/usr/bin/env bash
# dal-koruma.sh — GitHub deposunun tedarik zinciri ayarlarını BİR KEZ kurar.
#
# NEDEN: aşağıdakilerin hiçbiri kod değil, DEPO AYARIDIR; kod olarak
# işaretlenip unutuluyorlar. Bu betik onları tek yerden, tekrar koşulabilir
# biçimde uygular.
#
#   · dal koruması: main'e doğrudan push YOK, kuvvetli push YOK, silme YOK,
#     CI yeşil olmadan birleşme YOK.
#   · gizli tarama (secret scanning) + push koruması: sızmış anahtar
#     push edilmeden yakalanır.
#   · Dependabot/CodeQL: `codeql.yml` kod olarak var; depo ayarında
#     "code scanning" açık olmalı ki SARIF sonuçları kuyruğa düşsün.
#
# Kullanım:
#   bash deploy/scripts/dal-koruma.sh          # durumu göster (değişiklik yok)
#   bash deploy/scripts/dal-koruma.sh --uygula # ayarları uygula
#
# `gh` kimlik doğrulaması ve depo YÖNETİCİSİ yetkisi gerekir.
set -uo pipefail

REPO="koinkafasi/-bitcoin_block"
UYGULA=0
[ "${1:-}" = "--uygula" ] && UYGULA=1

command -v gh >/dev/null 2>&1 || { echo "gh gerekli" >&2; exit 1; }

echo "==> depo: $REPO"

if [ "$UYGULA" = 0 ]; then
  echo
  echo "--- dal koruması (main)"
  gh api "repos/$REPO/branches/main/protection" 2>/dev/null \
    | jq '{required_status_checks: .required_status_checks.contexts,
           enforce_admins: .enforce_admins.enabled,
           force_push: .allow_force_pushes.enabled,
           silme: .allow_deletions.enabled,
           inceleme: .required_pull_request_reviews.required_approving_review_count}' \
    || echo "  (koruma YOK — --uygula ile kurulur)"
  echo
  echo "--- güvenlik özellikleri"
  gh api "repos/$REPO" --jq '{secret_scanning: .security_and_analysis.secret_scanning.status,
                              push_protection: .security_and_analysis.secret_scanning_push_protection.status,
                              dependabot: .security_and_analysis.dependabot_security_updates.status}'
  echo
  echo "uygulamak için: bash deploy/scripts/dal-koruma.sh --uygula"
  exit 0
fi

echo "==> dal koruması uygulanıyor"
gh api --method PUT "repos/$REPO/branches/main/protection" \
  -H "Accept: application/vnd.github+json" \
  -F "required_status_checks[strict]=true" \
  -F "required_status_checks[contexts][]=bitcosar-ci / Güvenlik kapıları" \
  -F "required_status_checks[contexts][]=bitcosar-ci / Derleme ve test" \
  -F "required_status_checks[contexts][]=kalite / golangci-lint (yeni kod)" \
  -F "required_status_checks[contexts][]=kalite / Kapsam çıtı" \
  -F "enforce_admins=true" \
  -F "required_pull_request_reviews[required_approving_review_count]=0" \
  -F "restrictions=" \
  -F "allow_force_pushes=false" \
  -F "allow_deletions=false" \
  -F "required_conversation_resolution=true" \
  | jq '{required_status_checks: .required_status_checks.contexts, enforce_admins: .enforce_admins.enabled}'

echo "==> güvenlik özellikleri açılıyor"
gh api --method PATCH "repos/$REPO" \
  -H "Accept: application/vnd.github+json" \
  -F "security_and_analysis[secret_scanning][status]=enabled" \
  -F "security_and_analysis[secret_scanning_push_protection][status]=enabled" \
  -F "security_and_analysis[dependabot_security_updates][status]=enabled" \
  --jq '.security_and_analysis' 2>/dev/null \
  || echo "UYARI: güvenlik özellikleri API ile açılamadı (plan/kapsam)." \
        "Elle: Settings → Code security and analysis."

echo
echo "bitti. dal koruması yalnız CI iş adları DEPODA BİR KEZ KOŞTUKTAN sonra tam eşleşir."
