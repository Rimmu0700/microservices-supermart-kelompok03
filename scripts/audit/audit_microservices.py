#!/usr/bin/env python3
"""
audit_microservices.py - audit otomatis: apakah repositori ini sudah MICROSERVICE (bukan monolit)?
Jalankan dari root repositori:   python3 scripts/audit/audit_microservices.py
Memeriksa kemandirian kode, data, deployment, dan sisa artefak monolit. Hasil: PASS / WARN / FAIL (exit 1 bila ada FAIL).
Tidak memerlukan Docker (bila PyYAML tidak ada, compose dibaca lewat 'docker compose config --format json').
"""
import os, re, sys, json, glob, hashlib, subprocess
ROOT = os.getcwd()
OK = WARN = FAIL = 0
USE_COLOR = sys.stdout.isatty()
def c(code, s): return f"\033[{code}m{s}\033[0m" if USE_COLOR else s
def p(kind, msg):
    global OK, WARN, FAIL
    if kind == "PASS": OK += 1; tag = c("32", "[PASS]")
    elif kind == "WARN": WARN += 1; tag = c("33", "[WARN]")
    else: FAIL += 1; tag = c("31", "[FAIL]")
    print(f"  {tag} {msg}")
def head(t): print("\n" + c("1;36", ">> " + t))
def rd(path):
    try: return open(path, encoding="utf-8").read()
    except Exception: return ""

# ---------------------------------------------------------------- compose
def load_compose():
    for name in ("docker-compose.microservices.yml", "docker-compose.yml"):
        if not os.path.exists(name): continue
        txt = rd(name)
        if "./services/" not in txt and name == "docker-compose.yml":
            continue
        try:
            import yaml
            return name, yaml.safe_load(txt)
        except ImportError:
            try:
                out = subprocess.run(["docker", "compose", "-f", name, "config", "--format", "json"], capture_output=True, text=True, timeout=60)
                if out.returncode == 0: return name, json.loads(out.stdout)
            except Exception: pass
    return None, None

svc_dirs = sorted(d for d in glob.glob("services/*") if os.path.isdir(d) and os.path.exists(os.path.join(d, "go.mod")))
names = [os.path.basename(d) for d in svc_dirs]
cname, comp = load_compose()
print(c("1", "AUDIT MICROSERVICE") + f"  | folder: {ROOT} | service ditemukan: {len(names)} | compose: {cname or '(tidak terbaca)'}")

# ---------------------------------------------------------------- A. kode
head("A. KEMANDIRIAN KODE (tiap service = unit build sendiri)")
EXPECT = 10
(p("PASS", f"{len(names)} service berdiri sendiri di services/") if len(names) >= EXPECT else p("FAIL", f"baru {len(names)} dari {EXPECT} service: {', '.join(names)} (belum semua service dimasukkan)"))
for d, n in zip(svc_dirs, names):
    need = ["go.mod", "go.sum", "Dockerfile", ".dockerignore", "db/schema.sql", "README.md", f"cmd/{n}/main.go"]
    miss = [x for x in need if not os.path.exists(os.path.join(d, x))]
    (p("PASS", f"{n}: berkas mandiri lengkap (go.mod, Dockerfile, db/schema.sql, README)") if not miss else p("FAIL", f"{n}: berkas hilang {miss}"))
    gm = rd(os.path.join(d, "go.mod"))
    if re.search(r"^\s*replace .*=>\s*\.\.", gm, re.M): p("FAIL", f"{n}: go.mod merujuk folder di luar service (replace => ..)")
    internal = [x for x in os.listdir(os.path.join(d, "internal"))] if os.path.isdir(os.path.join(d, "internal")) else []
    foreign = [x for x in internal if x != "messaging" and len(internal) > 2]
    if len(internal) > 2: p("FAIL", f"{n}: internal/ berisi lebih dari domain sendiri + messaging: {internal}")
    df = rd(os.path.join(d, "Dockerfile"))
    if re.search(r"COPY[^\n]*\.\./", df): p("FAIL", f"{n}: Dockerfile menyalin dari luar folder service")
    if not re.search(r"^USER\s+\S+", df, re.M): p("FAIL", f"{n}: Dockerfile tidak memakai USER non-root")
