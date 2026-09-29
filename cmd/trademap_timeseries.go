package cmd

import (
	"fmt"

	"github.com/amar-jay/amartrade/internal/trademap/timeseries"
	"github.com/spf13/cobra"
)

func newGoodsCommand() *cobra.Command {
	command := &cobra.Command{Use: "goods", Short: "Query Trade Map goods data"}
	series := newTimeSeriesCommand(true)
	command.AddCommand(series)
	return command
}

func newServicesCommand() *cobra.Command {
	command := &cobra.Command{Use: "services", Short: "Query Trade Map services data"}
	series := newTimeSeriesCommand(false)
	command.AddCommand(series)
	return command
}

func newTimeSeriesCommand(goods bool) *cobra.Command {
	use := "time-series"
	short := "Fetch one page, or every page, of a time series"
	command := &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			if goods {
				return runGoodsTimeSeries(command)
			}
			return runServiceTimeSeries(command)
		},
	}
	flags := command.Flags()
	flags.String("frequency", "", "yearly, quarterly, or monthly")
	flags.String("dimension", "", "byProduct, byCountry, or byPartner for goods; byService, byCountry, or byPartner for services")
	flags.String("reporter", "", "three-digit reporter economy code")
	flags.String("reporter-group", "", "reporter economy-group ID")
	flags.String("partner", "", "three-digit partner economy code")
	flags.String("partner-group", "", "partner economy-group ID")
	flags.String("from", "", "first period (YYYY, YYYYQQ, or YYYYMM)")
	flags.String("to", "", "last period (YYYY, YYYYQQ, or YYYYMM)")
	flags.String("flow", "", "trade flow: I, E, RE, or TB")
	flags.String("currency", "USD", "three-letter currency for value queries")
	flags.Int("page", 1, "page number")
	flags.Int("page-size", 100, "records per page")
	flags.Bool("all", false, "fetch every page within --max-pages and --max-records")
	flags.Int("max-pages", timeseries.DefaultLimits.MaxPages, "page limit when --all is set")
	flags.Int("max-records", timeseries.DefaultLimits.MaxRecords, "record limit when --all is set")
	flags.String("sort-by", "", "provider sort field")
	flags.String("sort-dir", "", "asc or desc")
	if goods {
		flags.String("product", "", "HS product code or ALL")
		flags.String("product-group", "", "product-group ID")
		flags.String("unit", "VAL", "measure: VAL, QTY, or GV")
		flags.String("data", "D", "direct, mirror, or mixed data: D, M, or X")
		flags.String("hs-level", "", "product depth for byProduct: 2, 4, 6, or 10")
	} else {
		flags.String("service", "", "EBOPS service code")
		flags.Bool("all-services", false, "request aggregate services (ALL)")
		flags.String("service-level", "", "EBOPS depth for byService: 3, 6, 9, 12, or 15")
	}
	required := []string{"frequency", "dimension", "from", "to", "flow"}
	for _, name := range required {
		_ = command.MarkFlagRequired(name)
	}
	return command
}

func runGoodsTimeSeries(command *cobra.Command) error {
	request, err := goodsRequest(command)
	if err != nil {
		return err
	}
	client, err := tradeMapClient(command)
	if err != nil {
		return err
	}
	return fetchPages(command, client.GoodsTimeSeries(), request)
}

func runServiceTimeSeries(command *cobra.Command) error {
	request, err := serviceRequest(command)
	if err != nil {
		return err
	}
	client, err := tradeMapClient(command)
	if err != nil {
		return err
	}
	return fetchPages(command, client.ServiceTimeSeries(), request)
}

func fetchPages(command *cobra.Command, service *timeseries.Service, request timeseries.Query) error {
	format, err := outputFormat(command)
	if err != nil {
		return err
	}
	all, err := command.Flags().GetBool("all")
	if err != nil {
		return err
	}
	ctx, cancel := commandContext(command)
	defer cancel()
	var pages []*timeseries.Page
	if all {
		maxPages, err := command.Flags().GetInt("max-pages")
		if err != nil {
			return err
		}
		maxRecords, err := command.Flags().GetInt("max-records")
		if err != nil {
			return err
		}
		pages, err = service.AllPages(ctx, request, timeseries.Limits{MaxPages: maxPages, MaxRecords: maxRecords})
		if err != nil {
			return err
		}
	} else {
		page, err := service.Page(ctx, request)
		if err != nil {
			return err
		}
		pages = []*timeseries.Page{page}
	}
	return writeTimeSeries(command.OutOrStdout(), format, pages)
}

