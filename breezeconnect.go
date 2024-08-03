package main

import (
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type BreezeInstance struct {
	UserID                    string
	APIKey                    string
	SessionKey                string
	SecretKey                 string
	SIORateRefreshHandler     *SocketEventBreeze
	SIOOrderRefreshHandler    *SocketEventBreeze
	SIOOhlcvStreamHandler     *SocketEventBreeze
	APIHandler                *ApificationBreeze
	OnTicks                   func(map[string]interface{})
	OnTicks2                  func(map[string]interface{})
	StockScriptDictList       []map[string]string
	TokenScriptDictList       []map[string][]string
	TuxToUserValue            map[string]map[string]string
	OrderConnect              int
	Interval                  string
	LiveFeedsURL              string
	LiveStreamURL             string
	LiveOhlcStreamURL         string
	StockScriptCSVURL         string
	CustomerDetailsEndpoint   string
	ExceptMessage             map[string]string
	ConfigChannelIntervalMap  map[string]string
	ConfigIntervalTypesStream []string
	ResponseMessage           map[string]string
}

func NewBreezeInstance(apiKey string) *BreezeInstance {
	return &BreezeInstance{
		APIKey:              apiKey,
		StockScriptDictList: make([]map[string]string, 6),
		TokenScriptDictList: make([]map[string][]string, 6),
		OrderConnect:        0,
	}
}

func (b *BreezeInstance) socketConnectionResponse(message string) map[string]string {
	return map[string]string{"message": message}
}

func (b *BreezeInstance) subscribeException(message string) error {
	return errors.New(message)
}

func (b *BreezeInstance) _wsConnect(handler *SocketEventBreeze, orderFlag bool, ohlcvFlag bool, strategyFlag bool) error {
	var err error
	if orderFlag || strategyFlag {
		if b.SIOOrderRefreshHandler == nil {
			b.SIOOrderRefreshHandler = NewSocketEventBreeze("/", b)
		}
		if b.OrderConnect == 0 {
			err = b.SIOOrderRefreshHandler.Connect(b.LiveFeedsURL, false, false)
			b.OrderConnect++
		}
	} else if ohlcvFlag {
		if b.SIOOhlcvStreamHandler == nil {
			b.SIOOhlcvStreamHandler = NewSocketEventBreeze("/", b)
		}
		err = b.SIOOhlcvStreamHandler.Connect(b.LiveOhlcStreamURL, true, false)
	} else {
		if b.SIORateRefreshHandler == nil {
			b.SIORateRefreshHandler = NewSocketEventBreeze("/", b)
		}
		err = b.SIORateRefreshHandler.Connect(b.LiveStreamURL, false, false)
	}
	return err
}

func (b *BreezeInstance) WSDisconnect() []map[string]string {
	response := []map[string]string{}
	if b.SIORateRefreshHandler == nil {
		response = append(response, b.socketConnectionResponse(b.ResponseMessage["RATE_REFRESH_NOT_CONNECTED"]))
	} else {
		b.SIORateRefreshHandler.OnDisconnect()
		b.SIORateRefreshHandler = nil
		response = append(response, b.socketConnectionResponse(b.ResponseMessage["RATE_REFRESH_DISCONNECTED"]))
	}

	if b.SIOOhlcvStreamHandler == nil {
		response = append(response, b.socketConnectionResponse(b.ResponseMessage["OHLCV_STREAM_NOT_CONNECTED"]))
	} else {
		b.SIOOhlcvStreamHandler.OnDisconnect()
		b.SIOOhlcvStreamHandler = nil
		response = append(response, b.socketConnectionResponse(b.ResponseMessage["OHLCV_STREAM_DISCONNECTED"]))
	}

	if b.SIOOrderRefreshHandler == nil {
		response = append(response, b.socketConnectionResponse(b.ResponseMessage["ORDER_REFRESH_NOT_CONNECTED"]))
	} else {
		b.OrderConnect = 0
		b.SIOOrderRefreshHandler.OnDisconnect()
		b.SIOOrderRefreshHandler = nil
		response = append(response, b.socketConnectionResponse(b.ResponseMessage["ORDER_REFRESH_DISCONNECTED"]))
	}
	return response
}

func (b *BreezeInstance) WSDisconnectOhlc() map[string]string {
	if b.SIORateRefreshHandler != nil {
		b.SIORateRefreshHandler.OnDisconnect()
		b.SIORateRefreshHandler = nil
	}
	if b.SIOOhlcvStreamHandler == nil {
		return b.socketConnectionResponse(b.ResponseMessage["OHLCV_STREAM_NOT_CONNECTED"])
	}
	b.SIOOhlcvStreamHandler.OnDisconnect()
	b.SIOOhlcvStreamHandler = nil
	return b.socketConnectionResponse(b.ResponseMessage["OHLCV_STREAM_DISCONNECTED"])
}

func (b *BreezeInstance) WSConnect() error {
	return b._wsConnect(b.SIORateRefreshHandler, false, false, false)
}

func (b *BreezeInstance) GetDataFromStockTokenValue(inputStockToken string) (map[string]interface{}, error) {
	outputData := map[string]interface{}{}
	parts := strings.Split(inputStockToken, ".")
	if len(parts) < 2 {
		return nil, b.subscribeException(b.ExceptMessage["WRONG_EXCHANGE_CODE_EXCEPTION"])
	}
	exchangeType := parts[0]
	stockTokenParts := strings.Split(parts[1], "!")
	if len(stockTokenParts) < 2 {
		return nil, b.subscribeException(b.ExceptMessage["WRONG_EXCHANGE_CODE_EXCEPTION"])
	}
	stockToken := stockTokenParts[1]

	exchangeCodeList := map[string]string{
		"1":  "BSE",
		"4":  "NSE",
		"13": "NDX",
		"6":  "MCX",
	}
	exchangeCodeName, ok := exchangeCodeList[exchangeType]
	if !ok {
		return nil, b.subscribeException(b.ExceptMessage["WRONG_EXCHANGE_CODE_EXCEPTION"])
	}

	var stockData []string
	switch exchangeCodeName {
	case "BSE":
		stockData = b.TokenScriptDictList[0][stockToken]
	case "NSE":
		stockData = b.TokenScriptDictList[1][stockToken]
		if stockData == nil {
			stockData = b.TokenScriptDictList[4][stockToken]
			if stockData != nil {
				exchangeCodeName = "NFO"
			}
		}
	case "NDX":
		stockData = b.TokenScriptDictList[2][stockToken]
	case "MCX":
		stockData = b.TokenScriptDictList[3][stockToken]
	}

	if stockData == nil {
		return nil, b.subscribeException(fmt.Sprintf(b.ExceptMessage["STOCK_NOT_EXIST_EXCEPTION"], exchangeCodeName, inputStockToken))
	}

	outputData["stock_name"] = stockData[1]
	if exchangeCodeName != "NSE" && exchangeCodeName != "BSE" {
		productType := strings.Split(stockData[0], "-")[0]
		if productType == "FUT" {
			outputData["product_type"] = "Futures"
		}
		if productType == "OPT" {
			outputData["product_type"] = "Options"
		}
		dateString := strings.Join(stockData[0:5], "-")
		outputData["expiry_date"] = dateString
		if len(stockData) > 5 {
			outputData["strike_price"] = stockData[5]
			right := stockData[6]
			if right == "PE" {
				outputData["right"] = "Put"
			} else if right == "CE" {
				outputData["right"] = "Call"
			}
		}
	}

	return outputData, nil
}

func (b *BreezeInstance) getStockTokenValue(exchangeCode, stockCode, productType, expiryDate, strikePrice, right string, getExchangeQuotes, getMarketDepth bool) (string, string, error) {
	if !getExchangeQuotes && !getMarketDepth {
		return "", "", b.subscribeException(b.ExceptMessage["QUOTE_DEPTH_EXCEPTION"])
	}

	exchangeCodeList := map[string]string{
		"BSE": "1.",
		"NSE": "4.",
		"NDX": "13.",
		"MCX": "6.",
		"NFO": "4.",
		"BFO": "2.",
	}

	if b.Interval == "" {
		exchangeCodeList["BFO"] = "8."
	}

	exchangeCodeName, ok := exchangeCodeList[exchangeCode]
	if !ok {
		return "", "", b.subscribeException(b.ExceptMessage["EXCHANGE_CODE_EXCEPTION"])
	}

	if stockCode == "" {
		return "", "", b.subscribeException(b.ExceptMessage["STOCK_CODE_EXCEPTION"])
	}

	var tokenValue string
	switch exchangeCode {
	case "BSE":
		tokenValue = b.StockScriptDictList[0][stockCode]
	case "NSE":
		tokenValue = b.StockScriptDictList[1][stockCode]
	default:
		if expiryDate == "" {
			return "", "", b.subscribeException(b.ExceptMessage["EXPIRY_DATE_EXCEPTION"])
		}
		var contractDetailValue string
		if productType == "futures" {
			contractDetailValue = "FUT"
		} else if productType == "options" {
			contractDetailValue = "OPT"
		} else {
			return "", "", b.subscribeException(b.ExceptMessage["PRODUCT_TYPE_EXCEPTION"])
		}
		contractDetailValue += "-" + stockCode + "-" + expiryDate
		if productType == "options" {
			if strikePrice == "" {
				return "", "", b.subscribeException(b.ExceptMessage["STRIKE_PRICE_EXCEPTION"])
			}
			contractDetailValue += "-" + strikePrice
			if right == "put" {
				contractDetailValue += "-PE"
			} else if right == "call" {
				contractDetailValue += "-CE"
			} else {
				return "", "", b.subscribeException(b.ExceptMessage["RIGHT_EXCEPTION"])
			}
		}
		switch exchangeCode {
		case "NDX":
			tokenValue = b.StockScriptDictList[2][contractDetailValue]
		case "MCX":
			tokenValue = b.StockScriptDictList[3][contractDetailValue]
		case "NFO":
			tokenValue = b.StockScriptDictList[4][contractDetailValue]
		case "BFO":
			tokenValue = b.StockScriptDictList[5][contractDetailValue]
		}
	}

	if tokenValue == "" {
		return "", "", b.subscribeException(b.ExceptMessage["STOCK_INVALID_EXCEPTION"])
	}

	var exchangeQuotesTokenValue, marketDepthTokenValue string
	if getExchangeQuotes {
		exchangeQuotesTokenValue = exchangeCodeName + "1!" + tokenValue
	}
	if getMarketDepth {
		marketDepthTokenValue = exchangeCodeName + "2!" + tokenValue
	}
	return exchangeQuotesTokenValue, marketDepthTokenValue, nil
}

func (b *BreezeInstance) SubscribeFeeds(stockToken, exchangeCode, stockCode, productType, expiryDate, strikePrice, right, interval string, getExchangeQuotes, getMarketDepth, getOrderNotification bool) (map[string]string, error) {
	b.Interval = interval
	if b.SIORateRefreshHandler != nil && !b.SIORateRefreshHandler.Authentication {
		return nil, errors.New(b.ExceptMessage["AUTHENICATION_EXCEPTION"])
	}

	if interval != "" {
		if !contains(b.ConfigIntervalTypesStream, interval) {
			return nil, errors.New(b.ExceptMessage["STREAM_OHLC_INTERVAL_ERROR"])
		}
		interval = b.ConfigChannelIntervalMap[interval]
	}

	var returnObject map[string]string
	if b.SIORateRefreshHandler != nil {
		if b.SIOOrderRefreshHandler != nil && contains(config.STRATEGY_SUBSCRIPTION, stockToken) {
			err := b._wsConnect(b.SIOOrderRefreshHandler, false, false, true)
			if err != nil {
				return nil, err
			}
			b.SIOOrderRefreshHandler.Watch(stockToken)
			returnObject = b.socketConnectionResponse(fmt.Sprintf(b.ResponseMessage["STRATEGY_STREAM_SUBSCRIBED"], stockToken))
			return returnObject, nil
		}
		if getOrderNotification {
			err := b._wsConnect(b.SIOOrderRefreshHandler, true, false, false)
			if err != nil {
				return nil, err
			}
			b.SIOOrderRefreshHandler.Notify()
			returnObject = b.socketConnectionResponse(b.ResponseMessage["ORDER_NOTIFICATION_SUBSRIBED"])
			return returnObject, nil
		}
		if stockToken != "" {
			if interval != "" {
				if b.SIOOhlcvStreamHandler == nil {
					err := b._wsConnect(b.SIOOhlcvStreamHandler, false, true, false)
					if err != nil {
						return nil, err
					}
				}
				b.SIOOhlcvStreamHandler.WatchStreamData(stockToken, interval)
			} else {
				b.SIORateRefreshHandler.Watch(stockToken)
			}
			returnObject = b.socketConnectionResponse(fmt.Sprintf(b.ResponseMessage["STOCK_SUBSCRIBE_MESSAGE"], stockToken))
		} else if getOrderNotification && exchangeCode == "" {
			return returnObject, nil
		} else {
			exchangeQuotesToken, marketDepthToken, err := b.getStockTokenValue(exchangeCode, stockCode, productType, expiryDate, strikePrice, right, getExchangeQuotes, getMarketDepth)
			if err != nil {
				return nil, err
			}
			if interval != "" {
				if b.SIOOhlcvStreamHandler == nil {
					err := b._wsConnect(b.SIOOhlcvStreamHandler, false, true, false)
					if err != nil {
						return nil, err
					}
				}
				b.SIOOhlcvStreamHandler.WatchStreamData(exchangeQuotesToken, interval)
			} else {
				if exchangeQuotesToken != "" {
					b.SIORateRefreshHandler.Watch(exchangeQuotesToken)
				}
				if marketDepthToken != "" {
					b.SIORateRefreshHandler.Watch(marketDepthToken)
				}
			}
			returnObject = b.socketConnectionResponse(fmt.Sprintf(b.ResponseMessage["STOCK_SUBSCRIBE_MESSAGE"], stockCode))
		}
	}
	return returnObject, nil
}

func (b *BreezeInstance) UnsubscribeFeeds(stockToken, exchangeCode, stockCode, productType, expiryDate, strikePrice, right, interval string, getExchangeQuotes, getMarketDepth, getOrderNotification bool) (map[string]string, error) {
	if interval != "" {
		if !contains(b.ConfigIntervalTypesStream, interval) {
			return nil, errors.New(b.ExceptMessage["STREAM_OHLC_INTERVAL_ERROR"])
		}
		interval = b.ConfigChannelIntervalMap[interval]
	}

	if getOrderNotification {
		if b.SIOOrderRefreshHandler != nil {
			b.SIOOrderRefreshHandler.OnDisconnect()
			b.SIOOrderRefreshHandler = nil
			b.OrderConnect = 0
			return b.socketConnectionResponse(b.ResponseMessage["ORDER_REFRESH_DISCONNECTED"]), nil
		}
		return b.socketConnectionResponse(b.ResponseMessage["ORDER_REFRESH_NOT_CONNECTED"]), nil
	}

	if contains(config.STRATEGY_SUBSCRIPTION, stockToken) {
		if b.SIOOrderRefreshHandler != nil {
			b.SIOOrderRefreshHandler.Unwatch(stockToken)
			return b.socketConnectionResponse(fmt.Sprintf(b.ResponseMessage["STRATEGY_STREAM_UNSUBSCRIBED"], stockToken)), nil
		}
		return b.socketConnectionResponse(b.ResponseMessage["STRATEGY_STREAM_NOT_CONNECTED"]), nil
	}

	if b.SIORateRefreshHandler != nil {
		if stockToken != "" {
			if interval != "" {
				if b.SIOOhlcvStreamHandler != nil {
					b.SIOOhlcvStreamHandler.Unwatch(stockToken)
				}
			} else {
				b.SIORateRefreshHandler.Unwatch(stockToken)
			}
			return b.socketConnectionResponse(fmt.Sprintf(b.ResponseMessage["STOCK_UNSUBSCRIBE_MESSAGE"], stockToken)), nil
		} else {
			exchangeQuotesToken, marketDepthToken, err := b.getStockTokenValue(exchangeCode, stockCode, productType, expiryDate, strikePrice, right, getExchangeQuotes, getMarketDepth)
			if err != nil {
				return nil, err
			}
			if interval != "" {
				if b.SIOOhlcvStreamHandler != nil {
					b.SIOOhlcvStreamHandler.Unwatch(exchangeQuotesToken)
				}
			} else {
				if exchangeQuotesToken != "" {
					b.SIORateRefreshHandler.Unwatch(exchangeQuotesToken)
				}
				if marketDepthToken != "" {
					b.SIORateRefreshHandler.Unwatch(marketDepthToken)
				}
			}
			return b.socketConnectionResponse(fmt.Sprintf(b.ResponseMessage["STOCK_UNSUBSCRIBE_MESSAGE"], stockCode)), nil
		}
	}
	return nil, nil
}

func (b *BreezeInstance) parseOhlcData(data string) map[string]interface{} {
	splitData := strings.Split(data, ",")
	parsedData := make(map[string]interface{})
	if len(splitData) == 9 {
		parsedData = map[string]interface{}{
			"interval":      config.feedIntervalMap[splitData[8]],
			"exchange_code": splitData[0],
			"stock_code":    splitData[1],
			"low":           splitData[2],
			"high":          splitData[3],
			"open":          splitData[4],
			"close":         splitData[5],
			"volume":        splitData[6],
			"datetime":      splitData[7],
		}
	} else if len(splitData) == 13 {
		parsedData = map[string]interface{}{
			"interval":      config.feedIntervalMap[splitData[12]],
			"exchange_code": splitData[0],
			"stock_code":    splitData[1],
			"expiry_date":   splitData[2],
			"strike_price":  splitData[3],
			"right_type":    splitData[4],
			"low":           splitData[5],
			"high":          splitData[6],
			"open":          splitData[7],
			"close":         splitData[8],
			"volume":        splitData[9],
			"oi":            splitData[10],
			"datetime":      splitData[11],
		}
	} else if len(splitData) == 11 {
		parsedData = map[string]interface{}{
			"interval":      config.feedIntervalMap[splitData[10]],
			"exchange_code": splitData[0],
			"stock_code":    splitData[1],
			"expiry_date":   splitData[2],
			"low":           splitData[3],
			"high":          splitData[4],
			"open":          splitData[5],
			"close":         splitData[6],
			"volume":        splitData[7],
			"oi":            splitData[8],
			"datetime":      splitData[9],
		}
	}
	return parsedData
}

func (b *BreezeInstance) parseMarketDepth(data [][]string, exchange string) []map[string]interface{} {
	depth := []map[string]interface{}{}
	for i, lis := range data {
		dict := make(map[string]interface{})
		if exchange == "1" {
			dict[fmt.Sprintf("BestBuyRate-%d", i+1)] = lis[0]
			dict[fmt.Sprintf("BestBuyQty-%d", i+1)] = lis[1]
			dict[fmt.Sprintf("BestSellRate-%d", i+1)] = lis[2]
			dict[fmt.Sprintf("BestSellQty-%d", i+1)] = lis[3]
		} else if exchange == "8" {
			dict[fmt.Sprintf("BestBuyRate-%d", i+1)] = lis[0]
			dict[fmt.Sprintf("BestBuyQty-%d", i+1)] = lis[1]
			dict[fmt.Sprintf("BuyNoOfOrders-%d", i+1)] = lis[2]
			dict[fmt.Sprintf("BestSellRate-%d", i+1)] = lis[3]
			dict[fmt.Sprintf("BestSellQty-%d", i+1)] = lis[4]
			dict[fmt.Sprintf("SellNoOfOrders-%d", i+1)] = lis[5]
		} else {
			dict[fmt.Sprintf("BestBuyRate-%d", i+1)] = lis[0]
			dict[fmt.Sprintf("BestBuyQty-%d", i+1)] = lis[1]
			dict[fmt.Sprintf("BuyNoOfOrders-%d", i+1)] = lis[2]
			dict[fmt.Sprintf("BuyFlag-%d", i+1)] = lis[3]
			dict[fmt.Sprintf("BestSellRate-%d", i+1)] = lis[4]
			dict[fmt.Sprintf("BestSellQty-%d", i+1)] = lis[5]
			dict[fmt.Sprintf("SellNoOfOrders-%d", i+1)] = lis[6]
			dict[fmt.Sprintf("SellFlag-%d", i+1)] = lis[7]
		}
		depth = append(depth, dict)
	}
	return depth
}

func (b *BreezeInstance) parseData(data []interface{}) map[string]interface{} {
	if len(data) == 0 {
		return nil
	}

	if s, ok := data[0].(string); ok && !strings.Contains(s, "!") {
		if len(data) == 19 {
			iclickData := map[string]interface{}{
				"stock_name":                 data[0],
				"stock_code":                 data[1],
				"action_type":                data[2],
				"expiry_date":                data[3],
				"strike_price":               data[4],
				"option_type":                data[5],
				"stock_description":          data[6],
				"recommended_price_and_date": data[7],
				"recommended_price_from":     data[8],
				"recommended_price_to":       data[9],
				"recommended_date":           data[10],
				"target_price":               data[11],
				"sltp_price":                 data[12],
				"part_profit_percentage":     data[13],
				"profit_price":               data[14],
				"exit_price":                 data[15],
				"recommended_update":         data[16],
				"iclick_status":              data[17],
				"subscription_type":          data[18],
			}
			return iclickData
		} else if len(data) == 28 {
			strategyDict := map[string]interface{}{
				"strategy_date":           data[0],
				"modification_date":       data[1],
				"portfolio_id":            data[2],
				"call_action":             data[3],
				"portfolio_name":          data[4],
				"exchange_code":           data[5],
				"product_type":            data[6],
				"underlying":              data[8],
				"expiry_date":             data[9],
				"option_type":             data[11],
				"strike_price":            data[12],
				"action":                  data[13],
				"recommended_price_from":  data[14],
				"recommended_price_to":    data[15],
				"minimum_lot_quantity":    data[16],
				"last_traded_price":       data[17],
				"best_bid_price":          data[18],
				"best_offer_price":        data[19],
				"last_traded_quantity":    data[20],
				"target_price":            data[21],
				"expected_profit_per_lot": data[22],
				"stop_loss_price":         data[23],
				"expected_loss_per_lot":   data[24],
				"total_margin":            data[25],
				"leg_no":                  data[26],
				"status":                  data[27],
			}
			return strategyDict
		} else if len(data) == 42 {
			orderDict := map[string]interface{}{
				"sourceNumber":              data[0],
				"group":                     data[1],
				"userId":                    data[2],
				"key":                       data[3],
				"messageLength":             data[4],
				"requestType":               data[5],
				"messageSequence":           data[6],
				"messageDate":               data[7],
				"messageTime":               data[8],
				"messageCategory":           data[9],
				"messagePriority":           data[10],
				"messageType":               data[11],
				"orderMatchAccount":         data[12],
				"orderExchangeCode":         data[13],
				"stockCode":                 data[14],
				"orderFlow":                 b.TuxToUserValue["orderFlow"][data[15].(string)],
				"limitMarketFlag":           b.TuxToUserValue["limitMarketFlag"][data[16].(string)],
				"orderType":                 b.TuxToUserValue["orderType"][data[17].(string)],
				"orderLimitRate":            data[18],
				"productType":               b.TuxToUserValue["productType"][data[19].(string)],
				"orderStatus":               b.TuxToUserValue["orderStatus"][data[20].(string)],
				"orderDate":                 data[21],
				"orderTradeDate":            data[22],
				"orderReference":            data[23],
				"orderQuantity":             data[24],
				"openQuantity":              data[25],
				"orderExecutedQuantity":     data[26],
				"cancelledQuantity":         data[27],
				"expiredQuantity":           data[28],
				"orderDisclosedQuantity":    data[29],
				"orderStopLossTrigger":      data[30],
				"orderSquareFlag":           data[31],
				"orderAmountBlocked":        data[32],
				"orderPipeId":               data[33],
				"channel":                   data[34],
				"exchangeSegmentCode":       data[35],
				"exchangeSegmentSettlement": data[36],
				"segmentDescription":        data[37],
				"marginSquareOffMode":       data[38],
				"orderValidDate":            data[40],
				"orderMessageCharacter":     data[41],
				"averageExecutedRate":       data[42],
				"orderPriceImprovementFlag": data[43],
				"orderMBCFlag":              data[44],
				"orderLimitOffset":          data[45],
				"systemPartnerCode":         data[46],
			}
			return orderDict
		} else if len(data) == 43 {
			orderDict := map[string]interface{}{
				"sourceNumber":              data[0],
				"group":                     data[1],
				"userId":                    data[2],
				"key":                       data[3],
				"messageLength":             data[4],
				"requestType":               data[5],
				"messageSequence":           data[6],
				"messageDate":               data[7],
				"messageTime":               data[8],
				"messageCategory":           data[9],
				"messagePriority":           data[10],
				"messageType":               data[11],
				"orderMatchAccount":         data[12],
				"orderExchangeCode":         data[13],
				"stockCode":                 data[14],
				"orderFlow":                 b.TuxToUserValue["orderFlow"][data[21].(string)],
				"limitMarketFlag":           b.TuxToUserValue["limitMarketFlag"][data[22].(string)],
				"orderType":                 b.TuxToUserValue["orderType"][data[23].(string)],
				"orderLimitRate":            data[24],
				"productType":               b.TuxToUserValue["productType"][data[15].(string)],
				"orderStatus":               b.TuxToUserValue["orderStatus"][data[25].(string)],
				"orderReference":            data[26],
				"orderTotalQuantity":        data[27],
				"executedQuantity":          data[28],
				"cancelledQuantity":         data[29],
				"expiredQuantity":           data[30],
				"stopLossTrigger":           data[31],
				"specialFlag":               data[32],
				"pipeId":                    data[33],
				"channel":                   data[34],
				"modificationOrCancelFlag":  data[35],
				"tradeDate":                 data[36],
				"acknowledgeNumber":         data[37],
				"stopLossOrderReference":    data[37],
				"totalAmountBlocked":        data[38],
				"averageExecutedRate":       data[39],
				"cancelFlag":                data[40],
				"squareOffMarket":           data[41],
				"quickExitFlag":             data[42],
				"stopValidTillDateFlag":     data[43],
				"priceImprovementFlag":      data[44],
				"conversionImprovementFlag": data[45],
				"trailUpdateCondition":      data[45],
				"systemPartnerCode":         data[46],
			}
			return orderDict
		}
	}

	exchange := strings.Split(data[0].(string), "!")[0]
	dataType := strings.Split(data[0].(string), "!")[1]
	dataDict := make(map[string]interface{})
	if exchange == "6" {
		dataDict["symbol"] = data[0]
		dataDict["AndiOPVolume"] = data[1]
		dataDict["Reserved"] = data[2]
		dataDict["IndexFlag"] = data[3]
		dataDict["ttq"] = data[4]
		dataDict["last"] = data[5]
		dataDict["ltq"] = data[6]
		dataDict["ltt"] = time.Unix(int64(data[7].(float64)), 0).Format(time.RFC3339)
		dataDict["AvgTradedPrice"] = data[8]
		dataDict["TotalBuyQnt"] = data[9]
		dataDict["TotalSellQnt"] = data[10]
		dataDict["ReservedStr"] = data[11]
		dataDict["ClosePrice"] = data[12]
		dataDict["OpenPrice"] = data[13]
		dataDict["HighPrice"] = data[14]
		dataDict["LowPrice"] = data[15]
		dataDict["ReservedShort"] = data[16]
		dataDict["CurrOpenInterest"] = data[17]
		dataDict["TotalTrades"] = data[18]
		dataDict["HightestPriceEver"] = data[19]
		dataDict["LowestPriceEver"] = data[20]
		dataDict["TotalTradedValue"] = data[21]
		for i := 22; i < len(data); i++ {
			dataDict[fmt.Sprintf("Quantity-%d", i-22)] = data[i].([]interface{})[0]
			dataDict[fmt.Sprintf("OrderPrice-%d", i-22)] = data[i].([]interface{})[1]
			dataDict[fmt.Sprintf("TotalOrders-%d", i-22)] = data[i].([]interface{})[2]
			dataDict[fmt.Sprintf("Reserved-%d", i-22)] = data[i].([]interface{})[3]
			dataDict[fmt.Sprintf("SellQuantity-%d", i-22)] = data[i].([]interface{})[4]
			dataDict[fmt.Sprintf("SellOrderPrice-%d", i-22)] = data[i].([]interface{})[5]
			dataDict[fmt.Sprintf("SellTotalOrders-%d", i-22)] = data[i].([]interface{})[6]
			dataDict[fmt.Sprintf("SellReserved-%d", i-22)] = data[i].([]interface{})[7]
		}
	} else if dataType == "1" {
		dataDict["symbol"] = data[0]
		dataDict["open"] = data[1]
		dataDict["last"] = data[2]
		dataDict["high"] = data[3]
		dataDict["low"] = data[4]
		dataDict["change"] = data[5]
		dataDict["bPrice"] = data[6]
		dataDict["bQty"] = data[7]
		dataDict["sPrice"] = data[8]
		dataDict["sQty"] = data[9]
		dataDict["ltq"] = data[10]
		dataDict["avgPrice"] = data[11]
		dataDict["quotes"] = "Quotes Data"
		if len(data) == 21 {
			dataDict["ttq"] = data[12]
			dataDict["totalBuyQt"] = data[13]
			dataDict["totalSellQ"] = data[14]
			dataDict["ttv"] = data[15]
			dataDict["trend"] = data[16]
			dataDict["lowerCktLm"] = data[17]
			dataDict["upperCktLm"] = data[18]
			dataDict["ltt"] = time.Unix(int64(data[19].(float64)), 0).Format(time.RFC3339)
			dataDict["close"] = data[20]
		} else if len(data) == 23 {
			dataDict["OI"] = data[12]
			dataDict["CHNGOI"] = data[13]
			dataDict["ttq"] = data[14]
			dataDict["totalBuyQt"] = data[15]
			dataDict["totalSellQ"] = data[16]
			dataDict["ttv"] = data[17]
			dataDict["trend"] = data[18]
			dataDict["lowerCktLm"] = data[19]
			dataDict["upperCktLm"] = data[20]
			dataDict["ltt"] = time.Unix(int64(data[21].(float64)), 0).Format(time.RFC3339)
			dataDict["close"] = data[22]
		}
	} else {
		dataDict["symbol"] = data[0]
		dataDict["time"] = time.Unix(int64(data[1].(float64)), 0).Format(time.RFC3339)
		dataDict["depth"] = b.parseMarketDepth(data[2].([][]string), exchange)
		dataDict["quotes"] = "Market Depth"
	}
	switch exchange {
	case "4":
		if len(data) == 21 {
			dataDict["exchange"] = "NSE Equity"
		} else if len(data) == 23 {
			dataDict["exchange"] = "NSE Futures & Options"
		}
	case "1":
		dataDict["exchange"] = "BSE"
	case "13":
		dataDict["exchange"] = "NSE Currency"
	case "6":
		dataDict["exchange"] = "Commodity"
	}
	return dataDict
}

func (b *BreezeInstance) apiUtil() error {
	headers := map[string]string{
		"Content-Type": "application/json",
	}
	body := map[string]string{
		"SessionToken": b.SessionKey,
		"AppKey":       b.APIKey,
	}
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return err
	}

	url := b.CustomerDetailsEndpoint
	req, err := http.NewRequest("GET", url, strings.NewReader(string(bodyJSON)))
	if err != nil {
		return err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var jsonData map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&jsonData)
	if err != nil {
		return err
	}

	if success, ok := jsonData["Success"].(map[string]interface{}); ok {
		if base64SessionToken, ok := success["session_token"].(string); ok {
			result, err := base64.StdEncoding.DecodeString(base64SessionToken)
			if err != nil {
				return err
			}
			resultStr := string(result)
			parts := strings.Split(resultStr, ":")
			if len(parts) < 2 {
				return errors.New("invalid session token format")
			}
			b.UserID = parts[0]
			b.SessionKey = parts[1]
			return nil
		}
	}

	if status, ok := jsonData["Status"].(float64); ok && int(status) != 200 {
		if errMsg, ok := jsonData["Error"].(string); ok {
			switch errMsg {
			case "Invalid session.":
				return errors.New(b.ExceptMessage["SESSIONKEY_INCORRECT"])
			case "Public Key does not exist.":
				return errors.New(b.ExceptMessage["APPKEY_INCORRECT"])
			case "Resource not available.":
				return errors.New(b.ExceptMessage["SESSIONKEY_EXPIRED"])
			default:
				return errors.New(b.ExceptMessage["CUSTOMERDETAILS_API_EXCEPTION"])
			}
		}
	}

	return errors.New("unexpected format in API response")
}

