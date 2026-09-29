# Trade Map beta API inventory

Snapshot date: **2026-09-30**  
Frontend build: `main-6VQHBE7Z.js`  
Base URL: `https://www.trademap.org/api`

This is an independent description of the undocumented API used by the current
ITC Trade Map beta frontend. It is not official ITC documentation. Paths,
parameters, access rules, and response shapes can change without notice.

## Coverage and evidence

The inventory was produced by recursively examining all 50 JavaScript chunks
referenced by the current production application. Every call site rooted at the
configured `apiRoot` was collected. Read-only routes were then probed without
cookies or authorization to distinguish public, authenticated, and
parameter-required behavior.

The result is **48 route templates** and **56 method/path operations** used by
this frontend build. CMS, identity, account-management, analytics, static
assets, and ordinary browser routes are intentionally excluded.

Evidence labels used below:

- **Verified**: exercised anonymously and returned a route-specific response.
- **Frontend**: method, path, and parameters were recovered from a live bundle,
  but the operation was not executed (normally because it mutates state).

## General behavior

### Headers

Localized reference and data requests use:

```http
Accept-Language: en
```

The production config currently lists only `en`. When a user is signed in, the
frontend also adds:

```http
Authorization: Bearer <access-token>
```

The verified public GET routes below do not require that header. Custom groups
and saved queries returned `401 Authentication is required` without it.

Company detail calls may include a token returned by `GET /companies`:

```http
X-Public-Companies-Token: <company-public-access-token>
```

Treat this as opaque and short-lived. Do not log or persist it.

### Core enums

| Concept | Values |
|---|---|
| `tradeFlow` | `I` imports, `E` exports, `RE` re-exports, `TB` trade balance |
| `directMirror` | `D` direct, `M` mirror, `X` mixed |
| time-series `indicator` | `VAL`, `QTY`, or `GV` |
| frequency path | `yearly`, `quarterly`, `monthly` |
| output path | Goods: `byProduct`, `byCountry`, `byPartner`; services: `byService`, `byCountry`, `byPartner` |
| `currency` | Frontend currently sends `USD` |
| `sortDir` | `asc` or `desc` |
| coverage `dataType` | `Y`, `Q`, or `M` for yearly, quarterly, or monthly |

Period encoding:

| Frequency | Encoding | Example |
|---|---|---|
| Year | `YYYY` | `2025` |
| Quarter | `YYYYQQ` | `202501` for Q1 2025 |
| Month | `YYYYMM` | `202509` for September 2025 |

### Entity selection

The frontend sends exactly one member of each applicable pair:

| Individual | Group |
|---|---|
| `country` | `countryGrp` |
| `partner` | `partnerGrp` |
| `product` | `productGrp` |

Country codes are three-digit M49 strings such as `792`; `000` means World.
Goods products use HS codes or `ALL`. Service requests use EBOPS codes, with
the frontend translating its `S00` UI value to `ALL` for API requests.

### Pagination and sorting

Data-list endpoints generally accept:

| Parameter | Meaning |
|---|---|
| `page` | One-based page number |
| `pageSize` | Requested rows per page |
| `sortBy` | Field, indicator code, or period used for sorting |
| `sortDir` | `asc` or `desc` |

The common paginated response envelope is:

```json
{
  "nbRecords": 177,
  "page": 1,
  "nbRecordPerPage": 500,
  "nbPages": 1,
  "records": [],
  "aggregateRecords": [],
  "sources": []
}
```

Not every endpoint includes every field.

### Export mode

The frontend exports tables by repeating the same data request with:

```text
export=csv
export=excel
```

`csv` is handled as a downloadable blob. Despite its name, `excel` returns
text which the frontend parses as CSV and converts to an `.xlsx` workbook
locally. Export is not a separate API route.

## Complete endpoint index

### Countries, products, services, and groups

| Method | Path | Access | Purpose | Evidence |
|---|---|---|---|---|
| GET | `/countries` | Public | Country list and coverage periods | Verified |
| GET | `/countries/groups/generic` | Public | Built-in country groups | Verified |
| GET | `/countries/groups/custom` | Bearer | Current user's country groups | Verified |
| POST | `/countries/groups/custom` | Bearer | Create a country group | Frontend |
| PUT | `/countries/groups/custom` | Bearer | Update a country group | Frontend |
| DELETE | `/countries/groups/custom` | Bearer | Delete a country group | Frontend |
| GET | `/products/HS` | Public | HS hierarchy | Verified |
| GET | `/products/NTL` | Public | National tariff-line product lookup | Verified |
| GET | `/products/SIC` | Public | SIC products used by company search | Verified |
| GET | `/products/KOMPASS` | Public | Kompass products used by company search | Verified |
| GET | `/products/search/NTL` | Public | National tariff-line text/filter search | Verified |
| GET | `/products/groups/generic` | Public | Built-in product groups | Verified |
| GET | `/products/groups/custom` | Bearer | Current user's product groups | Verified |
| POST | `/products/groups/custom` | Bearer | Create a product group | Frontend |
| PUT | `/products/groups/custom` | Bearer | Update a product group | Frontend |
| DELETE | `/products/groups/custom` | Bearer | Delete a product group | Frontend |
| GET | `/services/EBOPS` | Public | EBOPS service hierarchy | Verified |

