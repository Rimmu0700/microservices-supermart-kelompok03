#!/usr/bin/env python3
"""
build_krakend.py - menggabungkan gateway/base.json + gateway/routes/*.json menjadi gateway/krakend.json
dan memeriksa konsistensinya (tanpa perlu KrakenD):
  python3 gateway/build_krakend.py
Setiap anggota merawat berkas routes/<domain>.json miliknya; krakend.json adalah HASIL GABUNGAN (jalankan ulang
skrip ini setelah merge, lalu commit krakend.json oleh pemilik gateway).
"""
import json, os, re, sys
HERE = os.path.dirname(os.path.abspath(__file__))
ORDER = ["gateway", "identity", "catalog", "inventory", "order"]
base = json.load(open(os.path.join(HERE, "base.json"), encoding="utf-8"))
eps, problems = [], []
for name in ORDER:
    p = os.path.join(HERE, "routes", name + ".json")
    if not os.path.exists(p):
        print(f"[PERINGATAN] routes/{name}.json belum ada (domain belum di-merge?) - dilewati"); continue
    eps += json.load(open(p, encoding="utf-8"))["endpoints"]
seen = {}
for e in eps:
    key = (e["method"], e["endpoint"])
    if key in seen: problems.append(f"duplikat endpoint {key}")
    seen[key] = 1
    params = set(re.findall(r"\{(\w+)\}", e["endpoint"]))
    be = e["backend"]
    if e.get("output_encoding") == "no-op" and len(be) != 1: problems.append(f"{key}: no-op hanya boleh 1 backend")
    if e.get("output_encoding") != "no-op" and len(be) < 1: problems.append(f"{key}: tanpa backend")
    for b in be:
        bp = set(re.findall(r"\{(\w+)\}", b["url_pattern"]))
        if not bp <= params: problems.append(f"{key}: parameter backend {bp - params} tidak ada di endpoint")
        if not b["host"]: problems.append(f"{key}: host kosong")
# pemeriksaan konflik nama parameter pada posisi segmen yang sama (router Gin)
trie = {}
for (m, ep) in seen:
    node = trie
    for seg in ep.strip("/").split("/"):
        if seg.startswith("{"):
            node.setdefault("{}", {"names": set(), "kids": {}})["names"].add(seg)
            node = node["{}"]["kids"]
        else:
            node = node.setdefault(seg, {"names": set(), "kids": {}})["kids"]
def walk(n, path=""):
    for k, v in n.items():
        if k == "{}" and len(v["names"]) > 1: problems.append(f"nama parameter bertabrakan di {path}: {sorted(v['names'])}")
        walk(v["kids"], path + "/" + k)
walk(trie)
base["endpoints"] = eps
out = os.path.join(HERE, "krakend.json")
json.dump(base, open(out, "w", encoding="utf-8"), indent=2, ensure_ascii=False); open(out, "a").write("\n")
agg = [e["endpoint"] for e in eps if e.get("output_encoding") == "json" and len(e["backend"]) >= 2]
rl = [e["endpoint"] for e in eps if "qos/ratelimit/router" in e.get("extra_config", {})]
print(f"[OK] krakend.json: {len(eps)} endpoint | agregasi: {agg} | rate limit: {rl}")
if problems:
    print("[MASALAH]"); [print("  -", x) for x in problems]; sys.exit(1)
print("[OK] pemeriksaan konsistensi lulus")
