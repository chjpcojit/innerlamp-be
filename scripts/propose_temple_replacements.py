#!/usr/bin/env python3
"""Fourth pass: propose replacement temples for ids in assets/images/temples/missing.json
(no free image exists for them). Candidates come from vi.wikipedia "Thể loại:Chùa Việt Nam"
whose lead image is a real photo on Commons. Facts are taken ONLY from the Wikipedia
article / OSM geocoding; unverifiable fields (abbot, hours, amenities) stay empty/false.

Writes scripts/__proposals.json (review before running apply_temple_replacements.py).
Run from repo root.
"""
import json, re, time, unicodedata, urllib.parse, urllib.request
from pathlib import Path

UA = {"User-Agent": "innerlamp-seed/1.0 (duc@wee.vn)"}
ROOT = Path(__file__).resolve().parent.parent
OUT = ROOT / "assets/images/temples"
BAD = ["tuong", "statue", "map", "logo", "bia ", "lu huong", "ban tho", "noi that", "bouddha", "phat ", "nam 19", "relief", "bang do", "truong hoc", "le hoi", "buddhist temple outside"]


def get(url, tries=3):
    for i in range(tries):
        try:
            return json.load(urllib.request.urlopen(urllib.request.Request(url, headers=UA), timeout=40))
        except Exception:
            time.sleep(3 * (i + 1))


def wapi(host, **p):
    p.update(format="json", action="query")
    return get(f"https://{host}/w/api.php?" + urllib.parse.urlencode(p))


def norm(s):
    s = unicodedata.normalize("NFD", s.lower().replace("đ", "d"))
    s = "".join(c for c in s if unicodedata.category(c) != "Mn")
    return " ".join(re.sub(r"[^a-z0-9 ]+", " ", re.sub(r"\(.*?\)", "", s)).split())


def region_of(lat):
    return "north" if lat >= 20.0 else "central" if lat >= 11.6 else "south"


def rank_of(text):
    t = text.lower()
    if "quốc gia đặc biệt" in t: return "DI_TICH_QUOC_GIA_DAC_BIET"
    if "di tích quốc gia" in t or "di tích lịch sử - văn hóa quốc gia" in t or "di tích kiến trúc nghệ thuật quốc gia" in t: return "DI_TICH_QUOC_GIA"
    if "danh lam thắng cảnh" in t: return "DANH_LAM_THANG_CANH"
    if "cấp tỉnh" in t: return "DI_TICH_CAP_TINH"
    if "cấp thành phố" in t: return "DI_TICH_CAP_THANH_PHO"
    return "CHUA_XEP_HANG"


def nominatim(q):
    r = get("https://nominatim.openstreetmap.org/search?" + urllib.parse.urlencode(
        dict(q=q, format="jsonv2", countrycodes="vn", limit=3, addressdetails=1)))
    time.sleep(1.2)
    return r or []


