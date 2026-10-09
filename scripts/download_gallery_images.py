#!/usr/bin/env python3
"""Download the gallery photos locally and point assets/data/buddhist_gallery.json at them.

Dev setup: appService serves assets/images at /static/images/ (server.static_images_dir), so
the gallery is served from the file system on this machine's LAN IP, like temples/deities.

  assets/images/gallery/full/<id>.jpg    (1280px, detail page)
  assets/images/gallery/thumb/<id>.jpg   (500px, list/home card)
  assets/images/gallery/credits.json     (author, license, source per id)

First run keeps the Wikimedia URLs in remote_image_url / remote_thumbnail_url, so the script is
re-runnable and `make lan-urls` can rewrite only the host of image_url / thumbnail_url.
Skips files already on disk. Usage (from anywhere):

  python3 scripts/download_gallery_images.py [--host http://192.168.1.186:8080]
"""
import argparse, json, re, sys, time, urllib.error, urllib.request
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
DATA = ROOT / "assets/data/buddhist_gallery.json"
OUT = ROOT / "assets/images/gallery"
UA = "innerlamp-seed/1.0 (duc@wee.vn)"
FULL_PX = 1280  # standard Wikimedia thumbnail step; the source has 1920


def fetch(url: str, dest: Path) -> None:
    for attempt in range(6):
        try:
            req = urllib.request.Request(url, headers={"User-Agent": UA})
            with urllib.request.urlopen(req, timeout=60) as r:
                data = r.read()
            if len(data) < 2000:
                raise ValueError(f"suspiciously small ({len(data)} bytes)")
            dest.parent.mkdir(parents=True, exist_ok=True)
            dest.write_bytes(data)
            return
        except urllib.error.HTTPError as e:
            if e.code in (429, 503):
                time.sleep(5 * (attempt + 1))
                continue
            raise
    raise RuntimeError(f"gave up on {url}")


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--host", default=None, help="image host, e.g. http://192.168.1.186:8080 (default: host already used by assets/data/temple.json)")
    args = ap.parse_args()
    host = args.host
    if not host:
        m = re.search(r'"(http://[^/"]+)/static/images/', (ROOT / "assets/data/temple.json").read_text())
        if not m:
            sys.exit("no host found in temple.json; pass --host")
        host = m.group(1)

    items = json.loads(DATA.read_text())
    credits, failed = [], []
    for i, e in enumerate(items, 1):
        remote_full = e.get("remote_image_url") or e["image_url"]
        remote_thumb = e.get("remote_thumbnail_url") or e["thumbnail_url"]
        if not remote_full.startswith("https://"):
            sys.exit(f"{e['id']}: no remote URL to download from")
        remote_full = re.sub(r"/\d+px-", f"/{FULL_PX}px-", remote_full)
        for url, sub in ((remote_full, "full"), (remote_thumb, "thumb")):
            dest = OUT / sub / f"{e['id']}.jpg"
            if dest.exists():
                continue
            try:
                fetch(url, dest)
                time.sleep(1)
            except Exception as ex:  # noqa: BLE001 — report and continue, never half-rewrite the JSON
                failed.append((e["id"], sub, str(ex)))
        e["remote_image_url"], e["remote_thumbnail_url"] = e.get("remote_image_url") or e["image_url"], remote_thumb
        e["image_url"] = f"{host}/static/images/gallery/full/{e['id']}.jpg"
        e["thumbnail_url"] = f"{host}/static/images/gallery/thumb/{e['id']}.jpg"
        credits.append({k: e[k] for k in ("id", "title", "author", "license", "source_url")})
        print(f"[{i}/{len(items)}] {e['id']}", flush=True)

    if failed:
        for f in failed:
            print("FAILED", *f, file=sys.stderr)
        sys.exit(f"{len(failed)} download(s) failed; JSON left untouched, re-run to retry")
    DATA.write_text(json.dumps(items, ensure_ascii=False, indent=2) + "\n")
    (OUT / "credits.json").write_text(json.dumps(credits, ensure_ascii=False, indent=2) + "\n")
    print(f"done: {len(items)} images in {OUT}, JSON rewritten with host {host}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
