package trademap

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// ISO 3166-1 alpha-3 to numeric. Numeric codes are only accepted after they
// are confirmed against Trade Map's live country reference data.
const isoPairs = "ABW533 AFG004 AGO024 AIA660 ALA248 ALB008 AND020 ARE784 ARG032 ARM051 ASM016 ATA010 ATF260 ATG028 AUS036 AUT040 AZE031 BDI108 BEL056 BEN204 BES535 BFA854 BGD050 BGR100 BHR048 BHS044 BIH070 BLM652 BLR112 BLZ084 BMU060 BOL068 BRA076 BRB052 BRN096 BTN064 BVT074 BWA072 CAF140 CAN124 CCK166 CHE756 CHL152 CHN156 CIV384 CMR120 COD180 COG178 COK184 COL170 COM174 CPV132 CRI188 CUB192 CUW531 CXR162 CYM136 CYP196 CZE203 DEU276 DJI262 DMA212 DNK208 DOM214 DZA012 ECU218 EGY818 ERI232 ESH732 ESP724 EST233 ETH231 FIN246 FJI242 FLK238 FRA250 FRO234 FSM583 GAB266 GBR826 GEO268 GGY831 GHA288 GIB292 GIN324 GLP312 GMB270 GNB624 GNQ226 GRC300 GRD308 GRL304 GTM320 GUF254 GUM316 GUY328 HKG344 HMD334 HND340 HRV191 HTI332 HUN348 IDN360 IMN833 IND356 IOT086 IRL372 IRN364 IRQ368 ISL352 ISR376 ITA380 JAM388 JEY832 JOR400 JPN392 KAZ398 KEN404 KGZ417 KHM116 KIR296 KNA659 KOR410 KWT414 LAO418 LBN422 LBR430 LBY434 LCA662 LIE438 LKA144 LSO426 LTU440 LUX442 LVA428 MAC446 MAF663 MAR504 MCO492 MDA498 MDG450 MDV462 MEX484 MHL584 MKD807 MLI466 MLT470 MMR104 MNE499 MNG496 MNP580 MOZ508 MRT478 MSR500 MTQ474 MUS480 MWI454 MYS458 MYT175 NAM516 NCL540 NER562 NFK574 NGA566 NIC558 NIU570 NLD528 NOR578 NPL524 NRU520 NZL554 OMN512 PAK586 PAN591 PCN612 PER604 PHL608 PLW585 PNG598 POL616 PRI630 PRK408 PRT620 PRY600 PSE275 PYF258 QAT634 REU638 ROU642 RUS643 RWA646 SAU682 SDN729 SEN686 SGP702 SGS239 SHN654 SJM744 SLB090 SLE694 SLV222 SMR674 SOM706 SPM666 SRB688 SSD728 STP678 SUR740 SVK703 SVN705 SWE752 SWZ748 SXM534 SYC690 SYR760 TCA796 TCD148 TGO768 THA764 TJK762 TKL772 TKM795 TLS626 TON776 TTO780 TUN788 TUR792 TUV798 TWN158 TZA834 UGA800 UKR804 UMI581 URY858 USA840 UZB860 VAT336 VCT670 VEN862 VGB092 VIR850 VNM704 VUT548 WLF876 WSM882 YEM887 ZAF710 ZMB894 ZWE716"

var iso3ToNumeric = func() map[string]string {
	m := map[string]string{"WORLD": "000"}
	for _, pair := range strings.Fields(isoPairs) {
		m[pair[:3]] = pair[3:]
	}
	return m
}()

func normalized(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToUpper(r)
		}
		return -1
	}, s)
}

func resolveEconomy(input string, countries []country, groups []countryGroup) (Selector, error) {
	prefix, raw, hasPrefix := strings.Cut(input, ":")
	if hasPrefix && strings.EqualFold(prefix, "group") {
		for _, g := range groups {
			if raw == strconv.Itoa(g.ID) {
				return Selector{Kind: "group", Code: raw, Label: g.Label}, nil
			}
		}
		return Selector{}, fmt.Errorf("unknown economy group %q", raw)
	}
	want := normalized(input)
	if numeric, ok := iso3ToNumeric[want]; ok {
		want = numeric
	}
	for _, c := range countries {
		if input == c.Code || want == c.Code || want == normalized(c.Label) {
			return Selector{Kind: "economy", Code: c.Code, Label: c.Label}, nil
		}
	}
	for _, g := range groups {
		if input == strconv.Itoa(g.ID) || want == normalized(g.Label) {
			return Selector{Kind: "group", Code: strconv.Itoa(g.ID), Label: g.Label}, nil
		}
	}
	return Selector{}, fmt.Errorf("unknown economy or group %q", input)
}

func resolveProduct(input string, products []product, groups []productGroup) (Selector, error) {
	prefix, raw, hasPrefix := strings.Cut(input, ":")
	if hasPrefix && strings.EqualFold(prefix, "group") {
		for _, g := range groups {
			if raw == strconv.Itoa(g.ID) {
				return Selector{Kind: "group", Code: raw, Label: g.Label}, nil
			}
		}
		return Selector{}, fmt.Errorf("unknown product group %q", raw)
	}
	want := normalized(input)
	for _, p := range products {
		if strings.EqualFold(input, p.Code) || want == normalized(p.Label) || (want == "TOTAL" && p.Code == "ALL") {
			return Selector{Kind: "product", Code: p.Code, Label: p.Label}, nil
		}
	}
	for _, g := range groups {
		if input == strconv.Itoa(g.ID) || want == normalized(g.Label) {
			return Selector{Kind: "group", Code: strconv.Itoa(g.ID), Label: g.Label}, nil
		}
	}
	return Selector{}, fmt.Errorf("unknown product or product group %q", input)
}
