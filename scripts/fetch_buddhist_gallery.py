#!/usr/bin/env python3
"""Build assets/data/buddhist_gallery.json: curated Buddhist photos from Wikimedia Commons.

Each entry: id, type, title, location, thumbnail_url (500px), image_url (1920px),
plus credit fields (author, license, source_url) required by CC licenses.
URLs hotlink upload.wikimedia.org (stable). Run from anywhere: python3 scripts/fetch_buddhist_gallery.py
"""
import html, json, re, sys, time, urllib.parse, urllib.request
from pathlib import Path

UA = "innerlamp-seed/1.0 (duc@wee.vn)"
OUT = Path(__file__).resolve().parent.parent / "assets/data/buddhist_gallery.json"
API = "https://commons.wikimedia.org/w/api.php"
FREE = re.compile(r"^(CC0|CC BY|CC BY-SA|Public domain|PD)", re.I)
BAD = re.compile(r"map|plan|diagram|logo|icon|flag|stamp|coin|poster|screenshot|book|manuscript|\.svg|\.tif|\.pdf", re.I)

# Commons hits that matched the query but not the subject (replica, painting, wrong site...).
EXCLUDE = ("Tachileik", "Wat_Khun_Inthapramun", "myrobalan", "Monastery_6_monk_cell", "Pacific_Partnership",
           "Huile_sur_toile", "Đình", "Bao_Hai_Linh_Thong", "Lascar_One_Pillar", "Death_of_Buddha",
    "Ang_Thong_-_Pho_Thong_(2024)_Reclining_Buddha_Wat_Khun_Intha",
    "A_monk_meditates_under_a_tree.jpg",
    "Bodhgaya_20_Mahabodhi_Temple,_lotus_flowers_(32949923781).jp",
    "20160125_Sri_Lanka_3886_crop_Kandy_sRGB_(25675623531).jpg",
    "Kinnla_Incense_at_the_Mahabodhi_Temple.jpg",
    "Pacific_Partnership_2026_conducts_faith-based_host_nation_ou",
    "Buddhist_procession_during_the_Waisak_Day_at_Borobudur,_Indo",
    "Dai_le_vesak_sa_dec.JPG",
    "Vesak_in_Singapore.jpg",
    "Jakar_tshechu,_public_(15656565548).jpg",
    "Jakar_tshechu,_school_children_(15226200413).jpg",
    "Chiang_Mai,_Loy_Krathong_Lantern_Festival,_Thailand.jpg",
)
# Fix title/location when the best photo is of a more specific subject (matched on source filename).
RELABEL = {
    "Maya_Devi_Temple_with_Monks": ("Chư tăng tại đền Maya Devi", "Lumbini, Nepal"),
    "Shakyamuni_Buddha_Statue_Linh_Ung": ("Tượng Phật Thích Ca chùa Linh Ứng", "Bà Nà Hills, Đà Nẵng, Việt Nam"),
    "Weishan_Guanyin": ("Quan Âm nghìn tay nghìn mắt", "Vy Sơn, Vân Nam, Trung Quốc"),
    "Kinnla_Incense": ("Hương trầm tại đại tháp Mahabodhi", "Bodh Gaya, Ấn Độ"),
    "Buddha's_Birthday-Seoul": ("Lễ Phật Đản", "Seoul, Hàn Quốc"),
    "One_Pillar_Pagoda,_Hanoi_(30646044573)": ("Chùa Một Cột và hoa sen", "Hà Nội, Việt Nam"),
    "Mucalinda_in_Dambulla": ("Tượng Phật Mucalinda hang Dambulla", "Dambulla, Sri Lanka"),
    "Buddha_Head_in_Tree_Roots": ("Đầu tượng Phật trong rễ cây", "Ayutthaya, Thái Lan"),
}

