"""Write reviewed structured addresses into assets/data/temple.json.

Spec: docs/specs/features/temple/ (v1.7+, US-TMP-08, tasks.md Phase 10). Companion of
scripts/build_temple_addresses.py, which only PROPOSES; this script applies the proposal plus the
reviewer's decisions.

    python3 -I scripts/apply_temple_addresses.py            # dry run: validate and report
    python3 -I scripts/apply_temple_addresses.py --write    # patch temple.json (backup first)

Inputs (defaults):
  assets/data/temple.json                         JSONC; patched in place as TEXT so the /* */ region
                                                  headings and every other byte stay as they were
  assets/data/temple-address-proposal.json        from build_temple_addresses.py
  docs/specs/features/temple/address-decisions.json  {"<temple id>": {"ward_code": "...", "address"?: "..."}}
                                                  overrides / fills the rows the proposal left `review`
  assets/data/administrative-units.json           validates every code

Rules: a temple is written only when its ward_code exists in new_wards, belongs to the province of the
temple's current `location.province`, and the proposal row is `auto` or has a decision. Anything else
is reported and left untouched. The result replaces `"address"` + `"province"` of each patched
`location` with `"address"` (below ward level), `"province_code"`, `"ward_code"`; the province NAME is
no longer in the file (the seed takes it from the dataset by code, spec AC 35).
"""
import json
import os
import re
import shutil
import sys

args = [a for a in sys.argv[1:] if not a.startswith("--")]
WRITE = "--write" in sys.argv
temple_path = args[0] if len(args) > 0 else "assets/data/temple.json"
proposal_path = args[1] if len(args) > 1 else "assets/data/temple-address-proposal.json"
decisions_path = args[2] if len(args) > 2 else "docs/specs/features/temple/address-decisions.json"
units_path = args[3] if len(args) > 3 else "assets/data/administrative-units.json"

text = open(temple_path, encoding="utf-8").read()
proposal = json.load(open(proposal_path, encoding="utf-8"))["proposals"]
decisions = json.load(open(decisions_path, encoding="utf-8")) if os.path.exists(decisions_path) else {}
units = json.load(open(units_path, encoding="utf-8"))
new_ward = {w["code"]: w for w in units["new_wards"]}

patched, skipped = [], []
for p in proposal:
    tid = p["id"]
    dec = decisions.get(tid, {})
    ward_code = dec.get("ward_code") or (p["ward_code"] if p["status"] == "auto" else None)
    address = dec["address"] if "address" in dec else p["address"]
    if not ward_code:
        skipped.append((tid, "no ward_code: proposal is review and there is no decision"))
        continue
    w = new_ward.get(ward_code)
    if w is None:
        skipped.append((tid, f"ward_code {ward_code} is not in new_wards"))
        continue
    if w["province_code"] != p["province_code"]:
        skipped.append((tid, f"ward {ward_code} is in province {w['province_code']}, temple is in {p['province_code']}"))
        continue
    # Locate this temple's location block: first `"address": "...", "province": "..."` after its id.
    m_id = re.search(r'"id":\s*"%s"' % re.escape(tid), text)
    if not m_id:
        skipped.append((tid, "id not found in temple.json"))
        continue
    pat = re.compile(r'(?P<ind>[ \t]*)"address":\s*"(?P<addr>(?:[^"\\]|\\.)*)",\s*\n\s*"province":\s*"(?P<prov>(?:[^"\\]|\\.)*)"')
    m = pat.search(text, m_id.end())
    if not m:
        skipped.append((tid, "location block already patched or has an unexpected shape"))
        continue
    ind = m.group("ind")
    new_block = (
        f'{ind}"address": {json.dumps(address, ensure_ascii=False)},\n'
        f'{ind}"province_code": "{p["province_code"]}",\n'
        f'{ind}"ward_code": "{ward_code}"'
    )
    patched.append((tid, m.start(), m.end(), new_block, ward_code, w["full_name"], address))

# apply from the end so earlier offsets stay valid
out = text
for tid, a, b, blk, *_ in sorted(patched, key=lambda x: -x[1]):
    out = out[:a] + blk + out[b:]

print(f"temples in proposal: {len(proposal)}; to patch: {len(patched)}; skipped: {len(skipped)}")
for tid, why in skipped:
    print("  SKIP", tid, "-", why)
if not WRITE:
    print("dry run: nothing written (use --write)")
    sys.exit(0 if not skipped else 1)
if skipped:
    print("refusing to write while rows are skipped")
    sys.exit(1)

backup = temple_path + ".pre-v1.7.bak"
if not os.path.exists(backup):
    shutil.copyfile(temple_path, backup)
    print("backup:", backup)
with open(temple_path, "w", encoding="utf-8") as f:
    f.write(out)
print("wrote", temple_path)
