#!/usr/bin/env python3
"""Third pass: fill ids in assets/images/temples/missing.json using
 (a) Commons files geotagged within RADIUS metres of the temple's lat/lng
 (b) Openverse + Commons search by prefix-stripped name and aliases.
Provenance is stored in credits.json ("method": geo|name) so geo hits can be
reviewed. Same layout as the other passes. Needs macOS `sips`. Run from repo root.
"""
import json, re, subprocess, time, tempfile, unicodedata, urllib.parse, urllib.request, math
from pathlib import Path

UA = {"User-Agent": "innerlamp-seed/1.0 (duc@wee.vn)"}
ROOT = Path(__file__).resolve().parent.parent
OUT = ROOT / "assets/images/temples"
RADIUS = 400
PREFIX = {"chua", "tinh", "xa", "thien", "vien", "that", "to", "dinh", "phap", "tu", "am", "ni", "tu vien", "linh", "son"}
GOOD = ["tam quan", "cong", "toan canh", "mat tien", "thap", "panorama", "gate", "view", "exterior", "chanh dien", "main hall", "pagoda", "temple"]
BAD = ["tuong", "lu huong", "ban tho", "bia", "chuong", "noi that", "statue", "altar", "hoa ", "den long", "photo by", "bat nha", "bo tat", "map", "logo", "church", "nha tho", "cay "]


def norm(s):
    s = unicodedata.normalize("NFD", s.lower().replace("đ", "d"))
    s = "".join(c for c in s if unicodedata.category(c) != "Mn")
    return " ".join(re.sub(r"[^a-z0-9 ]+", " ", s).split())


def fetch(url, binary=False, tries=3):
    for i in range(tries):
        try:
            with urllib.request.urlopen(urllib.request.Request(url, headers=UA), timeout=60) as r:
                b = r.read()
            return b if binary else json.loads(b)
        except Exception:
            if i == tries - 1:
                raise
            time.sleep(4 * (i + 1))


def capi(**p):
    p.update(format="json", action="query")
    return fetch("https://commons.wikimedia.org/w/api.php?" + urllib.parse.urlencode(p))


def strip_core(name):
    n = norm(re.sub(r"\(.*?\)", "", name))
    toks = n.split()
    while toks and toks[0] in {"chua", "tinh", "xa", "thien", "vien", "thien", "thanh", "that", "phap", "to", "dinh", "ni", "tu"}:
        toks.pop(0)
    return " ".join(toks)


def info(titles):
    out = {}
    for k in range(0, len(titles), 25):
        r = capi(titles="|".join(titles[k:k + 25]), prop="imageinfo", iiprop="mime|size|extmetadata")
        for p in r.get("query", {}).get("pages", {}).values():
            ii = (p.get("imageinfo") or [{}])[0]
            if ii.get("mime") in ("image/jpeg", "image/png") and ii.get("width", 0) >= 900:
                em = ii.get("extmetadata", {})
                st = lambda s: re.sub(r"<.*?>", "", s or "").strip()
                out[p["title"]] = {"w": ii["width"], "author": st(em.get("Artist", {}).get("value")),
                                   "license": st(em.get("LicenseShortName", {}).get("value"))}
    return out


def score(title, w):
    t = " " + norm(title) + " "
    return sum(2 for g in GOOD if f" {g}" in t) - sum(3 for b in BAD if b in t) + min(w, 3000) / 1500


def candidates(t, used):
    core_names = {strip_core(t["name"])} | {strip_core(a) for a in t.get("aliases", [])}
    core_names = {c for c in core_names if len(c) >= 4}
    qual = re.search(r"\((.*?)\)", t["name"])
    cands = []
    # (a) geo
    g = capi(list="geosearch", gscoord=f'{t["location"]["lat"]}|{t["location"]["lng"]}', gsradius=RADIUS, gsnamespace=6, gslimit=30)
    geo = g.get("query", {}).get("geosearch", [])
    inf = info([x["title"] for x in geo]) if geo else {}
    for x in geo:
        if x["title"] in inf and x["title"] not in used:
            nt = norm(x["title"])
            named = any(c in nt for c in core_names)
            cands.append((score(x["title"], inf[x["title"]]["w"]) + (4 if named else 0) - x["dist"] / 400, "geo", x["title"], inf[x["title"]], x["dist"], named))
    # (b) commons text search
    for q in list(core_names)[:3]:
        r = capi(generator="search", gsrsearch=q + " chùa", gsrnamespace=6, gsrlimit=10)
        titles = [p["title"] for p in r.get("query", {}).get("pages", {}).values()]
        inf2 = info(titles) if titles else {}
        for ti, v in inf2.items():
            if ti not in used and any(c in norm(ti) for c in core_names) and (not qual or norm(qual.group(1)) in norm(ti)):
                cands.append((score(ti, v["w"]) + 4, "name", ti, v, -1, True))
    return sorted(cands, key=lambda c: -c[0])


def resize(src, dest, w):
    subprocess.run(["sips", "-s", "format", "jpeg", "-s", "formatOptions", "82", "--resampleWidth", str(w), str(src), "--out", str(dest)], check=True, capture_output=True)


def main():
    text = re.sub(r"/\*.*?\*/", "", (ROOT / "assets/data/temple.json").read_text(), flags=re.S)
    T = {t["id"]: t for t in json.loads(text)}
    cf = OUT / "credits.json"
    credits = json.loads(cf.read_text())
    used = {v["file"] for v in credits.values()}
    still = []
    for i in json.loads((OUT / "missing.json").read_text()):
        if i in credits:
            continue
        try:
            cs = candidates(T[i], used)
            # geo-only (name not in title) hits need to be near AND decent; the rest is reviewed by eye
            cs = [c for c in cs if c[5] or (c[1] == "geo" and c[4] <= 150 and c[0] > 1.5)]
            if not cs:
                still.append(i); print("MISS", i, T[i]["name"], flush=True); continue
            sc, method, title, v, dist, named = cs[0]
            fn = urllib.parse.quote(title.split(":", 1)[1])
            data = fetch(f"https://commons.wikimedia.org/wiki/Special:FilePath/{fn}?width=1600", binary=True)
            with tempfile.NamedTemporaryFile(suffix=".img") as tmp:
                Path(tmp.name).write_bytes(data)
                resize(tmp.name, OUT / "full" / f"{i}.jpg", 1200)
                resize(tmp.name, OUT / "thumb" / f"{i}.jpg", 400)
            used.add(title)
            credits[i] = {"name": T[i]["name"], "file": title, "source": "https://commons.wikimedia.org/wiki/" + urllib.parse.quote(title.replace(" ", "_")),
                          "author": v["author"], "license": v["license"], "method": method, "dist_m": dist, "name_match": named}
            cf.write_text(json.dumps(credits, ensure_ascii=False, indent=1))
            print("OK  ", i, T[i]["name"], f"[{method} {dist}m]", "<-", title[:60], flush=True)
        except Exception as e:
            still.append(i); print("ERR ", i, T[i]["name"], e, flush=True)
        time.sleep(1.2)
    (OUT / "missing.json").write_text(json.dumps(still, indent=1))
    print(f"done: {len(credits)} ok, {len(still)} missing", flush=True)


if __name__ == "__main__":
    main()