Reference parameters:

| Endpoint | Parameters |
|---|---|
| `/countries/groups/generic` | `loadMembers` boolean |
| `/countries/groups/custom` GET | `loadMembers=true`; frontend also adds cache-busting `counter` |
| `/countries/groups/custom` DELETE | `countryGroupId` |
| `/products/NTL` | `product`, `country` |
| `/products/search/NTL` | `searchTerms`; optional `hs`, `countries` serialized lists |
| `/products/groups/generic` | `loadProducts` boolean |
| `/products/groups/custom` GET | `loadProducts=true`; frontend also adds cache-busting `counter` |
| `/products/groups/custom` DELETE | `productGroupId` |

Custom country-group request body:

```json
{
  "id": 0,
  "label": "Group name",
  "countryCodes": ["008", "070"]
}
```

Custom product-group request body:

```json
{
  "id": 0,
  "label": "Group name",
  "productCodes": ["0101", "0102"],
  "level": 4
}
```

Create uses `id: 0`; update sends the existing ID.

Reference response highlights:

- `/countries`: `countryCd`, `label`, `nes`, `ti`, and first/last periods for
  yearly HS/NTL, quarterly, monthly, and services datasets.
- `/products/HS`: `productCd`, `label`, `revisions`.
- `/services/EBOPS`: `productCd`, `label`, `displayCd`, `maxLevel`.
- Generic groups: `id`, `label`, type/level metadata, plus members/products
  when requested.

### Goods trade indicators

| Method | Path | Access | Evidence |
|---|---|---|---|
| GET | `/goods/tradeIndicators/availableIndicators` | Public | Verified |
| GET | `/goods/tradeIndicators/byProduct` | Public | Verified |
| GET | `/goods/tradeIndicators/byCountry` | Public | Verified |
| GET | `/goods/tradeIndicators/byPartner` | Public | Verified |

Common selection parameters are `tradeFlow`, one country selector, one partner
selector, and one product selector. Additional parameters:

| Parameter | Applies to | Notes |
|---|---|---|
| `output` | available-indicators | `byProduct`, `byCountry`, or `byPartner`; required |
| `indicators` | data endpoints | Comma-separated codes; required |
| `directMirror` | data endpoints | Required for `byCountry` and `byPartner`; frontend supplies it to all three |
| `hsLevel` | `byProduct` | `2`, `4`, `6`, or `10` |
| `page`, `pageSize` | data endpoints | Pagination |
| `sortBy`, `sortDir` | data endpoints | Labels/codes or an indicator code |

Indicator codes present in the frontend build:

| Code | Meaning/category inferred by UI |
|---|---|
| `VAL` | Trade value |
| `BAL` | Trade balance |
| `QTY` | Quantity |
| `UV` | Unit value |
| `S`, `SPC`, `SW`, `SPW` | Share indicators |
| `RK`, `RKP` | Ranking indicators |
| `GV5`, `GV2`, `GV5W`, `GV5P`, `GV2P` | Value growth indicators |
| `GQ5`, `GQ2` | Quantity growth indicators |
| `POT` | Potential indicator |
| `TAR` | Tariff indicator |
| `DIST`, `DISTP` | Distance indicators |
| `CON`, `CONP` | Concentration indicators |

`availableIndicators` returns:

```json
{
  "refYear": 2025,
  "indicators": [
    {
      "code": "VAL",
      "category": "standard",
      "subCategory": null,
      "isAvailable": true,
      "reasonCodes": []
    }
  ]
}
```

Data endpoints return the paginated envelope plus `refYear`. Each record has
`countryCd` or `reporterCd`, `partnerCd`, `productCd`, and a `data` array of
`{indicatorCd, value, flag}`.

### Goods time series

All operations are public `GET` routes verified to exist.