# (type, title, location, commons search query, how many)
SPECS = [
    # --- CHUA (chùa / kiến trúc) ---
    ("CHUA", "Chùa Một Cột", "Hà Nội, Việt Nam", "One Pillar Pagoda Hanoi", 2),
    ("CHUA", "Chùa Trấn Quốc", "Hà Nội, Việt Nam", "Tran Quoc Pagoda Hanoi", 2),
    ("CHUA", "Chùa Thiên Mụ", "Huế, Việt Nam", "Thien Mu Pagoda Hue", 2),
    ("CHUA", "Chùa Bái Đính", "Ninh Bình, Việt Nam", "Bai Dinh Pagoda", 2),
    ("CHUA", "Chùa Hương", "Hà Nội, Việt Nam", "Huong Pagoda Perfume", 2),
    ("CHUA", "Chùa Cầu (Cầu Nhật Bản)", "Hội An, Việt Nam", "Japanese Covered Bridge Hoi An", 1),
    ("CHUA", "Chùa Linh Ứng", "Đà Nẵng, Việt Nam", "Linh Ung Pagoda Da Nang", 2),
    ("CHUA", "Chùa Vĩnh Tràng", "Tiền Giang, Việt Nam", "Vinh Trang Pagoda My Tho", 1),
    ("CHUA", "Chùa Bà Đen", "Tây Ninh, Việt Nam", "Ba Den Mountain pagoda Tay Ninh", 1),
    ("CHUA", "Chùa Vàng Shwedagon", "Yangon, Myanmar", "Shwedagon Pagoda", 2),
    ("CHUA", "Bagan lúc bình minh", "Bagan, Myanmar", "Bagan temples sunrise", 2),
    ("CHUA", "Angkor Wat", "Siem Reap, Campuchia", "Angkor Wat sunrise", 2),
    ("CHUA", "Bayon", "Angkor, Campuchia", "Bayon temple faces", 1),
    ("CHUA", "Borobudur", "Java, Indonesia", "Borobudur sunrise", 2),
    ("CHUA", "Wat Arun", "Bangkok, Thái Lan", "Wat Arun Bangkok", 2),
    ("CHUA", "Wat Rong Khun (Chùa Trắng)", "Chiang Rai, Thái Lan", "Wat Rong Khun White Temple", 2),
    ("CHUA", "Wat Pho", "Bangkok, Thái Lan", "Wat Pho Bangkok", 1),
    ("CHUA", "Kim Các Tự (Kinkaku-ji)", "Kyoto, Nhật Bản", "Kinkaku-ji Golden Pavilion", 2),
    ("CHUA", "Chùa Thanh Thủy (Kiyomizu-dera)", "Kyoto, Nhật Bản", "Kiyomizu-dera Kyoto", 1),
    ("CHUA", "Chùa Sensō-ji", "Tokyo, Nhật Bản", "Senso-ji Asakusa", 1),
    ("CHUA", "Cung điện Potala", "Lhasa, Tây Tạng", "Potala Palace Lhasa", 2),
    ("CHUA", "Đại Tháp Mahabodhi", "Bodh Gaya, Ấn Độ", "Mahabodhi Temple Bodh Gaya", 2),
    ("CHUA", "Bulguksa", "Gyeongju, Hàn Quốc", "Bulguksa temple", 1),
    ("CHUA", "Tu viện Tiger's Nest", "Paro, Bhutan", "Paro Taktsang Tiger's Nest", 2),
    ("CHUA", "Chùa Thiếu Lâm", "Hà Nam, Trung Quốc", "Shaolin Monastery", 1),
    # --- TUONG_PHAT (tượng Phật) ---
    ("TUONG_PHAT", "Tượng Phật trong chùa Việt", "Việt Nam", "Buddha statue Vietnam pagoda", 2),
    ("TUONG_PHAT", "Tượng Phật Bà Quan Âm", "Đà Nẵng, Việt Nam", "Lady Buddha Da Nang Linh Ung", 2),
    ("TUONG_PHAT", "Tượng Phật Thích Ca núi Bà", "Tây Ninh, Việt Nam", "Ba Den Buddha statue", 1),
    ("TUONG_PHAT", "Đại Phật Lạc Sơn", "Tứ Xuyên, Trung Quốc", "Leshan Giant Buddha", 2),
    ("TUONG_PHAT", "Đại Phật Thiên Đàn", "Hồng Kông", "Tian Tan Buddha Hong Kong", 2),
    ("TUONG_PHAT", "Đại Phật Todai-ji", "Nara, Nhật Bản", "Daibutsu Todai-ji Nara", 1),
    ("TUONG_PHAT", "Đại Phật Kamakura", "Kamakura, Nhật Bản", "Kamakura Daibutsu", 2),
    ("TUONG_PHAT", "Phật Nằm Wat Pho", "Bangkok, Thái Lan", "Reclining Buddha Wat Pho", 2),
    ("TUONG_PHAT", "Tượng Phật Ayutthaya trong rễ cây", "Ayutthaya, Thái Lan", "Buddha head tree roots Ayutthaya", 2),
    ("TUONG_PHAT", "Đại Phật Ling Shan", "Vô Tích, Trung Quốc", "Lingshan Grand Buddha Wuxi", 1),
    ("TUONG_PHAT", "Tượng Phật Spring Temple", "Hà Nam, Trung Quốc", "Spring Temple Buddha", 1),
    ("TUONG_PHAT", "Tượng Phật đá Gal Vihara", "Polonnaruwa, Sri Lanka", "Gal Vihara Polonnaruwa", 1),
    ("TUONG_PHAT", "Tượng Phật Dambulla", "Dambulla, Sri Lanka", "Dambulla cave temple Buddha", 1),
    ("TUONG_PHAT", "Tượng Quan Âm nghìn tay", "Trung Quốc", "Guanyin thousand hands statue", 1),
    ("TUONG_PHAT", "Phật Ngọc Wat Phra Kaew", "Bangkok, Thái Lan", "Emerald Buddha Wat Phra Kaew", 1),
    # --- THIEN (thiền / tu tập) ---
    ("THIEN", "Chư tăng đi khất thực", "Luang Prabang, Lào", "monks alms giving Luang Prabang", 2),
    ("THIEN", "Tăng sĩ thiền định", "Châu Á", "monk meditating temple", 2),
    ("THIEN", "Thiền hành trong vườn", "Kyoto, Nhật Bản", "Zen garden Ryoan-ji", 2),
    ("THIEN", "Vườn thiền Kennin-ji", "Kyoto, Nhật Bản", "Kennin-ji Zen garden", 1),
    ("THIEN", "Sa di trẻ trong chùa", "Myanmar", "novice monks Myanmar", 2),
    ("THIEN", "Chư tăng cầu nguyện", "Bodh Gaya, Ấn Độ", "monks praying Bodh Gaya", 1),
    ("THIEN", "Hoa sen và thiền định", "Việt Nam", "lotus flowers Buddhist temple offering", 2),
    ("THIEN", "Dâng đèn cầu nguyện", "Tây Tạng", "butter lamps Tibetan monastery", 1),
    ("THIEN", "Bánh xe cầu nguyện", "Dharamsala, Ấn Độ", "prayer wheels Tibetan Buddhism", 2),
    ("THIEN", "Nến và hương trầm trước điện Phật", "Việt Nam", "incense offering Buddhist temple", 2),
    # --- LE_HOI (lễ hội) ---
    ("LE_HOI", "Lễ Phật Đản thả đèn hoa đăng", "Huế, Việt Nam", "Vesak lantern festival Buddhist", 2),
    ("LE_HOI", "Lễ hội Vesak Borobudur", "Java, Indonesia", "Waisak Borobudur", 2),
    ("LE_HOI", "Đại lễ Phật Đản", "Sa Đéc, Đồng Tháp, Việt Nam", "Dai le vesak sa dec", 1),
    ("LE_HOI", "Lễ Phật Đản Vesak", "Singapore", "Vesak in Singapore", 1),
    ("LE_HOI", "Lễ Phật Đản Vesak", "Indonesia", "Indonesia Buddhists Vesak Day", 1),
    ("LE_HOI", "Lễ Magha Puja", "Wat Khung Taphao, Thái Lan", "Magha Puja Day in Wat Khung Taphao", 2),
    ("LE_HOI", "Dâng cúng thức ăn cho chư tăng", "Don Det, Lào", "trays with food offered to the monks Don Det", 1),
    ("LE_HOI", "Lễ hội đèn lồng Phật Đản", "Seoul, Hàn Quốc", "Lotus Lantern Festival Seoul", 2),
    ("LE_HOI", "Lễ hội Esala Perahera", "Kandy, Sri Lanka", "Esala Perahera Kandy", 2),
    ("LE_HOI", "Lễ hội Thadingyut", "Myanmar", "Thadingyut festival of lights Myanmar", 1),
    ("LE_HOI", "Lễ hội Losar", "Tây Tạng", "Losar Tibetan New Year monastery", 1),
    ("LE_HOI", "Lễ hội Tshechu múa mặt nạ", "Bhutan", "Tshechu festival Bhutan mask dance", 2),
    ("LE_HOI", "Lễ hội Loy Krathong", "Thái Lan", "Loy Krathong lanterns", 1),
]


