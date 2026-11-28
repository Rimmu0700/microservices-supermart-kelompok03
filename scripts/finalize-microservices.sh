#!/usr/bin/env bash
# =====================================================================================
# NAMA       : finalize-microservices.sh
# FUNGSI     : Merapikan repo SETELAH semua service dipisah dan terbukti jalan: membuang sisa monolit
#              (backend/, skema/seed terpusat, compose lama, nginx gateway lama), menjadikan stack microservices
#              sebagai compose default (docker compose up --build), dan berhenti melacak .env.
#              Riwayat monolit tetap aman lewat git tag 'legacy-monolith-m1'.
# DIJALANKAN : bash scripts/finalize-microservices.sh            (DRY-RUN: hanya menampilkan rencana)
#              bash scripts/finalize-microservices.sh --apply    (menjalankan perubahan dan membuat 1 commit)
# SYARAT     : berada di branch (bukan main), working tree bersih, 10 service lengkap, audit tanpa FAIL.
# =====================================================================================
set -uo pipefail
APPLY=0; [ "${1:-}" = "--apply" ] && APPLY=1
ROOT="$(git rev-parse --show-toplevel 2>/dev/null)" || { echo "[ERROR] bukan repositori git"; exit 1; }
cd "$ROOT"
say(){ echo "  $*"; }
run(){ if [ $APPLY = 1 ]; then echo "  + $*"; eval "$@" || { echo "[ERROR] gagal: $*"; exit 1; }; else echo "  (rencana) $*"; fi; }

echo "== Pemeriksaan syarat"
br="$(git rev-parse --abbrev-ref HEAD)"
[ "$br" != "main" ] && [ "$br" != "HEAD" ] && say "[OK] branch: $br" || { echo "[ERROR] jangan jalankan di main; buat branch dulu: git checkout -b chore/finalize-microservices"; exit 1; }
[ -z "$(git status --porcelain)" ] && say "[OK] working tree bersih" || { echo "[ERROR] ada perubahan belum di-commit:"; git status --short | head; exit 1; }
n="$(ls -d services/*/ 2>/dev/null | wc -l | tr -d ' ')"; [ "$n" -ge 10 ] && say "[OK] $n service di services/" || { echo "[ERROR] baru $n service (butuh 10)"; exit 1; }
[ -f gateway/routes/order.json ] && say "[OK] gateway/routes/order.json ada" || { echo "[ERROR] gateway/routes/order.json belum ada"; exit 1; }
[ -f docker-compose.microservices.yml ] || [ -f docker-compose.yml ] || { echo "[ERROR] compose tidak ditemukan"; exit 1; }
if out="$(python3 scripts/audit/audit_microservices.py 2>&1)"; then say "[OK] audit tanpa FAIL"; else echo "$out" | grep FAIL; echo "[ERROR] audit masih ada FAIL"; exit 1; fi
echo
echo "== Rencana perubahan"
run "git tag -a legacy-monolith-m1 -m 'Arsip monolit Milestone 1 sebelum dibersihkan' 2>/dev/null || true"
mkdir -p docs/legacy 2>/dev/null
[ -f scripts/split_services.py ] && run "git mv scripts/split_services.py docs/legacy/split_services.py"
[ -d deployments/docker ] && run "git mv deployments/docker docs/legacy/tugas4-dockerfiles"
for x in backend deployments db.sql seed.sql scripts/split_db.py scripts/init-db.sh; do [ -e "$x" ] && run "git rm -r -q --ignore-unmatch $x"; done
if [ -f docker-compose.microservices.yml ]; then
  [ -f docker-compose.yml ] && run "git rm -q docker-compose.yml"
  run "git mv docker-compose.microservices.yml docker-compose.yml"
  files="$(grep -rIl --exclude-dir=.git --exclude-dir=node_modules --exclude-dir=legacy --exclude=finalize-microservices.sh --exclude=audit_microservices.py --exclude=split_services.py 'docker-compose.microservices.yml' . 2>/dev/null)"
  for f in $files; do run "sed -i.bak 's/docker-compose\\.microservices\\.yml/docker-compose.yml/g' '$f' && rm -f '$f.bak'"; done
fi
grep -qxF '.env' .gitignore 2>/dev/null || run "printf '\n.env\n' >> .gitignore"
git ls-files --error-unmatch .env >/dev/null 2>&1 && run "git rm --cached -q .env"
[ $APPLY = 1 ] && { run "git add -A"; run "git commit -q -m 'chore: buang sisa monolit, jadikan stack microservices sebagai compose default, berhenti melacak .env'"; echo; echo "SELESAI. Tag arsip: legacy-monolith-m1. Dorong dengan: git push -u origin $br  (dan: git push origin legacy-monolith-m1)"; } || { echo; echo "Ini hanya DRY-RUN. Jalankan dengan --apply setelah stack diuji dan audit bersih."; }
