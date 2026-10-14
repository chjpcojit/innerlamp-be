"""Propose structured addresses (province_code, ward_code, below-ward address) for the temples.

Spec: docs/specs/features/temple/ (v1.7+, US-TMP-08, tasks.md Phase 10). This script only
PROPOSES. It never writes assets/data/temple.json: the proposal is reviewed by a person and
only then applied.

    python3 -I scripts/build_temple_addresses.py \
        [assets/data/temple.json] [assets/data/administrative-units.json] \
        [assets/data/temple-address-proposal.json] \
        [docs/specs/features/temple/address-review.md]

How a proposal is made (nothing is guessed silently):
  1. The old free-text `location.address` is split on commas. By convention it reads
     "<street / place...>, <ward>, <district>, <province>": the last segment is the province, the
     one before it the district, the one before that the ward; everything earlier is the part
     below ward level and becomes the proposed new `address`.
  2. province / district / ward are looked up in the dataset's LEGACY tables (the address was
     written under the old two-... three-level scheme), matching by Vietnamese name with
     diacritics and case folded ("TP." == "Thành phố", "TP.HCM" == Hồ Chí Minh).
  3. The legacy ward is mapped to the CURRENT ward(s) with the dataset's ward-level mapping.
  4. The proposed province is the temple's CURRENT `location.province` (already one of the 34
     units) resolved to its code; the proposed ward must belong to that province.

Each temple gets a `status`:
  auto    exactly one current ward, in the temple's province — safe to apply as proposed
  review  anything else; `reasons` says why and `candidates` lists what the dataset offers.
          A person decides `ward_code` (and `address`) for these.
`address_empty` is a separate flag: the old address had nothing below ward level.

The source dataset's data_provenance says it is NOT reconciled against the official
conversion table, so "official" here means "as the dataset has it".
"""
import collections
import json
import re
import sys
import unicodedata

temple_path = sys.argv[1] if len(sys.argv) > 1 else "assets/data/temple.json"
units_path = sys.argv[2] if len(sys.argv) > 2 else "assets/data/administrative-units.json"
out_path = sys.argv[3] if len(sys.argv) > 3 else "assets/data/temple-address-proposal.json"
md_path = sys.argv[4] if len(sys.argv) > 4 else "docs/specs/features/temple/address-review.md"


def strip_block_comments(s):
    """Remove /* ... */ outside string literals (temple.json is JSONC)."""
    out, i, in_str = [], 0, False
    while i < len(s):
        c = s[i]
        if in_str:
            out.append(c)
            if c == "\\":
                out.append(s[i + 1])
                i += 1
            elif c == '"':
                in_str = False
        elif c == '"':
            in_str = True
            out.append(c)
        elif s.startswith("/*", i):
            i = s.index("*/", i) + 1
        else:
            out.append(c)
        i += 1
    return "".join(out)


def fold(s):
    s = unicodedata.normalize("NFD", s.replace("đ", "d").replace("Đ", "D"))
    s = "".join(ch for ch in s if unicodedata.category(ch) != "Mn")
    return re.sub(r"\s+", " ", re.sub(r"[.,]", " ", s.lower())).strip()


def canon_unit(raw):
    """Normalise the abbreviations used in the old addresses before folding."""
    t = raw.strip()
    t = re.sub(r"^TP\.?\s*HCM$", "Hồ Chí Minh", t, flags=re.I)
    t = re.sub(r"^TP\.?\s*", "Thành phố ", t, flags=re.I)
    t = re.sub(r"^(Phường|Xã|Thị trấn)\s+\1\b", r"\1", t, flags=re.I)  # "Phường Phường Đúc"
    return t


PROVINCE_PREFIX = re.compile(r"^(tinh|thanh pho)\s+")
# Names the old addresses use that the dataset's legacy table spells differently.
OLD_PROVINCE_ALIAS = {"thua thien hue": "hue"}
KIND_PREFIX = re.compile(r"^(quan|huyen|thi xa|thanh pho|phuong|xa|thi tran)\s+")


def bare(s):
    return KIND_PREFIX.sub("", fold(s))


def load():
    t = json.loads(strip_block_comments(open(temple_path, encoding="utf-8").read()))
    u = json.load(open(units_path, encoding="utf-8"))
    return t, u


temples, units = load()
new_prov_by_name = {fold(p["name"]): p for p in units["new_provinces"]}
old_prov_by_name = {fold(p["name"]): p for p in units["old_provinces"]}
old_districts = collections.defaultdict(list)
for d in units["old_districts"]:
    old_districts[d["province_code"]].append(d)
