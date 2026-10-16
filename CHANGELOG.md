# Changelog

## 0.2.0 (unreleased)

- Added: Saudi Arabia support via `payriff.WithRegion(payriff.RegionSaudi)`. It uses
  `https://api.gccpay.sa` and defaults the currency to `SAR`. Azerbaijan (`AZN`) stays the default.
- Removed: direct (host-to-host) card payments: `Payments.DirectPay()`, `DirectPayParams`, `Card`
  and the `WithCardEncryptionKey` option. Use the hosted payment page (`Orders.Create()`) instead.
- Removed `payriff.ProductionURL`; use `payriff.RegionAzerbaijan.BaseURL()`.

## 0.1.0 (unreleased)

- Initial release: orders, direct and saved-card payments, saved cards, transactions, payouts,
  v2 invoices, order callback parsing.