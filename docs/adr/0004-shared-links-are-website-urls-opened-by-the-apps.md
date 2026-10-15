# ADR-0004: Shared links are website URLs that the apps open as verified Universal Links / App Links

| Field | Value |
|---|---|
| Status | Proposed |
| Date | 2026-10-15 |
| Deciders | anh Đức (decides) · Claude (`adr-writer`, proposes) |
| Features | cross-cutting (`article` first; later any shareable content such as `temple`, `calendar`, `deity`) |
| Platforms | Website, iOS, Android (Backend: none) |
| Supersedes | — |
| Related | `/mnt/project-files/reports/product/article/2026-10-15-cau-hoi-truoc-khi-viet.md` (Q3 and its answer) · `/mnt/project-files/reports/infra/2026-10-14-thiet-ke-gcp.md` (planned hostnames, domain not confirmed) |

## Context

The article feature gets a **Share** action. The owner decided (2026-10-15) that the shared link must
be a link to the InnerLamp website, and that a person who opens it on a phone with the app installed
lands **in the app** on the same content ("why not universal links?"). Without the app, the link
opens the website page, which already exists for articles.

This is the first shareable content, but not the last: temples, Buddhist events, Buddhas &
Bodhisattvas and future content all have a public website page and an app screen. Whatever URL shape
and opening mechanism is chosen now is baked into every link people paste into chats; after links
are out in the wild the shape cannot change without breaking them.

What exists today (checked 2026-10-15 on each repo's `main`):

- **Website** (`innerlamp-website`, SvelteKit, `adapter-auto`, not deployed): public pages per item,
  e.g. `src/routes/(site)/articles/[id]/`, `temples/`, `events/`. Locale routing: Vietnamese is
  unprefixed (`/articles/<id>`), English is `/en/articles/<id>` (`AGENTS.md` §4). `static/` holds only
  `robots.txt`; there is no `.well-known/` content.
- **iOS** (`innerlamp-iOS`): bundle id `weedigital.vn.InnerLamp`; no Associated Domains entitlement
  and no URL scheme handler (no `applinks`, no `continueUserActivity`). The article Share button is a
  `// TODO` (`ILArticleCoordinator.swift`).
- **Android** (`innerlamp-Android`): `applicationId = "com.example.innerlamp"` (template value, with
  `.dev` / `.staging` suffixes per flavor); the manifest has only the launcher intent filter. Share is
  a `TODO` (`MainActivity.kt`).
- **Domain**: none owned yet. The GCP design proposes `innerlamp.vn` for the website, marked "cần chốt"
  (not confirmed). The mailer is mocked for the same reason (ADR-0003).
- **Backend**: not involved; article ids are stable slugs (`migrations/0012_articles.sql`), so a URL
  built from an id stays valid across reseeds.

### Why this needs an ADR

Gate criterion "cross-cutting convention" plus "hard to reverse": every feature with a Share action
will reuse it, and a link format that has been shared publicly cannot be changed without breaking
the links people already sent. It also binds the website to serve verification files for the apps.

## Decision drivers

- One link works for everyone: app installed → opens the app; no app, desktop, or a crawler → the
  website page.
- No paid third-party service and no extra backend component (personal project, cost-sensitive).
- Links must not be hijackable by another app (verified domain ownership).
- The same URL shape for every feature, so a new shareable feature only adds a path.
- Works before launch with a staging domain, and with the real domain later, without rewriting specs.

## Options considered

### Option 0: Keep the status quo

No Share, or Share sends the original source link (for articles: the third-party page the summary
came from). Nothing to build.

| Dimension | Assessment |
|---|---|
| Complexity | None |
| Cost | None |
| Reversibility | Trivial |
| Cross-platform impact | None |
| Security & privacy | No concern |
| Reliability | Depends on third-party sites |
| Fit with current stack | Fits |
| Familiarity | n/a |

**Pros:** zero work.
**Cons:** the recipient never reaches InnerLamp; rejected by the owner (Q3).

### Option A: Website URLs + verified Universal Links (iOS) and App Links (Android)

The shared link is the canonical website URL of the item, e.g. `https://<web-domain>/articles/<id>`.
The website serves `/.well-known/apple-app-site-association` and `/.well-known/assetlinks.json`
for that domain. The iOS app declares `applinks:<web-domain>` and the Android app declares a
verified (`autoVerify`) intent filter for the same host, each limited to the **paths it can
render** (an allow-list grown feature by feature). When the OS opens such a link in the app, the app
routes the path to the matching screen; any path it does not know is handed back to the browser.
Without the app, the OS opens the website.

| Dimension | Assessment |
|---|---|
| Complexity | Low–medium: two static files on the website, one entitlement / one intent filter, one URL router per app |
| Cost | Domain + TLS only (needed for the website anyway) |
| Reversibility | Medium: the mechanism can be swapped, the URL shape cannot once links are shared |
| Cross-platform impact | Website serves files; iOS and Android add routing; backend none |
| Security & privacy | OS verifies domain ownership against the app's signing identity; no other app can claim the links; no tracking service |
| Reliability | Standard OS mechanism; some in-app browsers (e.g. chat apps) open the web page instead of the app — the web page is a correct fallback |
| Fit with current stack | Website already has the public pages and locale routing |
| Familiarity | Well documented on both OSes |

**Pros:** one link for all audiences; free; secure; the web page is a real fallback, not a redirect.
**Cons:** needs a real domain and the final app identities (iOS Team ID + bundle id, Android package
name + release signing certificate) before it can work end to end; no "open after install" (deferred
deep link).

### Option B: Custom URL scheme (`innerlamp://articles/<id>`)

The app registers a private scheme and Share sends that URL (optionally with a web link beside it).

| Dimension | Assessment |
|---|---|
| Complexity | Low |
| Cost | None |
| Reversibility | Easy |
| Cross-platform impact | iOS and Android only; meaningless on the web and on desktop |
| Security & privacy | Any app can register the same scheme (hijackable); no verification |
| Reliability | Broken link for anyone without the app; chat apps often do not linkify custom schemes |
| Fit with current stack | Ignores the existing website pages |
| Familiarity | High |

**Pros:** no domain needed.
**Cons:** fails the first driver (no fallback) and the security driver.

### Option C: Third-party deep-link service (Branch, Adjust, …)

A hosted service issues short links, routes to the store or the app, and supports deferred deep
links. Firebase Dynamic Links, the free option, was shut down in August 2025.

| Dimension | Assessment |
|---|---|
| Complexity | Medium (SDK in both apps, dashboard) |
| Cost | Paid beyond small free tiers |
| Reversibility | Hard: shared links point at the vendor's domain |
| Cross-platform impact | SDK in both apps |
| Security & privacy | Third party sees every click; tracking SDK in the apps |
| Reliability | Depends on the vendor staying in business (see Firebase) |
| Fit with current stack | New dependency |
| Familiarity | Medium |

**Pros:** deferred deep links, analytics.
**Cons:** cost, lock-in on the link domain, privacy; not needed by any current requirement.

## Trade-off analysis

Option A is the only one that meets every driver: one URL for app users, web users and crawlers,
verified ownership, no recurring cost. Its only real gap against C is deferred deep linking (open the
shared item right after installing the app), which no requirement asks for. B is cheaper only
because it skips the fallback the owner explicitly wants. The cost of A is front-loaded on things
the project needs before launch anyway: a domain and final app identities.

## Decision

**Recommended:** Option A — shared links are canonical website URLs, opened by the apps through
verified Universal Links / App Links, with the website page as the fallback.

Conventions that come with it (to be confirmed together with the option):

1. **URL shape.** `https://<web-domain>/<collection>/<id>` using the website's own public route and
   the item's stable id (articles: `/articles/<id>`). The shared link uses the unprefixed (Vietnamese
   base locale) form; the apps also accept the `/en/…` form of the same path.