func (b *BreezeInstance) getStockScriptList() error {
	b.StockScriptDictList = make([]map[string]string, 6)
	b.TokenScriptDictList = make([]map[string][]string, 6)
	resp, err := http.Get(b.StockScriptCSVURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	reader := csv.NewReader(resp.Body)
	records, err := reader.ReadAll()
	if err != nil {
		return err
	}

	for _, row := range records {
		switch row[2] {
		case "BSE":
			b.StockScriptDictList[0][row[3]] = row[5]
			b.TokenScriptDictList[0][row[5]] = []string{row[3], row[1]}
		case "NSE":
			b.StockScriptDictList[1][row[3]] = row[5]
			b.TokenScriptDictList[1][row[5]] = []string{row[3], row[1]}
		case "NDX":
			b.StockScriptDictList[2][row[7]] = row[5]
			b.TokenScriptDictList[2][row[5]] = []string{row[7], row[1]}
		case "MCX":
			b.StockScriptDictList[3][row[7]] = row[5]
			b.TokenScriptDictList[3][row[5]] = []string{row[7], row[1]}
		case "NFO":
			b.StockScriptDictList[4][row[7]] = row[5]
			b.TokenScriptDictList[4][row[5]] = []string{row[7], row[1]}
		case "BFO":
			b.StockScriptDictList[5][row[7]] = row[5]
			b.TokenScriptDictList[5][row[5]] = []string{row[7], row[1]}
		}
	}

	return nil
}

func (b *BreezeInstance) GenerateSession(apiSecret, sessionToken string) error {
	b.SessionKey = sessionToken
	b.SecretKey = apiSecret
	err := b.apiUtil()
	if err != nil {
		return err
	}
	err = b.getStockScriptList()
	if err != nil {
		return err
	}
	b.APIHandler = NewApificationBreeze(b)
	return nil
}

// Add other methods for BreezeInstance here as needed.

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}
