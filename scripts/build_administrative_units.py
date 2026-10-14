"""Build assets/data/administrative-units.json from the vietnam-address-kit dataset.

The data file is what `go run ./services/appService/cmd/seedadminunits` loads. This
script is how that file was produced, kept so it can be rebuilt when a new resolution
changes the units or the upstream dataset is corrected.

    git clone https://github.com/zindont/vietnam-address-kit /tmp/vak
    (cd /tmp/vak && git rev-parse HEAD && git log -1 --format=%cs)      # commit, commit date
    python3 -I scripts/build_administrative_units.py \
        /tmp/vak/src/data/official assets/data/administrative-units.json <commit> <commit_date>
    go run ./services/appService/cmd/seedadminunits -dry-run             # verify before loading

What it does, and nothing silently:
  * keeps the dataset's six files as the old (legacy) and new (current) schemes plus the
    ward-level mapping, and computes `sort_order` (Vietnamese alphabetical rank of the short
    name inside its parent, digits compared as numbers) so the database collation is irrelevant;
  * CORRECTS wards whose province disagrees with their district's province (a labelling defect
    in the source: "Phường 4" of Thành phố Tân An labelled Tiền Giang, "Xã Tiên Hải" of Thành phố
    Hà Tiên labelled An Giang) and drops the duplicate district rows they cause — every such edit
    is written to meta.corrections;
  * leaves out a district-level mapping for a district that has wards ("Huyện Kim Sơn") and
    records it in meta.unresolved;
  * refuses to write the file if its own consistency checks fail (the same rules as
    domain.Dataset.Verify).

The source dataset is MIT-licensed code; its data is derived from official Vietnamese
administrative records and has NOT been reconciled against the official conversion table.
usage: python3 -I build_administrative_units.py <official_dir> <out_file> <source_commit> <source_commit_date>
"""
import collections
import json
import os
import sys
import unicodedata

src, out, commit, commit_date = sys.argv[1:5]
L = lambda n: json.load(open(os.path.join(src, n + ".json"), encoding="utf-8"))
cp, cw, lp, ld, lw, mp = (L(n) for n in [
    "current-provinces", "current-wards", "legacy-provinces", "legacy-districts",
    "legacy-wards", "mappings"])

# ---- Vietnamese alphabetical sort key (DB-collation independent) -------------------
ALPHABET = "aăâbcdđeêghiklmnoôơpqrstuưvxy"
IDX = {c: i + 1 for i, c in enumerate(ALPHABET)}
TONE = {"̀": 1, "̉": 2, "̃": 3, "́": 4, "̣": 5}  # huyền hỏi ngã sắc nặng


def viet_key(s):
    s = unicodedata.normalize("NFD", s.lower())
    primary, tones, i = [], [], 0
    while i < len(s):
        ch = s[i]
        i += 1
        if ch.isdigit():
            j = i
            while j < len(s) and s[j].isdigit():
                j += 1
            primary.append((0, int(s[i - 1:j]), 0))
            i = j
            continue
        mods = []
        while i < len(s) and unicodedata.combining(s[i]):
            mods.append(s[i])
            i += 1
        letter, tone = ch, 0
        for m in mods:
            if m in TONE:
                tone = TONE[m]
            elif m == "̆" and ch == "a":
                letter = "ă"
            elif m == "̂" and ch in "aeo":
                letter = {"a": "â", "e": "ê", "o": "ô"}[ch]
            elif m == "̛" and ch in "ou":
                letter = {"o": "ơ", "u": "ư"}[ch]
        if ch == " ":
            primary.append((-1, 0, 0))
        else:
            primary.append((1, IDX.get(letter, 100 + ord(letter)), 0))
        tones.append(tone)
    return (primary, tones)


def ranked(items, scope, name="name"):
    """sort_order: 1-based Vietnamese alphabetical rank of `name` inside each scope."""
    groups = collections.defaultdict(list)
    for it in items:
        groups[scope(it)].append(it)
    for g in groups.values():
        g.sort(key=lambda x: (viet_key(x[name]), x["code"]))
        for n, it in enumerate(g, 1):
            it["sort_order"] = n


