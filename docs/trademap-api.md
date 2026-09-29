# Trade Map data API field guide

Snapshot date: **2026-09-30**  
Base URL: `https://www.trademap.org/api`

This document explains the public data API used by the current Trade Map beta
application. The API is undocumented and may change. The details below were
derived from the production frontend, public reference endpoints, validation
responses, and successful anonymous data requests.

The goal is practical: describe the identifiers, dimensions, parameters, and
responses needed to build `amartrade`. Account management, CAPTCHA, email,
saved queries, CMS routes, and Swagger infrastructure are out of scope.

## Mental model

A trade query selects:

1. a reporter economy or reporter economy group;
2. a partner economy or partner economy group;
3. a product, product group, or service;
4. imports or exports;
5. a period or period range; and
6. the dimension to expand in the result.

The final dimension is encoded in the endpoint path:

| Output | Meaning |
|---|---|
| `byCountry` | Vary reporter economies; hold partner and product/service fixed |
| `byPartner` | Vary partner economies; hold reporter and product/service fixed |
| `byProduct` | Vary HS products; hold reporter and partner fixed |
| `byService` | Vary EBOPS services; hold reporter and partner fixed |

For example, `country=000&partner=000&product=ALL` sent to a `byCountry`
endpoint lists reporting economies. The same selectors sent to `byProduct`
would instead list products.

## Economies

### Discovering economy codes

```http
GET /countries
```

This is the authoritative lookup. The current response contains 254 entries.
Each `countryCd` is exactly three decimal characters:

```json
{
  "countryCd": "792",
  "label": "Türkiye",
  "nes": false,
  "ti": true,
  "yearly246": { "firstPeriod": 2001, "lastPeriod": 2025 },
  "yearly10D": { "firstPeriod": 2001, "lastPeriod": 2025 },
  "yearlyReexport": { "firstPeriod": 0, "lastPeriod": 0 },
  "yearlyServices": { "firstPeriod": 2000, "lastPeriod": 2024 },
  "quarterly": { "firstPeriod": 200201, "lastPeriod": 202602 },
  "quarterlyServices": { "firstPeriod": 200501, "lastPeriod": 202304 },
  "monthly": { "firstPeriod": 200201, "lastPeriod": 202607 }
}
```

The exact periods vary by economy and over time.

Treat economy codes as strings, never integers. Leading zeros are meaningful:

| Code | Label |
|---|---|
| `000` | World |
| `004` | Afghanistan |
| `251` | France |
| `276` | Germany |
| `792` | Türkiye |

These are Trade Map/UN Comtrade area codes. They often resemble UN M49 codes,
but they are not safely interchangeable with ISO numeric country codes. Always
resolve them from `/countries`.

### World

`000` is the special World economy code. It is sent through the normal
individual-economy parameters:

```text
country=000
partner=000
```

It is not an economy-group ID.

### NES and other statistical areas

The list contains ordinary economies and statistical areas. Entries with
`nes: true` mean “not elsewhere specified”, for example:

```json
{ "countryCd": "492", "label": "European Union Nes", "nes": true }
```

These codes can appear in trade results and group membership. They must not be
discarded during decoding. A caller may choose to hide them in presentation,
but doing so changes totals and must be explicit.

### Availability fields

The country response tells us whether a query is meaningful before we call a
data endpoint:

| Field | Dataset |
|---|---|
| `ti` | Trade indicators available |
| `yearly246` | Yearly HS 2/4/6-digit goods |
| `yearly10D` | Yearly national tariff-line goods |
| `yearlyReexport` | Yearly re-export data |
| `yearlyServices` | Yearly services |
| `quarterly` | Quarterly goods |
| `quarterlyServices` | Quarterly services |
| `monthly` | Monthly goods |

A period object contains `firstPeriod`, `lastPeriod`, and sometimes `level`.
Zero boundaries mean no coverage.

## Economy groups

### Discovering groups and membership

```http
GET /countries/groups/generic?loadMembers=true
```