old_wards = collections.defaultdict(list)
for w in units["old_wards"]:
    old_wards[w["district_code"]].append(w)
new_ward = {w["code"]: w for w in units["new_wards"]}
mappings = collections.defaultdict(list)
for m in units["ward_mappings"]:
    mappings[m["old_ward_code"]].append(m)


def pick(cands, raw):
    """Match a segment against dataset rows: full name first, then bare name; unique or none."""
    want_full = fold(canon_unit(raw))
    hit = [c for c in cands if fold(c["full_name"]) == want_full]
    if len(hit) == 1:
        return hit[0], []
    want_bare = bare(canon_unit(raw))
    hit = [c for c in cands if bare(c["full_name"]) == want_bare]
    if len(hit) == 1:
        return hit[0], []
    return None, hit if hit else []


proposals = []
for t in temples:
    tid, loc = t["id"], t["location"]
    addr = loc["address"]
    segs = [s.strip() for s in addr.split(",") if s.strip()]
    reasons, below, old = [], "", {}
    cands = []
    p_new = new_prov_by_name.get(fold(loc["province"]))
    if p_new is None:
        reasons.append("current_province_not_in_dataset")
    if len(segs) < 3:
        reasons.append("unparsed_address")
    else:
        prov_raw, dist_raw, ward_raw = segs[-1], segs[-2], segs[-3]
        below = ", ".join(segs[:-3])
        pn = fold(canon_unit(prov_raw))
        pn = PROVINCE_PREFIX.sub("", pn)
        pn = OLD_PROVINCE_ALIAS.get(pn, pn)
        p_old = old_prov_by_name.get(pn)
        if p_old is None:
            reasons.append("old_province_not_found")
        else:
            d_old, _ = pick(old_districts[p_old["code"]], dist_raw)
            w_old, same_name = (None, [])
            if d_old is None:
                reasons.append("old_district_not_found")
            else:
                w_old, same_name = pick(old_wards[d_old["code"]], ward_raw)
                if w_old is None:
                    reasons.append("old_ward_not_found" if not same_name else "old_ward_ambiguous")
            if w_old is None and d_old is None:
                # The district no longer exists under that name (e.g. TP. Huế was split): look the
                # ward up by name in the whole old province. Only a unique match is used and the row
                # stays `review` because the district disagreed. Not done when the district WAS found
                # but lacks the ward: a same-named ward elsewhere in the province is another place.
                in_prov = [w for d in old_districts[p_old["code"]] for w in old_wards[d["code"]]
                           if fold(w["full_name"]) == fold(canon_unit(ward_raw))]
                if len(in_prov) == 1:
                    w_old = in_prov[0]
                    reasons.append("ward_matched_by_name_in_province")
            if w_old is not None:
                dist_name = d_old["full_name"] if d_old else dist_raw
                old = {"province": p_old["full_name"], "district": dist_name,
                       "ward": w_old["full_name"], "ward_code": w_old["code"]}
                for m in mappings.get(w_old["code"], []):
                    nw = new_ward[m["new_ward_code"]]
                    cands.append({"ward_code": nw["code"], "ward": nw["full_name"],
                                  "province_code": nw["province_code"],
                                  "mapping_type": m["mapping_type"], "note": m["note"]})
                if not cands:
                    reasons.append("no_mapping_for_old_ward")
    if not cands and len(segs) >= 3 and p_new is not None:
        # The dataset's legacy tables do not have this ward/district. Offer the current wards of the
        # temple's province that carry the same short name, so a person has something to check.
        # These are suggestions, never auto-applied (mapping_type "name_match").
        wanted = bare(canon_unit(segs[-3]))
        for nw in units["new_wards"]:
            if nw["province_code"] == p_new["code"] and bare(nw["full_name"]) == wanted:
                cands.append({"ward_code": nw["code"], "ward": nw["full_name"],
                              "province_code": nw["province_code"],
                              "mapping_type": "name_match",
                              "note": "same short name in the current scheme; not derived from the dataset mapping"})
        if cands:
            reasons.append("name_match_only")
    if p_new is not None and cands and "name_match_only" not in reasons:
        in_prov = [c for c in cands if c["province_code"] == p_new["code"]]
        if len(in_prov) == 1 and len(cands) == 1:
            pass
        elif len(cands) > 1:
            reasons.append("multiple_new_wards")
        else:
            reasons.append("province_mismatch")
    status = "auto" if not reasons else "review"
    # A review row with exactly one candidate in the right province is pre-filled so the
    # reviewer only has to confirm it; it is still not applied without that confirmation.
    suggested = None
    if status == "review" and p_new is not None:
        one = [c for c in cands if c["province_code"] == p_new["code"]]
        if len(one) == 1 and "multiple_new_wards" not in reasons:
            suggested = one[0]
    proposal = {
        "id": tid, "name": t["name"], "status": status, "reasons": reasons,
        "old_address": addr, "current_province": loc["province"],
        "province_code": p_new["code"] if p_new else None,
        "old_units": old,
        "address": below,
        "address_empty": below == "",
        "ward_code": cands[0]["ward_code"] if status == "auto" else None,
        "ward": cands[0]["ward"] if status == "auto" else None,
        "suggested_ward_code": suggested["ward_code"] if suggested else None,
        "candidates": cands,
    }
    proposals.append(proposal)