def unit(x, **extra):
    d = {"code": x["code"], "name": x["name"], "full_name": x["nameWithType"], "type": x["type"]}
    d.update(extra)
    return d


# ---- corrections: wards whose province disagrees with their district's province ----
district_province = {}
for x in ld:
    district_province.setdefault(x["code"], []).append(x["provinceCode"])
corrections = []
fixed_ward_province = {}
# the district that really owns each ward = the province holding the majority of that district's wards
by_district = collections.defaultdict(collections.Counter)
for w in lw:
    by_district[w["districtCode"]][w["provinceCode"]] += 1
spurious_districts = set()
for dcode, provs in district_province.items():
    if len(provs) > 1:
        owner = by_district[dcode].most_common(1)[0][0]
        for w in lw:
            if w["districtCode"] == dcode and w["provinceCode"] != owner:
                fixed_ward_province[w["code"]] = owner
                corrections.append({
                    "kind": "ward_province_label",
                    "unit": f'{w["nameWithType"]} (old ward {w["code"]}, district {dcode})',
                    "was": f'province {w["provinceCode"]}', "now": f"province {owner}",
                    "reason": "ward carried a province that disagrees with its own district's "
                              "province; all other wards of that district carry the owning province",
                })
        for p in provs:
            if p != owner:
                spurious_districts.add((dcode, p))

# ---- build tables -----------------------------------------------------------------
new_provinces = [unit(p) for p in cp]
new_wards = [unit(w, province_code=w["provinceCode"]) for w in cw]
old_provinces = [unit(p) for p in lp]
old_districts = [unit(x, province_code=x["provinceCode"]) for x in ld
                 if (x["code"], x["provinceCode"]) not in spurious_districts]
old_wards = [unit(w, district_code=w["districtCode"],
                  province_code=fixed_ward_province.get(w["code"], w["provinceCode"])) for w in lw]

for t, scope in [(new_provinces, lambda x: ""), (new_wards, lambda x: x["province_code"]),
                 (old_provinces, lambda x: ""), (old_districts, lambda x: x["province_code"]),
                 (old_wards, lambda x: x["district_code"])]:
    ranked(t, scope)

ward_mappings, district_mappings, unresolved = [], [], []
district_has_wards = {w["districtCode"] for w in lw}
for m in mp:
    row = {"new_ward_code": m["newWardCode"], "mapping_type": m["type"], "note": m["note"]}
    if "oldWardCode" in m:
        ward_mappings.append({"old_ward_code": m["oldWardCode"], **row})
    elif m["oldDistrictCode"] in district_has_wards:
        unresolved.append({
            "kind": "district_row_for_district_with_wards",
            "row": {"old_province_code": m["oldProvinceCode"], "old_district_code": m["oldDistrictCode"], **row},
            "reason": "a whole-district mapping exists although the district has wards (each ward has its "
                      "own mapping); not loaded as a district mapping, needs checking against the official table",
        })
    else:
        district_mappings.append({"old_district_code": m["oldDistrictCode"], **row})

# ---- self-check (same rules the seeder's Verify enforces) --------------------------
errs = []
old_w = {w["code"] for w in old_wards}
new_w = {w["code"] for w in new_wards}
dist = {x["code"]: x for x in old_districts}
if len(dist) != len(old_districts):
    errs.append("duplicate district code")
for w in old_wards:
    if dist[w["district_code"]]["province_code"] != w["province_code"]:
        errs.append(f'ward {w["code"]} province != district province')
