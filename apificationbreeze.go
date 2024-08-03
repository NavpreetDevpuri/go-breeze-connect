package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

type ApificationBreeze struct {
	Breeze             *BreezeConnect
	Hostname           string
	Base64SessionToken string
}

func NewApificationBreeze(breezeInstance *BreezeConnect) *ApificationBreeze {
	base64SessionToken := base64.StdEncoding.EncodeToString([]byte(breezeInstance.UserID + ":" + breezeInstance.SessionKey))
	return &ApificationBreeze{
		Breeze:             breezeInstance,
		Hostname:           breezeInstance.APIURL,
		Base64SessionToken: base64SessionToken,
	}
}

func (a *ApificationBreeze) ErrorException(funcName string, err error) {
	message := fmt.Sprintf("%s() Error: %s", funcName, err)
	panic(message)
}

func (a *ApificationBreeze) ValidationErrorResponse(message string) map[string]interface{} {
	return map[string]interface{}{
		"Success": "",
		"Status":  500,
		"Error":   message,
	}
}

func (a *ApificationBreeze) GenerateHeaders(body string) (map[string]string, error) {
	currentDate := time.Now().UTC().Format(time.RFC3339)[:19] + ".000Z"
	checksum := sha256.Sum256([]byte(currentDate + body + a.Breeze.SecretKey))
	headers := map[string]string{
		"Content-Type":   "application/json",
		"X-Checksum":     "token " + fmt.Sprintf("%x", checksum),
		"X-Timestamp":    currentDate,
		"X-AppKey":       a.Breeze.APIKey,
		"X-SessionToken": a.Base64SessionToken,
		"User-Agent":     "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_5_8) AppleWebKit/534.50.2 (KHTML, like Gecko) Version/5.0.6 Safari/533.22.3",
	}
	return headers, nil
}

func (a *ApificationBreeze) MakeRequest(method, endpoint, body string, headers map[string]string) (*http.Response, error) {
	url := a.Hostname + endpoint
	req, err := http.NewRequest(method, url, bytes.NewBuffer([]byte(body)))
	if err != nil {
		return nil, err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (a *ApificationBreeze) GetCustomerDetails(apiSession string) (map[string]interface{}, error) {
	if apiSession == "" {
		return a.ValidationErrorResponse("API session is missing"), nil
	}

	headers := map[string]string{
		"Content-Type": "application/json",
	}
	body := map[string]string{
		"SessionToken": apiSession,
		"AppKey":       a.Breeze.APIKey,
	}
	bodyJSON, _ := json.Marshal(body)
	response, err := a.MakeRequest("GET", "/cust_details", string(bodyJSON), headers)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	var result map[string]interface{}
	err = json.NewDecoder(response.Body).Decode(&result)
	if err != nil {
		return nil, err
	}
	if success, ok := result["Success"].(map[string]interface{}); ok {
		delete(success, "session_token")
	}
	return result, nil
}

func (a *ApificationBreeze) GetDematHoldings() (map[string]interface{}, error) {
	body := "{}"
	headers, err := a.GenerateHeaders(body)
	if err != nil {
		return nil, err
	}

	response, err := a.MakeRequest("GET", "/demat_holdings", body, headers)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	var result map[string]interface{}
	err = json.NewDecoder(response.Body).Decode(&result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (a *ApificationBreeze) GetFunds() (map[string]interface{}, error) {
	body := "{}"
	headers, err := a.GenerateHeaders(body)
	if err != nil {
		return nil, err
	}

	response, err := a.MakeRequest("GET", "/funds", body, headers)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	var result map[string]interface{}
	err = json.NewDecoder(response.Body).Decode(&result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (a *ApificationBreeze) SetFunds(transactionType, amount, segment string) (map[string]interface{}, error) {
	if transactionType == "" || amount == "" || segment == "" {
		return a.ValidationErrorResponse("Transaction type, amount or segment cannot be empty"), nil
	}

	body := map[string]string{
		"transaction_type": transactionType,
		"amount":           amount,
		"segment":          segment,
	}
	bodyJSON, _ := json.Marshal(body)
	headers, err := a.GenerateHeaders(string(bodyJSON))
	if err != nil {
		return nil, err
	}

	response, err := a.MakeRequest("POST", "/funds", string(bodyJSON), headers)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	var result map[string]interface{}
	err = json.NewDecoder(response.Body).Decode(&result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (a *ApificationBreeze) GetHistoricalData(interval, fromDate, toDate, stockCode, exchangeCode, productType, expiryDate, right, strikePrice string) (map[string]interface{}, error) {
	if interval == "" || fromDate == "" || toDate == "" || stockCode == "" || exchangeCode == "" {
		return a.ValidationErrorResponse("Required parameters are missing"), nil
	}

	body := map[string]string{
		"interval":      interval,
		"from_date":     fromDate,
		"to_date":       toDate,
		"stock_code":    stockCode,
		"exchange_code": exchangeCode,
		"product_type":  productType,
		"expiry_date":   expiryDate,
		"right":         right,
		"strike_price":  strikePrice,
	}
	bodyJSON, _ := json.Marshal(body)
	headers, err := a.GenerateHeaders(string(bodyJSON))
	if err != nil {
		return nil, err
	}

	response, err := a.MakeRequest("GET", "/hist_chart", string(bodyJSON), headers)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	var result map[string]interface{}
	err = json.NewDecoder(response.Body).Decode(&result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (a *ApificationBreeze) GetOrderDetail(exchangeCode, orderID string) (map[string]interface{}, error) {
	if exchangeCode == "" || orderID == "" {
		return a.ValidationErrorResponse("Exchange code or order ID cannot be empty"), nil
	}

	body := map[string]string{
		"exchange_code": exchangeCode,
		"order_id":      orderID,
	}
	bodyJSON, _ := json.Marshal(body)
	headers, err := a.GenerateHeaders(string(bodyJSON))
	if err != nil {
		return nil, err
	}

	response, err := a.MakeRequest("GET", "/order", string(bodyJSON), headers)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	var result map[string]interface{}
	err = json.NewDecoder(response.Body).Decode(&result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Add other methods as needed...

// Helper functions
func (a *ApificationBreeze) convertToJSON(v interface{}) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (a *ApificationBreeze) parseResponseBody(body *http.Response) (map[string]interface{}, error) {
	data, err := ioutil.ReadAll(body.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}
