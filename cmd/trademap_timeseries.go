package cmd

import (
	"fmt"

	"github.com/amar-jay/amartrade/internal/trademap/timeseries"
	"github.com/spf13/cobra"
)

func newGoodsCommand() *cobra.Command {
	command := &cobra.Command{Use: "goods", Short: "Query Trade Map goods data"}
	command.AddCommand(newTimeSeriesCommand(true))
	return command
}

func newServicesCommand() *cobra.Command {
	command := &cobra.Command{Use: "services", Short: "Query Trade Map services data"}
	command.AddCommand(newTimeSeriesCommand(false))
	return command
}

func newTimeSeriesCommand(goods bool) *cobra.Command {
	command := &cobra.Command{
		Use:   "time-series",
		Short: "Fetch one page, or every page, of a time series",
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
	flags.String("reporter", "", "reporter economy: name (Germany), 3-digit code (276), or group name/ID")
	flags.String("partner", "world", "partner economy: name, code, or group (default world)")
	flags.String("from", "", "first period (2024, 2024-Q1, 2024-01, Jan-2024)")
	flags.String("to", "", "last period (default: --from)")
	flags.String("flow", "", "trade flow: I, E, RE, or TB (imports, exports, ...)")
	flags.String("currency", "USD", "three-letter currency for value queries")
	flags.Int("page", 1, "page number")
	flags.Int("page-size", 100, "records per page")
	flags.Bool("all", false, "fetch every page within --max-pages and --max-records")
	flags.Int("max-pages", timeseries.DefaultLimits.MaxPages, "page limit when --all is set")
	flags.Int("max-records", timeseries.DefaultLimits.MaxRecords, "record limit when --all is set")
	flags.String("sort-by", "", "provider sort field")
	flags.String("sort-dir", "", "asc or desc")
	if goods {
		flags.String("product", "all", "HS product: name, code, group name/ID, or all")
		flags.String("unit", "VAL", "measure: VAL, QTY, or GV")
		flags.String("data", "D", "direct, mirror, or mixed data: D, M, or X")
		flags.String("hs-level", "", "product depth for byProduct: 2, 4, 6, or 10")
	} else {
		flags.String("service", "", "EBOPS service: name, code (S...), display code (3.1.1), or all")
		flags.Bool("all-services", false, "request aggregate services (ALL)")
		flags.String("service-level", "", "EBOPS depth for byService: 3, 6, 9, 12, or 15")
	}
	_ = command.MarkFlagRequired("frequency")
	_ = command.MarkFlagRequired("dimension")
	_ = command.MarkFlagRequired("from")
	_ = command.MarkFlagRequired("flow")
	return command
}

func runGoodsTimeSeries(command *cobra.Command) error {
	client, err := tradeMapClient(command)
	if err != nil {
		return err
	}
	ctx, cancel := commandContext(command)
	defer cancel()
	request, err := goodsRequest(command, newResolver(ctx, client.ReferenceData()))
	if err != nil {
		return err
	}
	return fetchPages(command, client.GoodsTimeSeries(), request)
}

func runServiceTimeSeries(command *cobra.Command) error {
	client, err := tradeMapClient(command)
	if err != nil {
		return err
	}
	ctx, cancel := commandContext(command)
	defer cancel()
	request, err := serviceRequest(command, newResolver(ctx, client.ReferenceData()))
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

func goodsRequest(command *cobra.Command, resolve *resolver) (timeseries.Request, error) {
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
	reporterName, _ := command.Flags().GetString("reporter")
	reporter, reporterNote, err := resolve.resolveReporter(reporterName)
	if err != nil {
		return timeseries.Request{}, fmt.Errorf("--reporter: %w", err)
	}
	partnerName, _ := command.Flags().GetString("partner")
	partner, partnerNote, err := resolve.resolvePartner(partnerName)
	if err != nil {
		return timeseries.Request{}, fmt.Errorf("--partner: %w", err)
	}
	productName, _ := command.Flags().GetString("product")
	goods, goodsNote, err := resolve.resolveGoods(productName)
	if err != nil {
		return timeseries.Request{}, fmt.Errorf("--product: %w", err)
	}
	from, _ := command.Flags().GetString("from")
	to, _ := command.Flags().GetString("to")
	periods, err := parseFriendlyPeriodRange(frequency, from, to)
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
		return timeseries.Request{}, missingFlagError("--hs-level is required for byProduct queries")
	}
	pagination, sort, currency, err := pageAndSort(command)
	if err != nil {
		return timeseries.Request{}, err
	}
	explain(command, "reporter %s; partner %s; %s", reporterNote, partnerNote, goodsNote)
	return timeseries.Request{
		Frequency: frequency, Dimension: dimension, Reporter: reporter, Partner: partner,
		Goods: goods, Periods: periods, Flow: flow, DataMode: dataMode, Unit: unit,
		Currency: currency, HSLevel: level, Pagination: pagination, Sort: sort,
	}, nil
}

func serviceRequest(command *cobra.Command, resolve *resolver) (timeseries.ServiceRequest, error) {
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
	reporterName, _ := command.Flags().GetString("reporter")
	reporter, reporterNote, err := resolve.resolveReporter(reporterName)
	if err != nil {
		return timeseries.ServiceRequest{}, fmt.Errorf("--reporter: %w", err)
	}
	partnerName, _ := command.Flags().GetString("partner")
	partner, partnerNote, err := resolve.resolvePartner(partnerName)
	if err != nil {
		return timeseries.ServiceRequest{}, fmt.Errorf("--partner: %w", err)
	}
	serviceName, _ := command.Flags().GetString("service")
	allServices, _ := command.Flags().GetBool("all-services")
	service, serviceNote, err := resolve.resolveService(serviceName, allServices)
	if err != nil {
		return timeseries.ServiceRequest{}, fmt.Errorf("--service: %w", err)
	}
	from, _ := command.Flags().GetString("from")
	to, _ := command.Flags().GetString("to")
	periods, err := parseFriendlyPeriodRange(frequency, from, to)
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
		return timeseries.ServiceRequest{}, missingFlagError("--service-level is required for byService queries")
	}
	pagination, sort, currency, err := pageAndSort(command)
	if err != nil {
		return timeseries.ServiceRequest{}, err
	}
	explain(command, "reporter %s; partner %s; %s", reporterNote, partnerNote, serviceNote)
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