mapped_old = {m["old_ward_code"] for m in ward_mappings}
mapped_new = {m["new_ward_code"] for m in ward_mappings} | {m["new_ward_code"] for m in district_mappings}
errs += [f"old ward without mapping {c}" for c in sorted(old_w - mapped_old)]
errs += [f"new ward never a target {c}" for c in sorted(new_w - mapped_new)]
errs += [f"mapping to unknown new ward {m['new_ward_code']}" for m in ward_mappings if m["new_ward_code"] not in new_w]
errs += [f"mapping from unknown old ward {m['old_ward_code']}" for m in ward_mappings if m["old_ward_code"] not in old_w]
new_wp = {w["code"]: w["province_code"] for w in new_wards}
prov_target = collections.defaultdict(set)
ow = {w["code"]: w for w in old_wards}
for m in ward_mappings:
    prov_target[ow[m["old_ward_code"]]["province_code"]].add(new_wp[m["new_ward_code"]])
multi = {k: sorted(v) for k, v in prov_target.items() if len(v) > 1}
if multi:
    errs.append(f"old province -> several new provinces: {multi}")
if errs:
    print("INTEGRITY FAILURES:", *errs[:20], sep="\n  ")
    sys.exit(1)

meta = {
    "name": "Vietnam administrative units — old (63 provinces) and new (34 provinces) schemes with ward-level mapping",
    "version": "2026-09-25",
    "format": 1,
    "source": {
        "dataset": "vietnam-address-kit",
        "repository": "https://github.com/zindont/vietnam-address-kit",
        "path": "src/data/official/*.json",
        "commit": commit,
        "commit_date": commit_date,
        "code_license": "MIT",
        "data_provenance": "generated by the dataset's authors from the official conversion table under "
                           "Quyết định 19/2025/QĐ-TTg, NQ-UBTVQH resolutions and 2026 amendments "
                           "(hand-transcribed by them); NOT yet reconciled against the official file",
    },
    "sort_order": "1-based Vietnamese alphabetical rank of `name` within its parent (provinces: global); "
                  "digits compare numerically; DB-collation independent",
    "counts": {
        "new_provinces": len(new_provinces), "new_wards": len(new_wards),
        "old_provinces": len(old_provinces), "old_districts": len(old_districts),
        "old_wards": len(old_wards), "ward_mappings": len(ward_mappings),
        "district_mappings": len(district_mappings),
    },
    "corrections": corrections + [{
        "kind": "duplicate_district_row_removed",
        "unit": f"district {d} under province {p}",
        "reason": "duplicate district row created by the ward-province defect above",
    } for d, p in sorted(spurious_districts)],
    "unresolved": unresolved,
    "notes": [
        "Old and new codes live in separate namespaces: the same number can denote different units "
        "(e.g. 26737 is old ward Đa Kao and new ward Tân Định). Always key by scheme.",
        "district_mappings cover districts that have no wards (islands); not used by the API yet "
        "(district conversion deferred).",
        "ward_mappings is many-to-many; 'ambiguous'/'split_population' rows mean part of the old ward "
        "moved to this new ward.",
    ],
}

doc = collections.OrderedDict([
    ("meta", meta), ("new_provinces", new_provinces), ("new_wards", new_wards),
    ("old_provinces", old_provinces), ("old_districts", old_districts), ("old_wards", old_wards),
    ("ward_mappings", ward_mappings), ("district_mappings", district_mappings),
])


def dump(o):
    return json.dumps(o, ensure_ascii=False, separators=(",", ":"))


with open(out, "w", encoding="utf-8") as f:
    f.write("{\n")
    keys = list(doc)
    for i, k in enumerate(keys):
        v = doc[k]
        if k == "meta":
            f.write(f'  "meta": ' + json.dumps(v, ensure_ascii=False, indent=2).replace("\n", "\n  "))
        else:
            f.write(f'  "{k}": [\n')
            f.write(",\n".join("    " + dump(x) for x in v))
            f.write("\n  ]")
        f.write(",\n" if i < len(keys) - 1 else "\n")
    f.write("}\n")

print("OK", json.dumps(meta["counts"]), "corrections:", len(meta["corrections"]), "unresolved:", len(unresolved))
print(os.path.getsize(out), "bytes")