def get(url, retries=4):
    for i in range(retries):
        try:
            req = urllib.request.Request(url, headers={"User-Agent": UA})
            with urllib.request.urlopen(req, timeout=40) as r:
                return json.loads(r.read())
        except Exception:
            if i == retries - 1:
                raise
            time.sleep(3 * (i + 1))


def strip_tags(s):
    return html.unescape(re.sub(r"<[^>]+>", "", s or "")).strip()


def search(q):
    p = dict(action="query", format="json", generator="search", gsrnamespace=6, gsrsearch=q + " filetype:bitmap",
             gsrlimit=30, prop="imageinfo", iiprop="url|size|mime|extmetadata", iiurlwidth=1920)
    r = get(API + "?" + urllib.parse.urlencode(p))
    pages = sorted(r.get("query", {}).get("pages", {}).values(), key=lambda p: p["index"])
    return pages


def thumb_at(info, width):
    # Commons thumb path: .../thumb/<h>/<hh>/<file>/<width>px-<file>
    url = info["thumburl"]
    return re.sub(r"/\d+px-", f"/{width}px-", url)


def pick(q, n, used):
    out = []
    for p in search(q):
        info = (p.get("imageinfo") or [{}])[0]
        title = p["title"]
        meta = info.get("extmetadata", {})
        lic = meta.get("LicenseShortName", {}).get("value", "")
        w, h = info.get("width", 0), info.get("height", 0)
        if (info.get("mime") != "image/jpeg" or BAD.search(title) or not FREE.match(lic)
                or w < 2000 or h < 1200 or w < h * 0.9 or title in used or "thumburl" not in info
                or any(x in title for x in EXCLUDE)):
            continue
        used.add(title)
        out.append(dict(
            file=title,
            thumbnail_url=thumb_at(info, 500),
            image_url=info["thumburl"],
            author=strip_tags(meta.get("Artist", {}).get("value"))[:120],
            license=lic,
            source_url=info["descriptionurl"],
        ))
        if len(out) == n:
            break
    return out


def main():
    items, used = [], set()
    for typ, title, loc, q, n in SPECS:
        try:
            got = pick(q, n, used)
        except Exception as e:
            print(f"! {q}: {e}", file=sys.stderr)
            got = []
        print(f"{len(got)}/{n}  {typ:10} {title}")
        for g in got:
            t, l = next((v for k, v in RELABEL.items() if k in g["file"]), (title, loc))
            g.pop("file")
            items.append(dict(id=f"gallery_{len(items)+1:03}", type=typ, title=t, location=l, **g))
        time.sleep(0.4)
    OUT.write_text(json.dumps(items, ensure_ascii=False, indent=2) + "\n")
    print(f"wrote {len(items)} -> {OUT}")


if __name__ == "__main__":
    main()
