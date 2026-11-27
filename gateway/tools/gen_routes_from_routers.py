#!/usr/bin/env python3
"""
gen_routes_from_routers.py - membuat KERANGKA AWAL gateway/routes/<domain>.json dari berkas router Go (Fiber).
Dijalankan SEKALI untuk membuat kerangka; setelah itu berkas routes/*.json dirawat manual (rate limit, agregasi, dst.).
  python3 gateway/tools/gen_routes_from_routers.py <folder-sumber> <folder-keluaran>
<folder-sumber> berisi internal/<nama>/router.go (mis. backend/ atau services/<svc>/).
"""
import os, re, sys, json
SRC, OUT = sys.argv[1], sys.argv[2]
DOMAINS = {
  "identity":  [("auth", "auth-service"), ("support", "support-service")],
  "catalog":   [("catalog", "catalog-service"), ("promotion", "promotion-service")],
  "inventory": [("inventory", "inventory-service"), ("procurement", "procurement-service")],
  "order":     [("order", "order-service"), ("payment", "payment-service"), ("logistics", "logistics-service")],
}
QUERY = {}   # nama service -> query string yang dibaca handler
for _, svc in sum(DOMAINS.values(), []):
    pass
def queries(pkg):
    q = set()
    d = os.path.join(SRC, "internal", pkg)
    for f in os.listdir(d):
        if f.endswith(".go") and not f.endswith("_test.go"):
            q |= set(re.findall(r'Query\("([a-z_]+)"', open(os.path.join(d, f), encoding="utf-8").read()))
    return sorted(q)
ROUTE = re.compile(r'(\w+)\.(Get|Post|Put|Patch|Delete)\("([^"]*)"')
GROUP = re.compile(r'(\w+)\s*:=\s*\w+\.Group\("([^"]*)"')
def conv(path): return re.sub(r':(\w+)', r'{\1}', path)
for dom, pairs in DOMAINS.items():
    eps = []
    for pkg, svc in pairs:
        t = open(os.path.join(SRC, "internal", pkg, "router.go"), encoding="utf-8").read()
        groups = {v: p for v, p in GROUP.findall(t)}
        qs = queries(pkg)
        for var, method, path in ROUTE.findall(t):
            if var not in groups: continue
            full = "/api/v1" + groups[var] + ("" if path in ("", "/") else path)
            full = conv(full)
            ep = {"endpoint": full, "method": method.upper(), "output_encoding": "no-op",
                  "input_headers": ["Authorization", "Content-Type", "Accept"]}
            if qs and method == "Get": ep["input_query_strings"] = qs
            ep["backend"] = [{"url_pattern": full, "encoding": "no-op", "host": [f"http://{svc}:3000"]}]
            eps.append(ep)
    json.dump({"endpoints": eps}, open(os.path.join(OUT, f"{dom}.json"), "w", encoding="utf-8"), indent=2, ensure_ascii=False)
    print(dom, len(eps), "rute")
