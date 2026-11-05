#!/usr/bin/env python3
"""
split_db.py - memecah db.sql + seed.sql (monolit, 120 tabel) menjadi satu file SQL per service.

Pemakaian (dari root repo):   python3 scripts/split_db.py
Keluaran:                     deployments/sql/generated/<nn>-<service>.sql
"""
import os
import re
import sys

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
DB_SQL = os.path.join(ROOT, "db.sql")
SEED_SQL = os.path.join(ROOT, "seed.sql")
OUT_DIR = os.path.join(ROOT, "deployments", "sql", "generated")

# KLUSTER 1..9 pada db.sql  ->  service pemilik
SERVICES = [
    "auth", "catalog", "inventory", "order", "payment",
    "promotion", "logistics", "procurement", "support",
]

# Tabel milik service lain yang dibaca lewat JOIN, disalin sebagai replika baca-saja.
# service pemakai -> daftar tabel replika (pemilik asli ditentukan otomatis dari db.sql)
REPLICAS = {
    "catalog":     ["users"],
    "inventory":   ["products", "users"],
    "order":       ["products", "product_images", "users", "user_addresses", "warehouses"],
    "payment":     ["orders", "users"],
    "promotion":   ["products", "product_images"],
    "logistics":   ["orders", "order_shipping_details", "users", "user_addresses"],
    "procurement": ["products", "warehouses"],
    "support":     ["users", "user_roles", "roles", "orders"],
}
# Kolom yang tidak boleh ikut direplikasi
SENSITIVE_COLUMNS = {"users": ["password_hash"]}

MESSAGING_SQL = """
CREATE TABLE IF NOT EXISTS `processed_events` (
    `consumer_name` VARCHAR(100) NOT NULL,
    `event_id`      VARCHAR(36)  NOT NULL,
    `event_type`    VARCHAR(100) NOT NULL,
    `processed_at`  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`consumer_name`, `event_id`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `outbox_events` (
    `id`             VARCHAR(36)  PRIMARY KEY,
    `aggregate_type` VARCHAR(50)  NOT NULL,
    `aggregate_id`   VARCHAR(36)  NOT NULL,
    `event_type`     VARCHAR(100) NOT NULL,
    `routing_key`    VARCHAR(255) NOT NULL,
    `payload`        JSON         NOT NULL,
    `created_at`     DATETIME(3)  DEFAULT CURRENT_TIMESTAMP(3),
    `published_at`   DATETIME(3)  NULL,
    INDEX `idx_outbox_unpublished` (`published_at`, `created_at`)
) ENGINE=InnoDB;
"""


def split_statements(text):
    """Pisahkan SQL menjadi statement pada ';' dengan memperhatikan string dan komentar."""
    stmts, buf = [], []
    i, n = 0, len(text)
    quote = None
    while i < n:
        c = text[i]
        if quote:
            buf.append(c)
            if c == "\\" and i + 1 < n:
                buf.append(text[i + 1])
                i += 2
                continue
            if c == quote:
                if i + 1 < n and text[i + 1] == quote:  # '' atau ""
                    buf.append(text[i + 1])
                    i += 2
                    continue
                quote = None
            i += 1
            continue
        if c in ("'", '"', "`"):
            quote = c
            buf.append(c)
            i += 1
            continue
        if c == "-" and text.startswith("--", i):
            j = text.find("\n", i)
            j = n if j == -1 else j
            i = j  # buang komentar baris
            continue
        if c == "/" and text.startswith("/*", i):
            j = text.find("*/", i + 2)
            i = n if j == -1 else j + 2
            continue
        if c == ";":
            s = "".join(buf).strip()
            if s:
                stmts.append(s)
            buf = []
            i += 1
            continue
        buf.append(c)
        i += 1
    tail = "".join(buf).strip()
    if tail:
        stmts.append(tail)
    return stmts


def cluster_of_ddl(db_text):
    """Kembalikan {nama_tabel: (indeks_kluster, statement_create_table)} mengikuti header KLUSTER."""
    parts = re.split(r"(?m)^-- KLUSTER (\d+):", db_text)
    tables = {}
    # parts[0] = sebelum kluster 1, lalu (nomor, isi) berselang-seling
    for k in range(1, len(parts), 2):
        idx = int(parts[k]) - 1
        body = parts[k + 1].split("\n", 1)[1]  # buang sisa baris judul kluster
        for st in split_statements(body):
            m = re.match(r"CREATE TABLE IF NOT EXISTS `(\w+)`", st)
            if m:
                tables[m.group(1)] = (idx, st)
    return tables


def element_lines(stmt):
    """Pecah isi tanda kurung CREATE TABLE menjadi elemen top-level (kolom/constraint)."""
    start = stmt.index("(")
    depth, end = 0, None
    for i in range(start, len(stmt)):
        if stmt[i] == "(":
            depth += 1
        elif stmt[i] == ")":
            depth -= 1
            if depth == 0:
                end = i
                break
    head, body, tail = stmt[:start], stmt[start + 1:end], stmt[end + 1:]
    elems, buf, depth, quote = [], [], 0, None
    for c in body:
        if quote:
            buf.append(c)
            if c == quote:
                quote = None
            continue
        if c in ("'", '"', "`"):
            quote = c
            buf.append(c)
        elif c == "(":
            depth += 1
            buf.append(c)
        elif c == ")":
            depth -= 1
            buf.append(c)
        elif c == "," and depth == 0:
            elems.append("".join(buf).strip())
            buf = []
        else:
            buf.append(c)
    last = "".join(buf).strip()
    if last:
        elems.append(last)
    return head, elems, tail


