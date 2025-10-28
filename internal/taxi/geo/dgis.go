package geo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
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
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	payload := struct {
		Points []struct {
			Lat float64 `json:"lat"`
			Lon float64 `json:"lon"`
		} `json:"points"`
		Sources   []int  `json:"sources"`
		Targets   []int  `json:"targets"`
		Transport string `json:"transport,omitempty"`
		Type      string `json:"type,omitempty"`
	}{
		Points: []struct {
			Lat float64 `json:"lat"`
			Lon float64 `json:"lon"`
		}{
			{Lat: fromLat, Lon: fromLon},
			{Lat: toLat, Lon: toLon},
		},
		Sources:   []int{0},
		Targets:   []int{1},
		Transport: "driving",
		Type:      "jam",
	}

	q := url.Values{}
	q.Set("key", c.apiKey)
	q.Set("version", "2.0")
	q.Set("response_format", "json")
	endpoint := fmt.Sprintf("%s/get_dist_matrix?%s", routingBaseURL, q.Encode())

	body, err := json.Marshal(&payload)
	if err != nil {
		return 0, 0, err
	}

	client := c.httpClient
	if client == nil {
		client = &http.Client{}
	}
	clone := *client
	clone.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	client = &clone

	const maxRedirects = 3
	currentURL := endpoint

	for redirects := 0; redirects <= maxRedirects; redirects++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, currentURL, bytes.NewReader(body))
		if err != nil {
			return 0, 0, err
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return 0, 0, err
		}

		if resp.StatusCode == http.StatusNoContent {
			resp.Body.Close()
			return 0, 0, errors.New("2gis: route not found (204)")
		}

		if resp.StatusCode >= 300 && resp.StatusCode < 400 {
			location, err := resp.Location()
			resp.Body.Close()
			if err != nil {
				return 0, 0, fmt.Errorf("2gis: redirect: %w", err)
			}
			currentURL = location.String()
			continue
		}

		if resp.StatusCode >= 300 {
			data, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return 0, 0, fmt.Errorf("2gis: %s: %s", resp.Status, strings.TrimSpace(string(data)))
		}

		var out struct {
			Routes []struct {
				Status   string `json:"status"`
				Distance int    `json:"distance"`
				Duration int    `json:"duration"`
			} `json:"routes"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			resp.Body.Close()
			return 0, 0, err
		}
		resp.Body.Close()
		if len(out.Routes) == 0 {
			return 0, 0, errors.New("2gis: empty routes")
		}
		route := out.Routes[0]
		if strings.ToUpper(route.Status) != "OK" {
			return 0, 0, fmt.Errorf("2gis: status=%s", route.Status)
		}
		return route.Distance, route.Duration, nil
	}

	return 0, 0, errors.New("2gis: too many redirects")
}