The current response contains 54 groups. A group looks like:

```json
{
  "id": 42,
  "label": "EU 27",
  "type": "economicGrouping",
  "note": "European Union",
  "members": [
    { "countryCd": "040", "label": "Austria" },
    { "countryCd": "056", "label": "Belgium" }
  ]
}
```

Current group types are:

| Type | Current count |
|---|---:|
| `economicGrouping` | 34 |
| `geographicRegion` | 9 |
| `customsUnion` | 5 |
| `intergovernmentalOrganisation` | 4 |
| `geopoliticalRegion` | 1 |
| `unilateralTradeArrangement` | 1 |

### Encoding a group in a query

Group IDs are opaque, variable-length integers. Examples include:

| ID | Group |
|---:|---|
| `7` | Africa |
| `24` | ASEAN |
| `42` | EU 27 |
| `2227` | WTO |
| `6578` | OECD |
| `6757` | BRICS |

Do not pad group IDs to three digits and do not send them through `country` or
`partner`. Choose exactly one selector from each row:

| Dimension | Individual economy | Economy group |
|---|---|---|
| Reporter | `country=792` | `countryGrp=42` |
| Partner | `partner=276` | `partnerGrp=42` |

Never send both members of a pair. Resolve IDs at runtime instead of embedding
them permanently; labels and memberships may change.

Membership is literal API data. It can include NES areas. For example, the
current EU 27 group contains 27 ordinary members plus `492 European Union Nes`,
so its returned membership count is 28.

### Group result behavior

For a grouped reporter query:

```text
countryGrp=42&partner=000&product=ALL
```

`records` contains individual group members:

```json
{ "reporterCd": "276", "partnerCd": "000", "productCd": "TOTAL" }
```

`aggregateRecords` contains the group total and uses the unpadded group ID:

```json
{ "reporterCd": "42", "partnerCd": "000", "productCd": "TOTAL" }
```

For `partnerGrp=42`, the same rule applies to `partnerCd`. A decoder therefore
needs dimension context to distinguish economy codes from group IDs.

## Goods products

### HS products

```http
GET /products/HS
```

The current lookup contains:

- `ALL` for all products;
- 2-digit HS chapters;
- 4-digit HS headings; and
- 6-digit HS subheadings.

Codes are strings and must preserve leading zeros:

```json
{ "productCd": "01", "label": "Live animals", "revisions": "111111" }
```

Use:

```text
product=ALL
product=01
product=0101
product=010121
```

The `revisions` value is an opaque API availability marker. We should retain it
but not interpret its character positions until that mapping is independently
verified.

`hsLevel` controls the product depth returned by `byProduct`:

```text
hsLevel=2
hsLevel=4
hsLevel=6
hsLevel=10
```

Level 10 is for national tariff-line data and is not populated for every
reporter.

### National tariff-line products

Look up a known tariff-line code for a reporting economy:

```http
GET /products/NTL?country=792&product=<code>
```

Search tariff-line descriptions with:

```http
GET /products/search/NTL?searchTerms=<text>
```

Optional filters are `hs` and `countries`. NTL codes are country-specific;
never resolve or validate one without its reporter economy.

### Product groups

```http
GET /products/groups/generic?loadProducts=true
```

The current response contains 18 groups. IDs are opaque integers, for example:

```json
{
  "id": 18,
  "label": "Vehicles",
  "level": 6,
  "products": []
}
```

Use `productGrp`, not `product`:

```text
productGrp=18
```

When expanded with `byProduct&hsLevel=6`, `records` contains the member HS
codes and `aggregateRecords[].productCd` contains the group ID as a string:

```json
{ "productCd": "18" }
```

As with economy groups, decode that aggregate code using query context rather
than treating it as an HS chapter.

## Services

```http
GET /services/EBOPS
```

The response currently has 150 EBOPS hierarchy entries:

```json
{
  "productCd": "S03001001",
  "label": "Passenger transport, Sea",
  "displayCd": "3.1.1",
  "maxLevel": "4"
}
```