2. **Path allow-list.** Each app claims only the paths of features it can show; a feature adds its
   path when its app screen ships. Unknown paths open in the browser.
3. **Unknown or removed item.** The app opens and shows its normal "not found" state for that
   feature; it never falls back to a different item.
4. **Who serves what.** The website serves both association files from its own domain; the backend is
   not involved.
5. **Environments.** Each environment's app build (dev / staging / prod) is associated with that
   environment's website domain only.
6. **Out of scope.** Deferred deep links, short links, link analytics.

## Consequences

- **Easier:** any future shareable feature only adds a path to the website and to the apps'
  allow-lists; links shared from any platform look and behave the same.
- **Harder:** the release checklist gains "association files reachable over HTTPS, no redirect,
  correct content type" and "app identities match the files"; local development cannot exercise the
  real link opening without a reachable HTTPS domain.
- **Revisit when:** a requirement needs "open after install" or link analytics, or the website moves
  to a different domain (then old links must keep redirecting).

## Follow-ups

| Layer | Follow-up |
|---|---|
| Product spec (`ba-writer`) | `article` product spec cites ADR-0004 for its Share requirement (opens in the app when installed, else the website page) |
| API contract | None |
| Backend spec / code | None |
| Website | Serve `/.well-known/apple-app-site-association` and `/.well-known/assetlinks.json` per environment; keep `/articles/<id>` stable |
| iOS | Associated Domains entitlement per environment; route allowed paths to screens; Share uses the website URL |
| Android | Final `applicationId` (replace `com.example.innerlamp`); verified intent filter per flavor; route allowed paths; Share uses the website URL |
| Infrastructure | Register the domain(s); HTTPS for the website on each environment |

## Open questions

- ⚠️ Not confirmed: which domain the website uses per environment (proposal `innerlamp.vn` for prod; staging/dev unknown) — blocks the association files and both apps' configuration, and the exact URL in every spec.
- ⚠️ Not confirmed: final Android package name (currently the template `com.example.innerlamp`) and the iOS Team ID for `weedigital.vn.InnerLamp` — blocks the content of the association files.

### Resolved

- 2026-10-15 — Shared content links go to the InnerLamp website and open the app when installed (universal links) → yes (anh Đức, article questions Q3).

## Decision record

- 2026-10-15 — Proposed by Claude (`adr-writer`).