func goodsRequest(command *cobra.Command) (timeseries.Request, error) {
	frequencyName, _ := command.Flags().GetString("frequency")
	frequency, err := parseFrequency(frequencyName, true)
	if err != nil {
		return timeseries.Request{}, err
	}
	dimensionName, _ := command.Flags().GetString("dimension")
	dimension, err := parseGoodsDimension(dimensionName)
	if err != nil {
		return timeseries.Request{}, err
	}
	reporter, err := reporterSelector(command)
	if err != nil {
		return timeseries.Request{}, err
	}
	partner, err := partnerSelector(command)
	if err != nil {
		return timeseries.Request{}, err
	}
	goods, err := goodsSelector(command)
	if err != nil {
		return timeseries.Request{}, err
	}
	from, _ := command.Flags().GetString("from")
	to, _ := command.Flags().GetString("to")
	periods, err := parsePeriods(frequency, from, to)
	if err != nil {
		return timeseries.Request{}, err
	}
	flowName, _ := command.Flags().GetString("flow")
	flow, err := parseFlow(flowName)
	if err != nil {
		return timeseries.Request{}, err
	}
	unitName, _ := command.Flags().GetString("unit")
	unit, err := parseUnit(unitName)
	if err != nil {
		return timeseries.Request{}, err
	}
	dataName, _ := command.Flags().GetString("data")
	dataMode, err := parseDataMode(dataName)
	if err != nil {
		return timeseries.Request{}, err
	}
	levelName, _ := command.Flags().GetString("hs-level")
	level, err := parseHSLevel(levelName)
	if err != nil {
		return timeseries.Request{}, err
	}
	if dimension.String() == "byProduct" && level.String() == "" {
		return timeseries.Request{}, fmt.Errorf("--hs-level is required for byProduct queries")
	}
	pagination, sort, currency, err := pageAndSort(command)
	if err != nil {
		return timeseries.Request{}, err
	}
	return timeseries.Request{
		Frequency: frequency, Dimension: dimension, Reporter: reporter, Partner: partner,
		Goods: goods, Periods: periods, Flow: flow, DataMode: dataMode, Unit: unit,
		Currency: currency, HSLevel: level, Pagination: pagination, Sort: sort,
	}, nil
}

func serviceRequest(command *cobra.Command) (timeseries.ServiceRequest, error) {
	frequencyName, _ := command.Flags().GetString("frequency")
	frequency, err := parseFrequency(frequencyName, false)
	if err != nil {
		return timeseries.ServiceRequest{}, err
	}
	dimensionName, _ := command.Flags().GetString("dimension")
	dimension, err := parseServiceDimension(dimensionName)
	if err != nil {
		return timeseries.ServiceRequest{}, err
	}
	reporter, err := reporterSelector(command)
	if err != nil {
		return timeseries.ServiceRequest{}, err
	}
	partner, err := partnerSelector(command)
	if err != nil {
		return timeseries.ServiceRequest{}, err
	}
	service, err := serviceSelector(command)
	if err != nil {
		return timeseries.ServiceRequest{}, err
	}
	from, _ := command.Flags().GetString("from")
	to, _ := command.Flags().GetString("to")
	periods, err := parsePeriods(frequency, from, to)
	if err != nil {
		return timeseries.ServiceRequest{}, err
	}
	flowName, _ := command.Flags().GetString("flow")
	flow, err := parseFlow(flowName)
	if err != nil {
		return timeseries.ServiceRequest{}, err
	}
	levelName, _ := command.Flags().GetString("service-level")
	level, err := parseServiceLevel(levelName)
	if err != nil {
		return timeseries.ServiceRequest{}, err
	}
	if dimension.String() == "byService" && level.String() == "" {
		return timeseries.ServiceRequest{}, fmt.Errorf("--service-level is required for byService queries")
	}
	pagination, sort, currency, err := pageAndSort(command)
	if err != nil {
		return timeseries.ServiceRequest{}, err
	}
	return timeseries.ServiceRequest{
		Frequency: frequency, Dimension: dimension, Reporter: reporter, Partner: partner,
		Service: service, Periods: periods, Flow: flow, Currency: currency,
		ServiceLevel: level, Pagination: pagination, Sort: sort,
	}, nil
}

func pageAndSort(command *cobra.Command) (timeseries.Pagination, timeseries.Sort, string, error) {
	page, err := command.Flags().GetInt("page")
	if err != nil {
		return timeseries.Pagination{}, timeseries.Sort{}, "", err
	}
	size, err := command.Flags().GetInt("page-size")
	if err != nil {
		return timeseries.Pagination{}, timeseries.Sort{}, "", err
	}
	sortBy, _ := command.Flags().GetString("sort-by")
	sortDir, _ := command.Flags().GetString("sort-dir")
	sort, err := parseSort(sortBy, sortDir)
	if err != nil {
		return timeseries.Pagination{}, timeseries.Sort{}, "", err
	}
	currency, err := command.Flags().GetString("currency")
	if err != nil {
		return timeseries.Pagination{}, timeseries.Sort{}, "", err
	}
	return timeseries.Pagination{Number: page, Size: size}, sort, currency, nil
}
