package pay

import (
    "bytes"
    "context"
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

// Client is a minimal AirbaPay API client.
type Client struct {
    httpClient *http.Client
    merchantID string
    secret     string
    callback   string
    baseURL    string
}

// NewClient constructs a new AirbaPay client.
func NewClient(httpClient *http.Client, merchantID, secret, callback string) *Client {
    if httpClient == nil {
        httpClient = &http.Client{Timeout: 10 * time.Second}
    }
    return &Client{
        httpClient: httpClient,
        merchantID: merchantID,
        secret:     secret,
        callback:   callback,
        baseURL:    "https://airbapay.kz/api",
    }
}

// Secret returns the configured API secret.
func (c *Client) Secret() string { return c.secret }

// CreatePaymentRequest describes parameters for invoice creation.
type CreatePaymentRequest struct {
    OrderID     int64  `json:"order_id"`
    Amount      int    `json:"amount"`
    Currency    string `json:"currency"`
    Description string `json:"description"`
}

// CreatePaymentResponse contains payment provider data.
type CreatePaymentResponse struct {
    PaymentURL string `json:"payment_url"`
    InvoiceID  string `json:"invoice_id"`
}

// CreatePayment creates an invoice via AirbaPay API.
func (c *Client) CreatePayment(ctx context.Context, req CreatePaymentRequest) (CreatePaymentResponse, error) {
    payload := map[string]interface{}{
        "merchant_id":  c.merchantID,
        "order_id":     req.OrderID,
        "amount":       req.Amount,
        "currency":     req.Currency,
        "description":  req.Description,
        "callback_url": c.callback,
    }

    body, err := json.Marshal(payload)
    if err != nil {
        return CreatePaymentResponse{}, err
    }

    httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/payment", bytes.NewReader(body))
    if err != nil {
        return CreatePaymentResponse{}, err
    }
    httpReq.Header.Set("Content-Type", "application/json")
    httpReq.Header.Set("X-Signature", c.sign(body))

    resp, err := c.httpClient.Do(httpReq)
    if err != nil {
        return CreatePaymentResponse{}, err
    }
    defer resp.Body.Close()

    if resp.StatusCode >= 300 {
        return CreatePaymentResponse{}, fmt.Errorf("airbapay: unexpected status %s", resp.Status)
    }

    var apiResp struct {
        Success bool   `json:"success"`
        URL     string `json:"payment_url"`
        Invoice string `json:"invoice_id"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
        return CreatePaymentResponse{}, err
    }
    if !apiResp.Success {
        return CreatePaymentResponse{}, fmt.Errorf("airbapay: unsuccessful response")
    }
    return CreatePaymentResponse{PaymentURL: apiResp.URL, InvoiceID: apiResp.Invoice}, nil
}

func (c *Client) sign(body []byte) string {
    mac := hmac.New(sha256.New, []byte(c.secret))
    mac.Write(body)
    return hex.EncodeToString(mac.Sum(nil))
}