# impor lintas service
bad = []
for d, n in zip(svc_dirs, names):
    for f in glob.glob(os.path.join(d, "**", "*.go"), recursive=True):
        for m in re.finditer(r'"github\.com/[^"]+/(?:backend|services)/([^"]+)"', rd(f)):
            rel = m.group(1)
            if not os.path.isdir(os.path.join(d, rel.split("/")[0], *rel.split("/")[1:2])): bad.append((n, rel))
(p("PASS", "tidak ada impor kode antar service (setiap impor internal ada di dalam foldernya sendiri)") if not bad else p("FAIL", f"impor tak terpenuhi/lintas service: {bad[:3]}"))
hs = {}
for d, n in zip(svc_dirs, names):
    pm = os.path.join(d, "internal", "messaging")
    if os.path.isdir(pm):
        h = hashlib.md5(b"".join(open(f, "rb").read() for f in sorted(glob.glob(pm + "/*.go")))).hexdigest(); hs.setdefault(h, []).append(n)
if hs: p("WARN", f"kode pustaka internal/messaging disalin IDENTIK ke {sum(len(v) for v in hs.values())} service (kopling versi; solusi lanjut: pustaka berversi di repo terpisah)")

# ---------------------------------------------------------------- B. data
head("B. KEMANDIRIAN DATA (database-per-service dan kebebasan memilih DBMS)")
if comp:
    S = comp["services"]; vols = []
    for n in names:
        pref = {"notification-worker": "notification"}.get(n, n.replace("-service", ""))
        db = S.get(pref + "-db")
        if not db: p("FAIL", f"{n}: tidak punya container database sendiri ({pref}-db)"); continue
        env = S[n].get("environment", {}); dbenv = db.get("environment", {})
        if isinstance(env, list): env = dict(x.split("=", 1) for x in env)
        if isinstance(dbenv, list): dbenv = dict(x.split("=", 1) for x in dbenv)
        ok = env.get("MYSQL_HOST") == pref + "-db" and env.get("MYSQL_DATABASE") == dbenv.get("MYSQL_DATABASE")
        (p("PASS", f"{n}: memakai database miliknya ({pref}-db, {db['image']}, DB {dbenv.get('MYSQL_DATABASE')})") if ok else p("FAIL", f"{n}: MYSQL_HOST/DATABASE tidak menunjuk database miliknya"))
        vols += [v.split(":")[0] for v in db.get("volumes", []) if not v.startswith(".")]
    (p("PASS", f"{len(set(vols))} volume data terpisah, tidak ada yang dipakai bersama") if len(vols) == len(set(vols)) else p("FAIL", "ada volume database yang dipakai bersama"))
    engines = sorted({S[k]["image"] for k in S if k.endswith("-db")})
    p("PASS" if len(engines) > 1 else "WARN", f"mesin database yang dipakai: {', '.join(engines)}" + ("" if len(engines) > 1 else " -> SEMUA sama; kebebasan memilih DBMS didukung struktur tetapi BELUM dibuktikan (belum ada service yang berbeda mesin)"))
else:
    p("WARN", "compose tidak terbaca; pemeriksaan database dilewati")
# skema: nama database unik, tidak ada FK lintas database, tidak menyebut database service lain
dbnames = {}
for d, n in zip(svc_dirs, names):
    sql = rd(os.path.join(d, "db", "schema.sql"))
    dbs = set(re.findall(r"CREATE DATABASE(?: IF NOT EXISTS)?\s+`?(\w+)`?", sql, re.I))
    for x in dbs: dbnames.setdefault(x, []).append(n)
    tabs = set(re.findall(r"CREATE TABLE(?: IF NOT EXISTS)?\s+`?(\w+)`?", sql, re.I))
    refs = set(re.findall(r"REFERENCES\s+`?(\w+)`?", sql, re.I))
    if refs - tabs: p("FAIL", f"{n}: foreign key menunjuk tabel di luar skemanya: {sorted(refs - tabs)}")
dup = {k: v for k, v in dbnames.items() if len(v) > 1}
(p("PASS", f"{len(dbnames)} nama database unik; nol foreign key lintas database") if not dup and dbnames else p("FAIL", f"nama database dipakai lebih dari satu service: {dup}"))

