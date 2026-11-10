#!/bin/bash
# Sonar kalite kapisi + ozet — terminalden tek komut.
#
# Kullanim:  bash deploy/scripts/sonar-durum.sh
set -uo pipefail
PORT="${SONAR_PORT:-9100}"
PROJE=bitcosar
TOKEN=$(cat "$HOME/.sonar-token" 2>/dev/null)
P=$(cat "$HOME/.sonar-admin-pass" 2>/dev/null)

if [ -z "$TOKEN" ]; then echo "token yok" >&2; exit 1; fi
AUTH="admin:$P"
if [ -z "$P" ]; then AUTH="admin:$TOKEN"; fi

url() { echo "http://127.0.0.1:$PORT$1"; }

echo "=============== ÖLÇÜLER ==============="
curl -fsS -u "$AUTH" "$(url "/api/measures/component?component=$PROJE&metricKeys=alert_status,bugs,vulnerabilities,code_smells,security_hotspots,coverage,duplicated_lines_density,ncloc,reliability_rating,security_rating,sqale_index")" \
  | sed 's/},{/}\n{/g' \
  | sed -n 's/.*"metric":"\([^"]*\)","value":"\([^"]*\)".*/\1 = \2/p'

echo
echo "=============== KALİTE KAPISI ==============="
curl -fsS -u "$AUTH" "$(url "/api/qualitygates/project_status?projectKey=$PROJE")" > /tmp/sonar_qg.json
python3 - <<'PY'
import json
d = json.load(open('/tmp/sonar_qg.json'))
s = d['projectStatus']
print('durum:', s['status'])
for c in s.get('conditions', []):
    isaret = 'OK  ' if c['status'] == 'OK' else 'FAIL'
    print(f"  {isaret} {c['metricKey']:32} eşik={c.get('errorThreshold')}  gerçek={c.get('actualValue')}")
PY

echo
echo "=============== AÇIK BULGULAR (kural bazında) ==============="
curl -fsS -u "$AUTH" "$(url "/api/issues/search?componentKeys=$PROJE&ps=500&resolved=false")" > /tmp/sonar_is.json
python3 - <<'PY'
import json, collections
d = json.load(open('/tmp/sonar_is.json'))
iss = d.get('issues', [])
print('toplam açık:', len(iss))
by = collections.Counter((i['type'], i['rule']) for i in iss)
for (tip, kural), n in by.most_common(20):
    print(f"  {n:4}  {tip:13} {kural}")
PY

echo
echo "Panoda: http://127.0.0.1:$PORT/dashboard?id=$PROJE"
