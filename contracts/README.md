# Shared API Contract Fixtures

File JSON di folder `fixtures/` adalah contoh kontrak lintas frontend dan
backend untuk Phase 1B. Seluruh nilainya sintetis dan tidak boleh berisi
credential atau data pelanggan nyata.

Perubahan fixture wajib lulus:

```powershell
npm.cmd run test -- tests/unit/api-contract.test.ts
npm.cmd run backend:test
```

Frontend memvalidasi fixture dengan Zod serta membandingkan content response
dengan `LocalContentSource`. Backend memvalidasi file yang sama terhadap
embedded OpenAPI schema, lalu melakukan decode/encode round-trip menggunakan
generated Go types.

Fixture bukan source data production. Konten Phase 1A tetap berada di folder
`content/`; database importer dan parity data penuh baru dikerjakan pada B3.