Fields:

| Field | Use |
|---|---|
| `productCd` | Machine identifier sent as `service` |
| `displayCd` | Human-readable EBOPS code |
| `label` | Human-readable description |
| `maxLevel` | Deepest hierarchy level reported for the branch |

The special lookup entry is:

```json
{ "productCd": "S00", "label": "All services", "displayCd": "S" }
```

The frontend sends it to data routes as:

```text
service=ALL
```

Other service codes have lengths 3, 6, 9, 12, or 15. For `byService`, the
frontend sends that desired code length through the parameter named
`bpmLevel`, despite the lookup being described as EBOPS.

## Shared query values

### Trade flow

| Value | Meaning |
|---|---|
| `I` | Imports |
| `E` | Exports |
| `RE` | Re-exports |
| `TB` | Trade balance |

Availability depends on the endpoint and selected economies. Imports and
exports are the normal time-series values. Re-exports depend on
`yearlyReexport` coverage. Trade balance is primarily used with indicators.

### Direct and mirror data

| Value | Meaning |
|---|---|
| `D` | Direct/reporter data |
| `M` | Mirror/partner-reported data |
| `X` | Mixed data |

Send this as `directMirror`. The frontend selects mirror data when a reporter
does not cover the requested product depth or period.

### Periods

| Frequency | Path | Encoding | Example |
|---|---|---|---|
| Year | `yearly` | `YYYY` | `2025` |
| Quarter | `quarterly` | `YYYYQQ` | `202501` = Q1 2025 |
| Month | `monthly` | `YYYYMM` | `202509` = September 2025 |

Both `periodFrom` and `periodTo` are inclusive. Use `/countries` and coverage
endpoints to avoid requesting unavailable periods.

### Pagination and sorting

| Parameter | Meaning |
|---|---|
| `page` | One-based page number |
| `pageSize` | Requested rows per page |
| `sortBy` | A label/code, indicator code, or period |
| `sortDir` | `asc` or `desc` |

Do not assume `pageSize` is honored exactly. One observed company query asked
for one record but reported `nbRecordPerPage: 3`.

### Currency and units

The frontend currently sends `currency=USD`. `VAL` and `BAL` values are in
thousands of the selected currency; a returned value of `1000` therefore means
USD 1,000,000 when `currency=USD`.

Time-series data points have:

```json
{ "period": 2025, "value": 273254542, "unit": null, "flag": null }
```

`unit` is relevant to quantity responses. `flag` contains provider quality or
estimation markers and should be preserved as opaque text until its legend is
documented.

## Goods time series

All routes are anonymous `GET` requests:

| Frequency | By product | By country | By partner |
|---|---|---|---|
| Yearly | `/goods/timeSeries/yearly/byProduct` | `/goods/timeSeries/yearly/byCountry` | `/goods/timeSeries/yearly/byPartner` |
| Quarterly | `/goods/timeSeries/quarterly/byProduct` | `/goods/timeSeries/quarterly/byCountry` | `/goods/timeSeries/quarterly/byPartner` |
| Monthly | `/goods/timeSeries/monthly/byProduct` | `/goods/timeSeries/monthly/byCountry` | `/goods/timeSeries/monthly/byPartner` |

Parameters:

| Parameter | Rule |
|---|---|
| `tradeFlow` | Required |
| `country` or `countryGrp` | Exactly one reporter selector |
| `partner` or `partnerGrp` | Exactly one partner selector |
| `product` or `productGrp` | Exactly one product selector |
| `periodFrom`, `periodTo` | Inclusive encoded periods |
| `directMirror` | Required |
| `indicator` | Required: `VAL`, `QTY`, or `GV` |
| `currency` | Send for `VAL`; normally `USD` |
| `hsLevel` | Send for `byProduct` |
| pagination/sorting | `page`, `pageSize`, `sortBy`, `sortDir` |

Response envelope:

```json
{
  "nbRecords": 97,
  "page": 1,
  "nbRecordPerPage": 10,
  "nbPages": 10,
  "records": [],
  "aggregateRecords": [],
  "sources": []
}
```