| Frequency | By product | By country | By partner |
|---|---|---|---|
| Yearly | `/goods/timeSeries/yearly/byProduct` | `/goods/timeSeries/yearly/byCountry` | `/goods/timeSeries/yearly/byPartner` |
| Quarterly | `/goods/timeSeries/quarterly/byProduct` | `/goods/timeSeries/quarterly/byCountry` | `/goods/timeSeries/quarterly/byPartner` |
| Monthly | `/goods/timeSeries/monthly/byProduct` | `/goods/timeSeries/monthly/byCountry` | `/goods/timeSeries/monthly/byPartner` |

Parameters:

| Parameter | Required/conditional |
|---|---|
| `tradeFlow` | Required |
| `country` or `countryGrp` | Selection-dependent |
| `partner` or `partnerGrp` | Selection-dependent |
| `product` or `productGrp` | Selection-dependent |
| `periodFrom`, `periodTo` | Frontend always sends both |
| `directMirror` | Required |
| `indicator` | Required: `VAL`, `QTY`, or `GV` |
| `currency` | Sent for `VAL` |
| `hsLevel` | Sent for `byProduct`: `2`, `4`, `6`, or `10` |
| `page`, `pageSize` | Pagination |
| `sortBy`, `sortDir` | Sorting |
| `export` | Optional `csv` or `excel` |

Response records have this shape:

```json
{
  "reporterCd": "792",
  "partnerCd": "000",
  "productCd": "87",
  "data": [
    { "period": 2024, "value": 32439511, "flag": null }
  ]
}
```

The envelope includes `records`, `aggregateRecords`, and `sources`.

### Services time series

All operations are public `GET` routes verified to exist.

| Frequency | By service | By country | By partner |
|---|---|---|---|
| Yearly | `/services/timeSeries/yearly/byService` | `/services/timeSeries/yearly/byCountry` | `/services/timeSeries/yearly/byPartner` |
| Quarterly | `/services/timeSeries/quarterly/byService` | `/services/timeSeries/quarterly/byCountry` | `/services/timeSeries/quarterly/byPartner` |

No monthly services route appears in this frontend build.

Parameters:

| Parameter | Required/conditional |
|---|---|
| `service` | Required; EBOPS code or `ALL` |
| `tradeFlow` | Required |
| `country` or `countryGrp` | Selection-dependent |
| `partner` or `partnerGrp` | Selection-dependent |
| `periodFrom`, `periodTo` | Frontend always sends both |
| `bpmLevel` | Sent for `byService`; frontend uses hierarchy depth |
| `currency` | Frontend sends `USD` |
| `page`, `pageSize` | Pagination |
| `sortBy`, `sortDir` | Sorting |
| `export` | Optional `csv` or `excel` |

The response uses the same paginated time-series envelope as goods. The service
code is returned in `productCd` (for example `S00` for total services).

### Data coverage

| Method | Path | Access | Evidence |
|---|---|---|---|
| GET | `/coverage/goods/latest` | Public | Verified |
| GET | `/coverage/goods/summary` | Public | Verified |
| GET | `/coverage/goods/details` | Public | Verified |
| GET | `/coverage/services/latest` | Public | Verified |
| GET | `/coverage/services/summary` | Public | Verified |
| GET | `/coverage/services/details` | Public | Verified |
| GET | `/coverage/companies/summary` | Public | Verified |
| GET | `/coverage/companies/details` | Public | Verified |

Goods summary/details parameters:

- `output`: `byProduct`, `byCountry`, or `byPartner`
- `dataType`: `Y`, `Q`, or `M`
- `periodFrom`, `periodTo`

Services summary/details parameters:

- `dataType`: `Y` or `Q`
- `periodFrom`, `periodTo`

`latest` returns entries shaped as:

```json
{ "dataType": "Y", "nbCountries": 123, "latestPeriod": 2025 }
```

Summary responses are arrays containing `period`, `nbCountries`, and—where
applicable—`globalMarketShare`. Detail responses are country arrays containing
period-by-period reporting status, reporting level, and sources.

Company coverage summary returns aggregate company/importer/exporter counts.
Company coverage details returns per-country availability and counts.

### Companies

| Method | Path | Access | Purpose | Evidence |
|---|---|---|---|---|
| GET | `/companies` | Public | Search companies | Verified |
| GET | `/companies/products` | Public/token-assisted | Products traded by a company | Verified |
| GET | `/companies/partners` | Public/token-assisted | Trading partners for a company | Verified |
| GET | `/companies/contact` | Public/token-assisted | Company contact/profile detail | Verified |

`GET /companies` parameters:

