package geo

import (
    "bytes"
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "net/http"
    "net/url"
    "time"
)

const (
    catalogBaseURL = "https://catalog.api.2gis.com"
    routingBaseURL = "https://routing.api.2gis.com"
)

// DGISClient provides access to 2GIS APIs.
type DGISClient struct {
    httpClient *http.Client
    apiKey     string
    regionID   string
}

// NewDGISClient constructs a new 2GIS client.
func NewDGISClient(httpClient *http.Client, apiKey, regionID string) *DGISClient {
    if httpClient == nil {
        httpClient = &http.Client{Timeout: 5 * time.Second}
    }
    return &DGISClient{httpClient: httpClient, apiKey: apiKey, regionID: regionID}
}

// Geocode returns coordinates for the given query.
func (c *DGISClient) Geocode(ctx context.Context, query string) (float64, float64, error) {
    if query == "" {
        return 0, 0, errors.New("empty query")
    }
    ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
    defer cancel()

    params := url.Values{}
    params.Set("q", query)
    params.Set("key", c.apiKey)
    if c.regionID != "" {
        params.Set("region_id", c.regionID)
    }

    endpoint := fmt.Sprintf("%s/3.0/items/geocode?%s", catalogBaseURL, params.Encode())
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
    if err != nil {
        return 0, 0, err
    }

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return 0, 0, err
    }
    defer resp.Body.Close()
    if resp.StatusCode >= 300 {
        return 0, 0, fmt.Errorf("dgis geocode status %s", resp.Status)
    }

    var payload struct {
        Result struct {
            Items []struct {
                Point struct {
                    Lon float64 `json:"lon"`
                    Lat float64 `json:"lat"`
                } `json:"point"`
            } `json:"items"`
        } `json:"result"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
        return 0, 0, err
    }
    if len(payload.Result.Items) == 0 {
        return 0, 0, errors.New("dgis: no results")
    }
    p := payload.Result.Items[0].Point
    return p.Lon, p.Lat, nil
}

// RouteMatrix returns distance and duration between two points.
func (c *DGISClient) RouteMatrix(ctx context.Context, fromLon, fromLat, toLon, toLat float64) (int, int, error) {
    ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
    defer cancel()

    params := url.Values{}
    params.Set("key", c.apiKey)
    params.Set("type", "driving")

    body := map[string]interface{}{
        "points": []map[string]float64{
            {"lon": fromLon, "lat": fromLat},
            {"lon": toLon, "lat": toLat},
        },
    }

    reqBody, err := json.Marshal(body)
    if err != nil {
        return 0, 0, err
    }

    endpoint := fmt.Sprintf("%s/4.0/matrix?%s", routingBaseURL, params.Encode())
    req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(reqBody))
    if err != nil {
        return 0, 0, err
    }
    req.Header.Set("Content-Type", "application/json")

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return 0, 0, err
    }
    defer resp.Body.Close()
    if resp.StatusCode >= 300 {
        return 0, 0, fmt.Errorf("dgis matrix status %s", resp.Status)
    }

    var payload struct {
        Routes []struct {
            Distance int `json:"distance"`
            Duration int `json:"duration"`
        } `json:"routes"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
        return 0, 0, err
    }
    if len(payload.Routes) == 0 {
        return 0, 0, errors.New("dgis: empty matrix")
    }
    r := payload.Routes[0]
    return r.Distance, r.Duration, nil
}
