package main

const (
	API_URL              = "https://api.icicidirect.com/breezeapi/api/v1/"
	BREEZE_NEW_URL       = "https://breezeapi.icicidirect.com/api/v2/"
	LIVE_FEEDS_URL       = "https://livefeeds.icicidirect.com"
	LIVE_STREAM_URL      = "https://livestream.icicidirect.com"
	LIVE_OHLC_STREAM_URL = "https://breezeapi.icicidirect.com"
	SECURITY_MASTER_URL  = "https://directlink.icicidirect.com/NewSecurityMaster/SecurityMaster.zip"
	STOCK_SCRIPT_CSV_URL = "https://traderweb.icicidirect.com/Content/File/txtFile/ScripFile/StockScriptNew.csv"
)

var (
	TRANSACTION_TYPES      = []string{"debit", "credit"}
	INTERVAL_TYPES         = []string{"1minute", "5minute", "30minute", "1day"}
	INTERVAL_TYPES_HIST_V2 = []string{"1second", "1minute", "5minute", "30minute", "1day"}
	PRODUCT_TYPES          = []string{"futures", "options", "futureplus", "optionplus", "cash", "eatm", "margin", "mtf", "btst"}
	PRODUCT_TYPES_HIST     = []string{"futures", "options", "futureplus", "optionplus"}
	PRODUCT_TYPES_HIST_V2  = []string{"futures", "options", "cash"}
	RIGHT_TYPES            = []string{"call", "put", "others"}
	ACTION_TYPES           = []string{"buy", "sell"}
	ORDER_TYPES            = []string{"limit", "market", "stoploss"}
	VALIDITY_TYPES         = []string{"day", "ioc", "vtc"}
	EXCHANGE_CODES_HIST    = []string{"nse", "nfo", "ndx", "mcx"}
	EXCHANGE_CODES_HIST_V2 = []string{"nse", "bse", "nfo", "ndx", "mcx"}
	FNO_EXCHANGE_TYPES     = []string{"nfo", "mcx", "ndx"}
	STRATEGY_SUBSCRIPTION  = []string{"one_click_fno", "i_click_2_gain"}
	ISEC_NSE_CODE_MAP_FILE = map[string]string{"nse": "NSEScripMaster.txt", "bse": "BSEScripMaster.txt", "cdnse": "CDNSEScripMaster.txt", "fonse": "FONSEScripMaster.txt"}
	FeedIntervalMap        = map[string]string{"1MIN": "1minute", "5MIN": "5minute", "30MIN": "30minute", "1SEC": "1second"}
	ChannelIntervalMap     = map[string]string{"1minute": "1MIN", "5minute": "5MIN", "30minute": "30MIN", "1second": "1SEC"}
)

type APIRequestType string

const (
	POST   APIRequestType = "POST"
	GET    APIRequestType = "GET"
	PUT    APIRequestType = "PUT"
	DELETE APIRequestType = "DELETE"
)

type APIEndPoint string

const (
	CUST_DETAILS       APIEndPoint = "customerdetails"
	DEMAT_HOLDING      APIEndPoint = "dematholdings"
	FUND               APIEndPoint = "funds"
	HIST_CHART         APIEndPoint = "historicalcharts"
	MARGIN             APIEndPoint = "margin"
	ORDER              APIEndPoint = "order"
	PORTFOLIO_HOLDING  APIEndPoint = "portfolioholdings"
	PORTFOLIO_POSITION APIEndPoint = "portfoliopositions"
	QUOTE              APIEndPoint = "quotes"
	TRADE              APIEndPoint = "trades"
	OPT_CHAIN          APIEndPoint = "optionchain"
	SQUARE_OFF         APIEndPoint = "squareoff"
	LIMIT_CALCULATOR   APIEndPoint = "fnolmtpriceandqtycal"
	MARGIN_CALCULATOR  APIEndPoint = "margincalculator"
)