summary = collections.Counter(p["status"] for p in proposals)
reason_count = collections.Counter(r for p in proposals for r in p["reasons"])
doc = {
    "meta": {
        "temples": len(proposals),
        "status": dict(summary),
        "reasons": dict(reason_count),
        "address_empty": sum(1 for p in proposals if p["address_empty"]),
        "dataset_version": units["meta"].get("version"),
        "note": "PROPOSAL ONLY. A person reviews `review` rows and sets ward_code/address; "
                "nothing here has been written to assets/data/temple.json.",
    },
    "proposals": proposals,
}
with open(out_path, "w", encoding="utf-8") as f:
    json.dump(doc, f, ensure_ascii=False, indent=2)
    f.write("\n")

def md_row(p):
    c = "; ".join(f"`{x['ward_code']}` {x['ward']} ({x['mapping_type']})" for x in p["candidates"]) or "—"
    sug = f"`{p['suggested_ward_code']}`" if p["suggested_ward_code"] else "—"
    return (f"| `{p['id']}` | {p['name']} | {p['old_address']} | {', '.join(p['reasons'])} | "
            f"{c} | {sug} | {p['address'] or '*(trống)*'} |")


review = [p for p in proposals if p["status"] == "review"]
auto = [p for p in proposals if p["status"] == "auto"]
lines = [
    "# Đề xuất địa chỉ có cấu trúc cho chùa (cần duyệt)",
    "",
    "Sinh bởi `scripts/build_temple_addresses.py` — **chỉ là đề xuất**, chưa ghi vào `assets/data/temple.json`.",
    f"Dataset hành chính: `{units['meta'].get('version')}` (ghi `data_provenance`: chưa đối chiếu với file chính thức).",
    "Spec: `docs/specs/features/temple/` v1.7+, `tasks.md` Phase 10.",
    "",
    f"- Tổng: **{len(proposals)}** chùa; `auto`: **{len(auto)}** (đúng một xã mới, đúng tỉnh); `review`: **{len(review)}**.",
    f"- Địa chỉ dưới cấp xã **trống**: **{doc['meta']['address_empty']}** chùa (địa chỉ cũ chỉ có xã/huyện/tỉnh).",
    "",
    "## Cần duyệt",
    "",
    "Với mỗi dòng: chọn `ward_code` trong cột *Ứng viên* (hoặc cho mã khác), và sửa cột *address* nếu cần.",
    "`Gợi ý` có giá trị khi chỉ còn một ứng viên nhưng dòng vẫn cần xác nhận vì lý do ở cột *Lý do*.",
    "",
    "| id | tên | địa chỉ cũ | lý do | ứng viên (xã mới) | gợi ý | address đề xuất |",
    "|---|---|---|---|---|---|---|",
] + [md_row(p) for p in review] + [
    "",
    "## Tự động (một xã mới, đúng tỉnh) — xem lướt",
    "",
    "| id | tên | địa chỉ cũ | tỉnh | xã mới | address đề xuất |",
    "|---|---|---|---|---|---|",
] + [f"| `{p['id']}` | {p['name']} | {p['old_address']} | `{p['province_code']}` {p['current_province']} | "
     f"`{p['ward_code']}` {p['ward']} | {p['address'] or '*(trống)*'} |" for p in auto] + [""]
with open(md_path, "w", encoding="utf-8") as f:
    f.write("\n".join(lines))
print(json.dumps(doc["meta"], ensure_ascii=False, indent=2))
print("wrote", out_path, "and", md_path)
