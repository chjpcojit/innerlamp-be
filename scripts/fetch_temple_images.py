#!/usr/bin/env python3
"""Download one image per temple in assets/data/temple.json from Wikipedia/Commons.

Output (named by temple id so it links to temple.json):
  assets/images/temples/full/<id>.jpg   (width 1200, detail page)
  assets/images/temples/thumb/<id>.jpg  (width 400, list/home card)
  assets/images/temples/credits.json    (source page, author, license per id)
Resumable: skips ids that already have both files. Run from repo root.
"""
import json, re, sys, time, unicodedata, urllib.parse, urllib.request
from pathlib import Path

UA = "innerlamp-seed/1.0 (duc@wee.vn)"
ROOT = Path(__file__).resolve().parent.parent
SRC = ROOT / "assets/data/temple.json"
OUT = ROOT / "assets/images/temples"
EXT = {"image/jpeg", "image/png"}


def get(url, binary=False, retries=4):
    for i in range(retries):
        try:
            req = urllib.request.Request(url, headers={"User-Agent": UA})
            with urllib.request.urlopen(req, timeout=40) as r:
                b = r.read()
                return b if binary else json.loads(b)
        except Exception as e:
            if i == retries - 1:
                raise
            time.sleep(3 * (i + 1))


def api(host, **p):
    p.update(format="json", action="query")
    return get(f"https://{host}/w/api.php?" + urllib.parse.urlencode(p))


def norm(s):
    s = unicodedata.normalize("NFD", s.lower().replace("đ", "d"))
    s = "".join(c for c in s if unicodedata.category(c) != "Mn")
    return re.sub(r"[^a-z0-9 ]+", " ", s).split()


def core(name):
    n = re.sub(r"\(.*?\)", "", name).strip()
    return n


def keys(name):
    toks = norm(core(name))
    return [t for t in toks if t not in {"chua", "tinh", "xa", "thien", "vien", "tu", "tho", "am", "ni", "tu", "vien"}] or toks


def viwiki_image(t):
    names = [core(t["name"])] + [a for a in t.get("aliases", []) if len(a) > 4][:2]
    want = {" ".join(norm(n)) for n in names}
    for q in names[:2]:
        r = api("vi.wikipedia.org", generator="search", gsrsearch=q, gsrlimit=5,
                prop="pageimages", piprop="name|original")
        for p in sorted(r.get("query", {}).get("pages", {}).values(), key=lambda p: p["index"]):
            if " ".join(norm(re.sub(r"\(.*?\)", "", p["title"]))) in want and p.get("pageimage"):
                return "File:" + p["pageimage"].replace("_", " "), "vi.wikipedia.org/wiki/" + urllib.parse.quote(p["title"].replace(" ", "_"))
    return None


def commons_image(t):
    k = keys(t["name"])
    r = api("commons.wikimedia.org", generator="search", gsrsearch=core(t["name"]) + " chùa",
            gsrnamespace=6, gsrlimit=8, prop="imageinfo", iiprop="mime|size")
    for p in sorted(r.get("query", {}).get("pages", {}).values(), key=lambda p: p["index"]):
        ii = (p.get("imageinfo") or [{}])[0]
        if ii.get("mime") in EXT and ii.get("width", 0) >= 800 and all(x in " ".join(norm(p["title"])) for x in k):
            return p["title"], "commons.wikimedia.org/wiki/" + urllib.parse.quote(p["title"].replace(" ", "_"))
    return None


def meta(title):
    r = api("commons.wikimedia.org", titles=title, prop="imageinfo", iiprop="mime|extmetadata")
    ii = list(r["query"]["pages"].values())[0].get("imageinfo", [{}])[0]
    em = ii.get("extmetadata", {})
    strip = lambda s: re.sub(r"<.*?>", "", s or "").strip()
    return ii.get("mime"), strip(em.get("Artist", {}).get("value")), strip(em.get("LicenseShortName", {}).get("value"))


def download(title, w, dest):
    from_name = urllib.parse.quote(title.split(":", 1)[1])
    data = get(f"https://commons.wikimedia.org/wiki/Special:FilePath/{from_name}?width={w}", binary=True)
    if data[:3] != b"\xff\xd8\xff" and data[:4] != b"\x89PNG":
        raise ValueError("not an image")
    dest.write_bytes(data)


def main():
    src = re.sub(r"/\*.*?\*/", "", SRC.read_text(), flags=re.S)
    temples = json.loads(src)
    (OUT / "full").mkdir(parents=True, exist_ok=True)
    (OUT / "thumb").mkdir(parents=True, exist_ok=True)
    cf = OUT / "credits.json"
    credits = json.loads(cf.read_text()) if cf.exists() else {}
    missing = []
    for t in temples:
        i = t["id"]
        if (OUT / "full" / f"{i}.jpg").exists() and (OUT / "thumb" / f"{i}.jpg").exists() and i in credits:
            continue
        try:
            hit = viwiki_image(t) or commons_image(t)
            if not hit:
                missing.append(i); print("MISS", i, t["name"]); continue
            title, page = hit
            mime, artist, lic = meta(title)
            if mime not in EXT:
                missing.append(i); print("BADMIME", i, title); continue
            download(title, 1200, OUT / "full" / f"{i}.jpg")
            time.sleep(1.5)
            download(title, 400, OUT / "thumb" / f"{i}.jpg")
            credits[i] = {"name": t["name"], "file": title, "source": "https://" + page, "author": artist, "license": lic}
            cf.write_text(json.dumps(credits, ensure_ascii=False, indent=1))
            print("OK  ", i, t["name"], "<-", title)
        except Exception as e:
            missing.append(i); print("ERR ", i, t["name"], e)
        time.sleep(1.5)
    (OUT / "missing.json").write_text(json.dumps(missing, indent=1))
    print(f"done: {len(credits)} ok, {len(missing)} missing")


if __name__ == "__main__":
    main()