A record is one reporter/partner/product tuple:

```json
{
  "reporterCd": "792",
  "partnerCd": "000",
  "productCd": "87",
  "reporterLabel": null,
  "partnerLabel": null,
  "productLabel": null,
  "canNavigateProductBelow": true,
  "canNavigateCountry": false,
  "data": [
    { "period": 2024, "value": 32439511, "unit": null, "flag": null }
  ]
}
```

Labels are often absent. Join the codes against the reference endpoints.

## Services time series

| Frequency | By service | By country | By partner |
|---|---|---|---|
| Yearly | `/services/timeSeries/yearly/byService` | `/services/timeSeries/yearly/byCountry` | `/services/timeSeries/yearly/byPartner` |
| Quarterly | `/services/timeSeries/quarterly/byService` | `/services/timeSeries/quarterly/byCountry` | `/services/timeSeries/quarterly/byPartner` |

There is no monthly services route in the current frontend.

Parameters:

| Parameter | Rule |
|---|---|
| `service` | Required; `ALL` or an EBOPS `productCd` |
| `tradeFlow` | Required |
| `country` or `countryGrp` | Exactly one reporter selector |
| `partner` or `partnerGrp` | Selection depends on output dimension |
| `periodFrom`, `periodTo` | Inclusive encoded periods |
| `bpmLevel` | Send for `byService` |
| `currency` | Frontend sends `USD` |
| pagination/sorting | `page`, `pageSize`, `sortBy`, `sortDir` |

The response uses the goods time-series envelope. The service code is returned
in `productCd`; total services is returned as `S00`.

## Goods trade indicators

First ask which indicators are valid for the selected query:

```http
GET /goods/tradeIndicators/availableIndicators
```

Parameters are the reporter, partner, and product selectors plus:

```text
tradeFlow=E
output=byCountry
```

Response:

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

Do not hard-code indicator availability. It changes with reporter, partner,
product, flow, and output dimension.

Fetch values from:

```text
GET /goods/tradeIndicators/byProduct
GET /goods/tradeIndicators/byCountry
GET /goods/tradeIndicators/byPartner
```

Additional parameters:

| Parameter | Rule |
|---|---|
| `indicators` | Required comma-separated codes, such as `VAL,BAL,GV5` |
| `directMirror` | Frontend sends `D`, `M`, or `X` |
| `hsLevel` | Send for `byProduct` |
| pagination/sorting | `page`, `pageSize`, `sortBy`, `sortDir` |

Codes currently exposed by the frontend:

```text
VAL BAL QTY UV S SPC SW SPW RK RKP
GV5 GV2 GV5W GV5P GV2P GQ5 GQ2
POT TAR DIST DISTP CON CONP
```

Known units from the frontend are:

- `VAL`, `BAL`: thousands of the selected currency;
- share and growth indicators: percent;
- `DIST`, `DISTP`: kilometres;
- rankings and concentration indicators: unitless.

Preserve indicator codes in the result. We should not assign expanded names to
the less obvious codes until their definitions are verified.

## Coverage endpoints

Use these to discover current availability instead of assuming the latest year:

```text
GET /coverage/goods/latest
GET /coverage/goods/summary
GET /coverage/goods/details
GET /coverage/services/latest
GET /coverage/services/summary
GET /coverage/services/details
```

`latest` needs no parameters and returns:

```json
{ "dataType": "Y", "nbCountries": 123, "latestPeriod": 2025 }
```

Summary/details parameters:

| Parameter | Goods | Services |
|---|---|---|
| `dataType` | `Y`, `Q`, or `M` | `Y` or `Q` |
| `periodFrom`, `periodTo` | Required | Required |
| `output` | `byProduct`, `byCountry`, `byPartner` | Not sent |

Summary responses contain `period`, `nbCountries`, and where applicable
`globalMarketShare`. Detail responses contain each economy's reporting status,
reporting level, and sources by period.

## Company data

