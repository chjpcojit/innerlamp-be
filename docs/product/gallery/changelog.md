# Changelog — gallery product spec

## v1.1 — 2026-10-15

- Reason: anh Đức's answers of 2026-10-15 to the questions raised by the website, iOS and Android
  gallery platform specs written against v1.0 (P1–P6 shared, A1–A4 apps, W1–W7 website). Released as
  a MINOR bump by the product owner's decision, although FR-GAL-008 changes observable behavior:
  platform specs citing v1.0 must re-check FR-GAL-008.
- Changed: FR-GAL-008 — a load failure is always a visible error block, never hidden: the apps' home
  block no longer hides on error and the website home no longer falls back to its sample mosaic (no
  sample mosaic any more); one error copy on every platform, "Không tải được danh sách ảnh. Vui lòng
  thử lại sau." with "Thử lại" where a retry is possible (P1). Scenario "Home block on an app when the
  backend is unreachable" replaced by "Home block when the backend is unreachable" (all clients);
  "Topic fails to load" renamed "… on an app".
- Added to FR-GAL-008: empty gallery → the home block shows "Chưa có ảnh nào." on every client (P4,
  W2); viewer cannot load the next photos → error toast, stay on the photo, moving next retries (P5);
  scenarios "Home block with an empty gallery", "Viewer cannot load the next photos", "Topic fails to
  load on the website". BR-GAL-06 added to its business rules; BR-GAL-02 now points to FR-GAL-008 for
  the empty home block.
- Changed: FR-GAL-007 — save errors are toasts with copy "Chưa có quyền lưu ảnh. Mở Cài đặt để cấp
  quyền." (with an action to open Settings) and "Không lưu được ảnh. Vui lòng thử lại." (P2);
  permission asked only where the device requires it; scenario "Save fails" added, "Permission
  refused" made concrete.
- Clarified: BR-GAL-04, FR-GAL-006 — a photo with no source link still shows its credit, not
  choosable (P6); scenario "Photo without a source link" added.
- Clarified: BR-GAL-05 — the viewer crops to 4:3, never letterboxed (P3); its ⚠️ marker removed.
- Clarified: BR-GAL-09 — confirmed for the English website (W3; the sample mosaic it asked about is gone
  with P1).
- Changed: NFR-GAL-02 — exception for the large tile of the website home mosaic, which may use the
  large picture (W1).
- Added: §2.7 release dependency — the apps' viewer is not released to users until the backend carries
  the credit in every list (Appendix B item 1); no temporary per-photo credit call (A2).
- §3.6: removed the FR-GAL-008 home-block row (sample mosaic / hidden); added NFR-GAL-02 / FR-GAL-001
  website large tile (W1), FR-GAL-002 website subtitle "Những khoảnh khắc an lành từ chùa chiền khắp
  nơi" (W4), FR-GAL-007 iOS permission purpose text "InnerLamp cần quyền thêm ảnh để lưu ảnh bạn chọn
  vào thư viện." (A1), FR-GAL-007 Android saves without a permission so "open Settings" is iOS only
  (A3), FR-GAL-008 website failed topic or page load shows the error block at the newly chosen topic or
  page (W5).
- §4.2 matrix: Website BR-GAL-05, FR-GAL-001, FR-GAL-004, FR-GAL-008 `done` → `drift` (v1.0 statuses
  were wrong, found by the website spec); notes and scenario counts updated for FR-GAL-006, FR-GAL-007,
  FR-GAL-008.
- Resolved: Appendix A items 3 (P3), 4 (P2), 5 (P1). Still open: items 1 (production photo storage
  ADR), 2 (who runs the production load), 6 (content review).
- Not in this spec: A4 (Android album folder) left to the Android spec; W6 (website filter address
  `life` → `meditation` with a permanent redirect) and W7 (closing the lightbox after paging to another
  page navigates to that page) handed to the website spec.
- Vectors unchanged.
- Platforms to follow up: Backend (none in this change; Appendix B item 1 now also gates the apps'
  viewer release), Website (FR-GAL-008 home error and empty block replacing the sample mosaic, "Thử lại",
  viewer next-page error, W4 subtitle, W1 large tile, W6/W7, §4.2 drift rows), iOS (FR-GAL-008 home
  block error and empty states, error copy, viewer next-page error, FR-GAL-007 copy and purpose text,
  P6, A2 release gate), Android (same as iOS minus the permission branch; A4 stays in its spec).

## v1.0 — 2026-10-15

- First product spec for the Buddhist photo gallery, written retroactively from the backend spec
  (`gallery` v1.1, `api-contract.md`, assumption A-001), the iOS `Photos` v1.2 and `PhotoViewer` v1.0
  specs with their assumptions and tech-debt entries, the website and Android code, and the prototype
  (`photos`, viewer block, home section), with anh Đức's answers of 2026-10-15 (Q1–Q14).
- Decisions: no ADR for the feature (production photo storage left to a separate cross-feature ADR);
  one name "Ảnh đẹp Phật giáo"; prototype topic labels and order; credit required on every large
  photo and carried in the lists; save on the apps only; no favourite, wallpaper or share; uniform 4:3
  tiles; viewer pages through the whole filtered gallery; home "+N" counts the whole gallery; public;
  Vietnamese content; content file committed with the backend, back office later.
- IDs: BR-GAL-01…09, FR-GAL-001…009, NFR-GAL-01…03. Vectors: `gallery-order` (7 cases),
  `gallery-home-more` (5 cases).
- Also in this change: `assets/data/buddhist_gallery.json` committed with an exception in `.gitignore`.
- Platforms to follow up: Backend (credit in the lists, loader requires author and licence), Website
  (labels, chip order, lightbox across pages), iOS (name, credit, viewer range, "+N", remove stubs, real
  save), Android (everything; sample data today).