| Parameter | Notes |
|---|---|
| `tradeFlow` | Required |
| `country` or `countryGrp` | Company location |
| `product` | Product code |
| `productType` | Frontend types: `p`, `psic`, `pk` |
| `page`, `pageSize` | Pagination |
| `sortBy`, `sortDir` | Frontend defaults to `companyName`, `asc` |

The frontend prevents the unbounded `country=000` plus `product=ALL` query.

Company-detail parameters:

| Endpoint | Parameters |
|---|---|
| `/companies/products` | Required `companyId`; frontend also sends `sourceId`, `country`, `tradeFlow` |
| `/companies/partners` | Required `companyId`; frontend also sends `sourceId` |
| `/companies/contact` | Required `companyId`; frontend also sends `sourceId` |

Search responses contain company records and `sources`. A record includes `id`,
`name`, `city`, `countryCd`, `activities`, `website`, turnover/employee fields,
`sourceId`, and an opaque `publicAccessToken`. The frontend forwards that token
through `X-Public-Companies-Token` for the three detail calls.

### Saved queries

All operations require bearer authentication.

| Method | Path | Purpose | Evidence |
|---|---|---|---|
| GET | `/User/saved-queries` | List current user's saved queries | Verified |
| POST | `/User/saved-queries` | Create a saved query | Frontend |
| PUT | `/User/saved-queries` | Update a saved query | Frontend |
| DELETE | `/User/saved-queries/{id}` | Delete a saved query | Frontend |

The frontend may add a cache-busting `counter` to GET. Create sends:

```json
{
  "module": "goods",
  "section": "timeSeries",
  "path": "/goods/time-series/...",
  "name": "Saved query name",
  "filtersNames": "Exports, Türkiye, World, All products",
  "tradeFlow": "E",
  "country": "792",
  "partner": "000",
  "productOrService": "ALL"
}
```

Update sends the existing object after changing its fields, including `id`.

### CAPTCHA and email support

These routes share the Trade Map API root but are not trade-data operations.

| Method | Path | Access | Purpose | Evidence |
|---|---|---|---|---|
| GET | `/captcha/challenge` | Public | Create an ALTCHA proof-of-work challenge | Verified |
| POST | `/captcha/validate` | Public/challenge | Validate an ALTCHA solution | Verified method via `Allow` |
| POST | `/emailing/contact-us` | Public/form validation | Submit contact form | Frontend; method verified via `Allow` |
| POST | `/emailing/report-bug` | Public/form validation | Submit bug report | Frontend; method verified via `Allow` |

The challenge response contains `algorithm`, `challenge`, `salt`, `signature`,
and `maxnumber`. Email bodies are generated from the corresponding UI forms and
should not be considered stable public contracts.

## Working examples

### World exports by reporting country

```sh
curl --get 'https://www.trademap.org/api/goods/timeSeries/yearly/byCountry' \
  --data-urlencode 'country=000' \
  --data-urlencode 'partner=000' \
  --data-urlencode 'product=ALL' \
  --data-urlencode 'tradeFlow=E' \
  --data-urlencode 'periodFrom=2021' \
  --data-urlencode 'periodTo=2025' \
  --data-urlencode 'directMirror=D' \
  --data-urlencode 'indicator=VAL' \
  --data-urlencode 'currency=USD' \
  --data-urlencode 'page=1' \
  --data-urlencode 'pageSize=500' \
  --data-urlencode 'sortBy=2025' \
  --data-urlencode 'sortDir=desc'
```

### Available indicators

```sh
curl --get 'https://www.trademap.org/api/goods/tradeIndicators/availableIndicators' \
  --data-urlencode 'output=byCountry' \
  --data-urlencode 'country=000' \
  --data-urlencode 'partner=000' \
  --data-urlencode 'product=ALL' \
  --data-urlencode 'tradeFlow=E'
```

## Implementation guidance for `amartrade`

- Model only the trade-data and reference routes needed by a command; do not
  expose a raw arbitrary-path request command.
- Use a default base URL that can be overridden in tests.
- Send `Accept-Language: en`, but do not send browser-only headers.
- Use finite request timeouts and a descriptive user agent.
- Retry only transient failures (`429`, `502`, `503`, `504`) with bounded
  exponential backoff and jitter.
- Respect `Retry-After` when present.
- Validate enum combinations before making a request.
- Decode provider error bodies and preserve the HTTP status in returned errors.
- Never accept access tokens as ordinary command flags; if authenticated
  features are added later, read credentials from a protected environment or
  credential store.
- Redact `publicAccessToken` from logs and fixture files.
- Keep recorded fixtures small and sanitized.

