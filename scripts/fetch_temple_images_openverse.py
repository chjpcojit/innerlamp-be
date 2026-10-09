#!/usr/bin/env python3
"""Second pass of fetch_temple_images.py: fill ids listed in
assets/images/temples/missing.json from Openverse (CC images from Flickr,
Wikimedia, ...), commercial-use + modification-allowed licenses only.

A candidate is accepted only if its title contains the full normalized temple
name (and the "(qualifier)" of the name, if any). Exterior shots are preferred
over statues/altars. Output uses the same layout/credits.json as the first pass.
Requires macOS `sips` for resizing. Run from repo root.
"""
import json, re, subprocess, sys, time, tempfile, unicodedata, urllib.parse, urllib.request
from pathlib import Path

UA = {"User-Agent": "innerlamp-seed/1.0 (duc@wee.vn)"}
ROOT = Path(__file__).resolve().parent.parent
OUT = ROOT / "assets/images/temples"
GOOD = ["tam quan", "cong", "toan canh", "mat tien", "thap", "panorama", "gate", "view", "exterior", "chanh dien", "main hall"]
BAD = ["tuong", "lu huong", "ban tho", "bia", "chuong", "noi that", "statue", "altar", "hoa ", "den long", "1969", "1971", "photo by", "nam 19", "bat nha", "bo tat", "vang", "mon dong"]


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


def search(q):
    u = "https://api.openverse.org/v1/images/?" + urllib.parse.urlencode(
        {"q": q, "license_type": "commercial,modification", "page_size": 20, "extension": "jpg"})
    return fetch(u).get("results", [])


def score(r):
    t = " " + norm(r["title"]) + " "
    return sum(2 for g in GOOD if f" {g}" in t) - sum(3 for b in BAD if b in t) + min((r.get("width") or 0), 3000) / 1500


def pick(t, used):
    name = re.sub(r"\(.*?\)", "", t["name"]).strip()
    qual = re.search(r"\((.*?)\)", t["name"])
    core = norm(name)
    cands = []
    for r in search(name):
        title = norm(r["title"])
        tags = " ".join(norm(x.get("name", "")) for x in r.get("tags") or [])
        if core not in title or (r.get("width") or 0) < 800 or r["url"] in used:
            continue
        if qual and norm(qual.group(1)) not in title + " " + tags:
            continue
        cands.append(r)
    return max(cands, key=score) if cands else None


def resize(src, dest, w):
    subprocess.run(["sips", "-s", "format", "jpeg", "-s", "formatOptions", "82", "--resampleWidth", str(w), str(src), "--out", str(dest)],
                   check=True, capture_output=True)


def main():
    text = re.sub(r"/\*.*?\*/", "", (ROOT / "assets/data/temple.json").read_text(), flags=re.S)
    T = {t["id"]: t for t in json.loads(text)}
    cf = OUT / "credits.json"
    credits = json.loads(cf.read_text())
    used = {v.get("image_url") for v in credits.values()}
    missing = json.loads((OUT / "missing.json").read_text())
    still = []
    for i in missing:
        if i in credits:
            continue
        try:
            r = pick(T[i], used)
            if not r:
                still.append(i); print("MISS", i, T[i]["name"], flush=True); continue
            with tempfile.NamedTemporaryFile(suffix=".jpg") as tmp:
                Path(tmp.name).write_bytes(fetch(r["url"], binary=True))
                resize(tmp.name, OUT / "full" / f"{i}.jpg", 1200)
                resize(tmp.name, OUT / "thumb" / f"{i}.jpg", 400)
            used.add(r["url"])
            credits[i] = {"name": T[i]["name"], "file": r["title"], "source": r.get("foreign_landing_url"),
                          "image_url": r["url"], "author": r.get("creator") or "", "license": f'{r["license"].upper()} {r.get("license_version") or ""}'.strip(),
                          "provider": r.get("source")}
            cf.write_text(json.dumps(credits, ensure_ascii=False, indent=1))
            print("OK  ", i, T[i]["name"], "<-", r["title"][:60], flush=True)
        except Exception as e:
            still.append(i); print("ERR ", i, T[i]["name"], e, flush=True)
        time.sleep(1.2)
    (OUT / "missing.json").write_text(json.dumps(still, indent=1))
    print(f"done: {len(credits)} ok, {len(still)} missing", flush=True)


if __name__ == "__main__":
    main()
