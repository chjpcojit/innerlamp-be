# assets/data/sutras

Curated Buddhist texts (kinh, kệ, chú) collected from public web sources, one JSON file per text plus `index.json` (metadata only, no lines). **Not wired into code yet**: nothing embeds, seeds or serves it (`assets/embed.go` embeds only `buddhist_articles.json`). ~11 MB, so do not `go:embed` the whole directory without deciding on a loading strategy first.

## Shape

`index.json` → `{"sutras":[{id, tradition, kind, collection, reference, originalTitle?, author?, hanTranslator?, hanTranslationYear?, referenceNote?, summary?, titles, deities, dailyChant, purposes, durationMinutes, reviewStatus, translations:[{key, lang, variant, translator, license, licenseUrl?, licenseNote?, lineCount}]}]}`.
`<id>.json` → the same metadata plus, per translation, `sourceName`, `sourceUrl`, optional `note`, and `lines: [{n, text, section?}]` (display lines for the phone UI; `n` is 1-based and contiguous **per translation**, languages do not align line-for-line).

- `tradition`: `THERAVADA` | `MAHAYANA` | `VAJRAYANA`
- `kind`: `SUTRA` (prose) | `VERSE` (kệ, Pháp Cú, sám, nguyện) | `DHARANI` (chú, chanted by sound)
- `reference`: **only the citation code** (`MN 86`, `Dhp 1-20`, `Taishō T0450`, `Toh 438, …`); `""` when the source has none. What used to be packed after it has its own optional field (missing = `""`): `originalTitle` (Hán-Việt / source title, or the scripture a dharani is drawn from), `author` (who composed it, not who translated), `hanTranslator`, `hanTranslationYear` (integer CE, or `null` / missing when unknown), `referenceNote` (any remainder, e.g. "Bản Đôn Hoàng").
- `summary`: a 2–4 sentence Vietnamese description of the text, **written by AI from the title, notes and the first/last lines** (2026-10-07, see `docs/assumptions/sutra/A-011`); not reviewed by anyone knowledgeable. Optional for the parser, present in all 153 files.
- `translations[].license`: a code from the closed set `CC0-1.0`, `CC-BY-NC-4.0`, `PUBLIC_DOMAIN`, `COPY_ALLOWED` (the page says copying is welcome; not a standard licence), `UNKNOWN`. `licenseUrl` is the URL the source gives (only the CC0 entries have one) and `licenseNote` is the source's own statement, verbatim; both optional.
- `variant`: `translation` | `chant` (Hán-Việt / transliteration, as chanted) | `meaning` (dịch nghĩa) | `poem`. One text may carry several `vi` entries; pick by `variant`, not by `lang`.
- `lang`: `vi`, `pli`, `en`, `zh`, `sa`, `bo` (Tibetan script or Latin phonetics, see `note`).
- `deities[]`: `{deity_id, role}` with `deity_id` from `../phat_botat.json`; `role` is `SPEAKER` | `SUBJECT` | `MENTIONED`.
- `translations[].key`: unique within a text, `"<lang>-<variant>"` with `-2`, `-3` from the second duplicate (only `tam-kinh` has one: `vi-meaning`, `vi-meaning-2`). The API selects a translation by `key`, because `lang` + `variant` is not always unique.
- `dailyChant`: `true` when the text is part of daily chanting (nhật tụng, sám hối, Thập Chú, Chú Đại Bi, Tâm Kinh, A Di Đà, Phổ Môn, Kinh Từ Bi, Kinh Hạnh Phúc, …), assigned by the collector from the table in `docs/specs/features/sutra/design.md`; 34 of 153 are `true`.
- `purposes`: list from the closed set `CAU_AN`, `CAU_SIEU`, `TRI_CHU`, `SAM_HOI`, `HOI_HUONG`, `PHAT_NGUYEN`, `BAO_HIEU`, `THIEN_DINH`, `HOC_GIAO_LY`, `QUY_Y_THO_GIOI`; `[]` when none fits (6 texts). Assigned by the collector from the table in `docs/specs/features/sutra/design.md`, not by a religious reviewer.
- `durationMinutes`: an **editorial estimate** of the minutes needed to chant/read the whole text, not a measurement. Initial value is `ceil(syllables / 80)` of the Vietnamese text (preferring `chant` > `translation` > `meaning` > `poem`), rounded for readability (≤10 as is, 11–60 up to a multiple of 5, 61–180 up to a multiple of 10, above that a multiple of 30); edit by hand when you know better. `null` when there is no Vietnamese text (10 Vajrayana texts).

## Before shipping

1. **Licenses** are recorded per translation (code in `license`, the source's own words in `licenseNote`), never guessed. Only Pali (Mahāsaṅgīti), English (Sujato) and the Pháp Cú Vietnamese (`phantuananh`) are CC0, and Wikisource zh/sa are public domain. Most Vietnamese texts are `unknown` or only "free to copy" (phatgiao.org.vn). Vajrayana files are Lotsawa House **CC BY-NC 4.0** (non-commercial). Clear permission before a commercial release.
2. Sujato asks that his English not be used for AI training.
3. `reviewStatus` is `UNREVIEWED` everywhere: titles, deity links, Hán/Sanskrit titles and Taishō numbers were added by the collector, not by the sources, and Vietnamese source typos were kept verbatim (e.g. "Quán Thế Ám"). Have a knowledgeable person review.
4. `chu-dai-bi` is the phatgiao.org.vn numbered 84-line version (chosen as the authoritative source). The same page also has a prose version that disagrees with its own numbered list, and the earlier hoatieu.vn text differed in 8 syllables (e.g. "kiết đế" vs "yết đế"); the variances are listed in the file's `note`. Still worth a monk's check, since a mantra is chanted syllable by syllable.
5. `dai-niem-xu-trung-bo` (MN 10) and `dai-niem-xu-truong-bo` (DN 22) are different suttas.
