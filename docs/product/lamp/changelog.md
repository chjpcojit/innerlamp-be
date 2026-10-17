# Changelog — lamp product spec

## v1.0 — 2026-10-16

- First product spec for Thắp đèn and Vườn đèn (retroactive: backend done, website connected to the
  backend, iOS local-only, Android not built). Replaces the backend SRS v0.4 and the iOS SRS.
- Added: BR-LMP-01 … 14, FR-LMP-001 … 014, NFR-LMP-01 … 03, vectors `lamp-lifetime`,
  `lamp-remaining-label`, `lamp-input-validation`, `lamp-lighter-name`.
- Decisions of 2026-10-16 (all 14 defaults confirmed): recipient optional (BR-LMP-06); hide-name switch
  and a visibility notice on every platform; own lamps marked by the backend, label "Của bạn"
  (FR-LMP-008); remaining time by the server end moment with "Còn dưới 1 ngày" (BR-LMP-12); center
  navigation button opens Thắp đèn; paid temple lamp hidden (FR-LMP-014); no sample lamps or base
  counts; iOS local lamps dropped.
- Platforms to follow up: Backend (recipient optional, own-lamp flag), Website (drift list in §4.2,
  remove the offering link), iOS (connect to the service, drop the lamps stored on the device), Android (build).