# ---------------------------------------------------------------- C. deployment
head("C. KEMANDIRIAN DEPLOYMENT DAN ISOLASI JARINGAN")
if comp:
    S = comp["services"]
    for n in names:
        b = S.get(n, {}).get("build")
        ctx = b.get("context") if isinstance(b, dict) else b
        ctx = os.path.normpath(ctx or "")
        (p("PASS", f"{n}: dibangun dari folder sendiri ({ctx})") if ctx.replace("\\", "/").endswith("services/" + n) else p("FAIL", f"{n}: konteks build {ctx} bukan folder sendiri"))
    hp = {k: v["ports"] for k, v in S.items() if v.get("ports")}
    leak = [k for k in hp if k in names or k.endswith("-db")]
    (p("PASS", "tidak ada service/database yang membuka port ke host") if not leak else p("FAIL", f"port host terbuka pada: {leak}"))
    info = ", ".join(f"{k}={v}" for k, v in hp.items()); p("PASS", f"port host yang terbuka: {info}")
    gw = [k for k in S if "gateway" in k]
    (p("PASS", f"pintu masuk tunggal: {', '.join(gw)}") if gw else p("WARN", "tidak ada API Gateway pada compose"))
kj = rd("gateway/krakend.json")
if kj:
    hosts = set(re.findall(r"http://([\w-]+):3000", kj))
    miss = sorted(hosts - set(names))
    (p("PASS", f"gateway merutekan ke {len(hosts)} service yang semuanya ada di services/") if not miss else p("FAIL", f"gateway merujuk service yang tidak ada: {miss}"))
    for r in ("order", "identity", "catalog", "inventory", "gateway"):
        if not os.path.exists(f"gateway/routes/{r}.json"): p("FAIL", f"gateway/routes/{r}.json belum ada (krakend.json tidak bisa dibangun ulang lengkap)")

# ---------------------------------------------------------------- D. rahasia
head("D. KEMANDIRIAN KREDENSIAL")
if comp:
    jw = {}
    for n in names:
        env = comp["services"].get(n, {}).get("environment", {})
        if isinstance(env, list): env = dict(x.split("=", 1) for x in env)
        jw.setdefault(env.get("JWT_SECRET"), []).append(n)
    if len(jw) == 1: p("WARN", "JWT_SECRET dan INTERNAL_TOKEN sama untuk semua service (satu rahasia bersama); perbaikan: JWT asimetris + token per service")
    else: p("PASS", "kredensial berbeda per service")

# ---------------------------------------------------------------- E. sisa monolit
head("E. SISA ARTEFAK MONOLIT (yang membuat repo terlihat monolit)")
if os.path.isdir("backend") and os.path.exists("backend/go.mod"):
    cmds = glob.glob("backend/cmd/*")
    p("WARN", f"folder backend/ (satu modul Go berisi {len(cmds)} cmd) masih ada -> ini kode monolit lama; hapus lewat scripts/finalize-microservices.sh")
else: p("PASS", "tidak ada folder backend/ monolit")
old = rd("docker-compose.yml")
if "context: ./backend" in old or "./backend" in old: p("WARN", "docker-compose.yml bawaan masih membangun dari ./backend (stack lama); 'docker compose up' default akan menjalankan stack lama")
elif os.path.exists("docker-compose.yml"): p("PASS", "docker-compose.yml default sudah stack microservices")
if os.path.exists("db.sql") or os.path.exists("seed.sql") or os.path.isdir("deployments/sql/generated"):
    p("WARN", "skema/seed terpusat (db.sql, seed.sql, deployments/sql/) masih ada; skema kini milik tiap service di services/*/db/schema.sql")
title = next((l for l in rd("README.md").splitlines() if l.startswith("#")), "")
if re.search(r"monolith", title, re.I): p("WARN", "judul README.md utama masih 'Monolith Seed Project' -> perbarui menjadi README microservice")
else: p("PASS", "README.md utama tidak lagi menyebut monolit")
if os.path.exists(".env") and ".env" not in [l.strip() for l in rd(".gitignore").splitlines()]: p("WARN", ".env ada dan tidak ada di .gitignore (berisi kredensial)")

print("\n" + c("1", f"RINGKASAN: {OK} PASS, {WARN} WARN, {FAIL} FAIL"))
if FAIL: print(c("31", "Ada FAIL: perbaiki dulu sebelum menyatakan arsitektur sudah microservice.")); sys.exit(1)
print(c("32", "Tidak ada FAIL.") + (" WARN adalah hal yang masih perlu dijelaskan/dirapikan." if WARN else ""))