These public endpoints are separate from the statistical trade series:

```text
GET /companies
GET /companies/products
GET /companies/partners
GET /companies/contact
GET /coverage/companies/summary
GET /coverage/companies/details
```

Company search accepts `tradeFlow`, an economy or group selector, `product`,
`productType`, pagination, and sorting. Product types used by the frontend are:

| Value | Nomenclature |
|---|---|
| `p` | HS |
| `psic` | SIC |
| `pk` | Kompass |

Discover the latter classifications with:

```text
GET /products/SIC
GET /products/KOMPASS
```

Company records return an opaque `publicAccessToken`. Forward it only to that
company's detail calls as `X-Public-Companies-Token`; do not log or persist it.
Detail calls require `companyId` and normally `sourceId`.

## Exporting complete tables

The same time-series or indicator URL can request an export:

```text
export=csv
export=excel
```

`csv` returns a downloadable blob. `excel` currently returns CSV-like text;
the web frontend converts that text into XLSX locally. It is not a native XLSX
response.

## Complete examples

### World exports by reporter

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

### EU members' exports to the world

EU 27 currently has group ID `42`:

```sh
curl --get 'https://www.trademap.org/api/goods/timeSeries/yearly/byCountry' \
  --data-urlencode 'countryGrp=42' \
  --data-urlencode 'partner=000' \
  --data-urlencode 'product=ALL' \
  --data-urlencode 'tradeFlow=E' \
  --data-urlencode 'periodFrom=2024' \
  --data-urlencode 'periodTo=2025' \
  --data-urlencode 'directMirror=D' \
  --data-urlencode 'indicator=VAL' \
  --data-urlencode 'currency=USD' \
  --data-urlencode 'page=1' \
  --data-urlencode 'pageSize=500'
```

The member economies are in `records`; the EU aggregate is in
`aggregateRecords` with `reporterCd: "42"`.

### Türkiye's exports to EU members

```sh
curl --get 'https://www.trademap.org/api/goods/timeSeries/yearly/byPartner' \
  --data-urlencode 'country=792' \
  --data-urlencode 'partnerGrp=42' \
  --data-urlencode 'product=ALL' \
  --data-urlencode 'tradeFlow=E' \
  --data-urlencode 'periodFrom=2024' \
  --data-urlencode 'periodTo=2025' \
  --data-urlencode 'directMirror=D' \
  --data-urlencode 'indicator=VAL' \
  --data-urlencode 'currency=USD' \
  --data-urlencode 'page=1' \
  --data-urlencode 'pageSize=500'
```

### Türkiye's vehicle exports by HS6 product

Vehicles currently has product-group ID `18`:

```sh
curl --get 'https://www.trademap.org/api/goods/timeSeries/yearly/byProduct' \
  --data-urlencode 'country=792' \
  --data-urlencode 'partner=000' \
  --data-urlencode 'productGrp=18' \
  --data-urlencode 'tradeFlow=E' \
  --data-urlencode 'periodFrom=2024' \
  --data-urlencode 'periodTo=2025' \
  --data-urlencode 'directMirror=D' \
  --data-urlencode 'indicator=VAL' \
  --data-urlencode 'hsLevel=6' \
  --data-urlencode 'currency=USD' \
  --data-urlencode 'page=1' \
  --data-urlencode 'pageSize=500'
```

## Client rules for `amartrade`

- Fetch reference data instead of hard-coding economy or group IDs.
- Keep all codes as strings, including numeric-looking codes.
- Model individual and group selectors as mutually exclusive types.
- Preserve NES economies, flags, units, sources, and aggregate records.
- Join missing labels locally from reference datasets.
- Validate periods against the selected economy's coverage.
- Ask `availableIndicators` before constructing an indicator query.
- Distinguish member records from aggregate records.
- Use finite timeouts and bounded retries for `429`, `502`, `503`, and `504`.
- Respect `Retry-After` and avoid concurrent bulk scraping.
- Redact company `publicAccessToken` values from logs and fixtures.