def main():
    T = json.loads(re.sub(r"/\*.*?\*/", "", (ROOT / "assets/data/temple.json").read_text(), flags=re.S))
    have = {norm(t["name"]) for t in T} | {norm(a) for t in T for a in t.get("aliases", [])}
    keep_names = {t["id"]: t["name"] for t in T}
    missing = json.loads((OUT / "missing.json").read_text())
    used_files = {v["file"] for v in json.loads((OUT / "credits.json").read_text()).values()}
    titles = [t for t in json.loads((ROOT / "scripts/__vipages.json").read_text())["pages"]
              if re.match(r"(Chùa|Thiền viện|Tịnh xá) ", t) and norm(t) not in have and t != "Chùa Việt Nam"]
    props = []
    for k in range(0, len(titles), 20):
        r = wapi("vi.wikipedia.org", titles="|".join(titles[k:k + 20]), prop="pageimages|coordinates|extracts",
                 piprop="name", pilimit=50, explaintext=1, exintro=1, exlimit=20, colimit=20)
        for p in r["query"]["pages"].values():
            if not p.get("pageimage") or "extract" not in p or "có thể là" in p["extract"][:200]:
                continue
            props.append(p)
    out = []
    for p in props:
        img = "File:" + p["pageimage"].replace("_", " ")
        if img in used_files or not re.search(r"\.(jpe?g|png)$", img, re.I) or any(b in norm(img) + " " for b in BAD):
            continue
        ci = wapi("commons.wikimedia.org", titles=img, prop="imageinfo", iiprop="mime|size|extmetadata")
        ii = (list(ci["query"]["pages"].values())[0].get("imageinfo") or [None])[0]
        if not ii or ii["width"] < 800 or ii["mime"] not in ("image/jpeg", "image/png"):
            continue
        em = ii["extmetadata"]; st = lambda s: re.sub(r"<.*?>", "", s or "").strip()
        intro = p["extract"].strip()
        full = wapi("vi.wikipedia.org", pageids=p["pageid"], prop="extracts", explaintext=1)["query"]["pages"][str(p["pageid"])].get("extract", intro)
        ext = full
        year = None
        m = re.search(r"(?:xây dựng|thành lập|khởi dựng|lập|dựng|sáng lập|ra đời|khai sơn|kiến tạo)[^.]{0,60}?(?:năm|vào)\s+(\d{3,4})", intro) or re.search(r"năm\s+(1\d{3}|[5-9]\d{2}|20[0-2]\d)", intro)
        if m: year = int(m.group(1))
        if not year: continue
        name = p["title"]
        q = re.sub(r"\((.*?)\)", r"\1", name)
        lat = lon = None; addr = ""
        if p.get("coordinates"):
            lat, lon = p["coordinates"][0]["lat"], p["coordinates"][0]["lon"]
        hits = nominatim(q + ", Việt Nam") or nominatim(name.split("(")[0].strip() + ", Việt Nam")
        if hits:
            a = hits[0]["address"]
            if lat is None: lat, lon = float(hits[0]["lat"]), float(hits[0]["lon"])
            addr = ", ".join(x for x in [a.get("road"), a.get("suburb") or a.get("quarter") or a.get("village") or a.get("hamlet"),
                                         a.get("city_district") or a.get("county") or a.get("city") or a.get("town"), a.get("state") or a.get("city")] if x)
        if lat is None: continue
        typ = "THIEN_VIEN" if name.startswith("Thiền viện") else "TINH_XA" if name.startswith("Tịnh xá") else "CHUA"
        sect = "KHAT_SI" if typ == "TINH_XA" else "TRUC_LAM_YEN_TU" if "Trúc Lâm" in name else \
               "NAM_TONG_KHMER" if re.search(r"Khmer|Nam tông", intro) else "BẮC_TÔNG"
        arch = ""
        sm = re.search(r"\n== (?:Kiến trúc|Công trình|Cấu trúc|Quy mô)[^=]*==\n(.*?)(?=\n==|\Z)", ext, re.S)
        if sm: arch = sm.group(1).strip()[:900]
        sents = re.split(r"(?<=[.!?])\s+", intro)
        out.append({"name": name, "wiki": "https://vi.wikipedia.org/wiki/" + urllib.parse.quote(name.replace(" ", "_")),
                    "region": region_of(lat), "type": typ, "sect": sect, "year": year, "rank": rank_of(ext),
                    "lat": round(lat, 5), "lng": round(lon, 5), "address": addr,
                    "introduction": " ".join(sents[:4])[:900], "architecture": arch,
                    "image": img, "image_author": st(em.get("Artist", {}).get("value")), "image_license": st(em.get("LicenseShortName", {}).get("value")),
                    "geo_src": "wikipedia" if p.get("coordinates") else "osm-nominatim", "osm_name": hits[0]["display_name"][:120] if hits else ""})
        print("OK", name, region_of(lat), year, flush=True)
    (ROOT / "scripts/__proposals.json").write_text(json.dumps(out, ensure_ascii=False, indent=1))
    from collections import Counter
    need = Counter(i.split("_")[1] for i in missing)
    print("proposals", len(out), Counter(o["region"] for o in out), "need", need)


if __name__ == "__main__":
    main()