FK_RE = re.compile(r"FOREIGN KEY\s*\([^)]*\)\s*REFERENCES\s*`(\w+)`", re.I)


def rebuild(stmt, keep_fk, drop_columns=()):
    head, elems, tail = element_lines(stmt)
    kept = []
    for e in elems:
        m = FK_RE.search(e)
        if m and not keep_fk(m.group(1)):
            continue
        col = re.match(r"`(\w+)`", e)
        if col and col.group(1) in drop_columns:
            continue
        kept.append(e)
    body = ",\n    ".join(re.sub(r"\s+", " ", e) for e in kept)
    return "%s(\n    %s\n)%s;" % (head, body, tail.rstrip())


def table_of_insert(st):
    m = re.match(r"INSERT\s+(?:IGNORE\s+)?INTO\s+`(\w+)`", st, re.I)
    return m.group(1) if m else None


def main():
    db_text = open(DB_SQL, encoding="utf-8").read()
    seed_text = open(SEED_SQL, encoding="utf-8").read()
    tables = cluster_of_ddl(db_text)
    owner = {t: SERVICES[idx] for t, (idx, _) in tables.items()}

    if len(tables) != 120:
        print("PERINGATAN: ditemukan %d tabel (diharapkan 120)" % len(tables), file=sys.stderr)

    seed_by_service = {s: [] for s in SERVICES}
    skipped = []
    for st in split_statements(seed_text):
        t = table_of_insert(st)
        if t is None:
            continue  # USE / SET diabaikan, tiap file punya header sendiri
        if t not in owner:
            skipped.append(t)
            continue
        seed_by_service[owner[t]].append(st + ";")

    os.makedirs(OUT_DIR, exist_ok=True)
    for old in os.listdir(OUT_DIR):
        if old.endswith(".sql"):
            os.remove(os.path.join(OUT_DIR, old))

    summary = []
    for n, svc in enumerate(SERVICES, start=1):
        dbname = "%s_db" % svc
        own = [t for t, s in owner.items() if s == svc]
        lines = [
            "-- File ini DIHASILKAN oleh scripts/split_db.py. Jangan diedit manual.",
            "CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;" % dbname,
            "USE `%s`;" % dbname,
            "SET FOREIGN_KEY_CHECKS = 0;",
            "",
            "-- ===== Tabel milik %s-service =====" % svc,
        ]
        for t in own:
            _, st = tables[t]
            lines.append(rebuild(st, keep_fk=lambda ref, o=own: ref in o))
            lines.append("")
        reps = REPLICAS.get(svc, [])
        if reps:
            lines.append("-- ===== Replika baca-saja (disinkronkan dari service pemilik, JANGAN ditulis oleh service ini) =====")
        for t in reps:
            _, st = tables[t]
            lines.append(rebuild(st, keep_fk=lambda ref: False, drop_columns=SENSITIVE_COLUMNS.get(t, ())))
            lines.append("")
        lines.append("-- ===== Tabel pendukung event (idempotensi + outbox) =====")
        lines.append(MESSAGING_SQL.strip())
        lines.append("")
        lines.append("-- ===== Data awal (seed) =====")
        lines.extend(seed_by_service[svc])
        lines.append("")
        lines.append("SET FOREIGN_KEY_CHECKS = 1;")
        path = os.path.join(OUT_DIR, "%02d-%s.sql" % (10 + n, svc))
        with open(path, "w", encoding="utf-8") as f:
            f.write("\n".join(lines) + "\n")
        summary.append((svc, len(own), len(reps), len(seed_by_service[svc]), path))

    # ---- 00-users.sql : satu user MySQL per service, hanya boleh mengakses database miliknya ----
    names = SERVICES + ["notification"]
    user_lines = ["-- DIHASILKAN oleh scripts/split_db.py. Satu user MySQL per service (isolasi akses)."]
    for svc in names:
        user_lines.append("CREATE USER IF NOT EXISTS '%s_user'@'%%' IDENTIFIED BY '%s_secret';" % (svc, svc))
        user_lines.append("GRANT ALL PRIVILEGES ON `%s_db`.* TO '%s_user'@'%%';" % (svc, svc))
    user_lines.append("FLUSH PRIVILEGES;")
    with open(os.path.join(OUT_DIR, "00-users.sql"), "w", encoding="utf-8") as f:
        f.write("\n".join(user_lines) + "\n")

    # ---- 20-notification.sql : notification-worker hanya butuh ledger idempotensi ----
    notif = [
        "-- DIHASILKAN oleh scripts/split_db.py.",
        "CREATE DATABASE IF NOT EXISTS `notification_db` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;",
        "USE `notification_db`;",
        MESSAGING_SQL.strip().split("CREATE TABLE IF NOT EXISTS `outbox_events`")[0].strip(),
    ]
    with open(os.path.join(OUT_DIR, "20-notification.sql"), "w", encoding="utf-8") as f:
        f.write("\n".join(notif) + "\n")

    print("%-12s %7s %8s %6s" % ("service", "tabel", "replika", "seed"))
    for svc, a, b, c, _ in summary:
        print("%-12s %7d %8d %6d" % (svc, a, b, c))
    print("Total tabel milik sendiri: %d" % sum(s[1] for s in summary))
    if skipped:
        print("Seed untuk tabel tak dikenal dilewati:", sorted(set(skipped)))


if __name__ == "__main__":
    main()