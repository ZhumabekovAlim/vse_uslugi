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
	"strconv"
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

// tryParseLonLat returns lon,lat if query looks like "lon,lat" (WGS84), otherwise (0,0,false).
func tryParseLonLat(query string) (float64, float64, bool) {
	q := strings.TrimSpace(query)
	// Accept comma or semicolon separators
	sep := ","
	if strings.Contains(q, ";") {
		sep = ";"
	}
	parts := strings.Split(q, sep)
	if len(parts) != 2 {
		return 0, 0, false
	}
	lonStr := strings.TrimSpace(parts[0])
	latStr := strings.TrimSpace(parts[1])

	lon, err1 := strconv.ParseFloat(lonStr, 64)
	lat, err2 := strconv.ParseFloat(latStr, 64)
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}
	// quick sanity checks
	if lon < -180 || lon > 180 || lat < -90 || lat > 90 {
		return 0, 0, false
	}
	return lon, lat, true
}

// Geocode returns coordinates for the given query (lon, lat).
func (c *DGISClient) Geocode(ctx context.Context, query string) (float64, float64, error) {
	if strings.TrimSpace(query) == "" {
		return 0, 0, errors.New("geocode: empty query")
	}

	// If user passed "lon,lat" — short-circuit without hitting API
	if lon, lat, ok := tryParseLonLat(query); ok {
		return lon, lat, nil
	}

	// Per-call timeout
	ctx, cancel := context.WithTimeout(ctx, 7*time.Second)
	defer cancel()

	type attempt struct {
		locale string
		typed  bool // whether to send type=building,street
	}

	attempts := []attempt{
		{locale: "ru_KZ", typed: true},
		{locale: "ru_KZ", typed: false},
		{locale: "kk_KZ", typed: true},
		{locale: "kk_KZ", typed: false},
	}

	var lastErr error

	for _, a := range attempts {
		params := url.Values{}
		params.Set("q", query)
		params.Set("key", c.apiKey)
		// Coordinates only if we ask for point:
		params.Set("fields", "items.point")
		// Prefer exact address intents
		if a.typed {
			params.Set("type", "building,street")
		}
		// Help searcher: full query committed by user
		params.Set("search_is_query_text_complete", "true")
		// Locale & region biasing
		params.Set("locale", a.locale)
		if c.regionID != "" {
			params.Set("region_id", c.regionID)
		}

		endpoint := fmt.Sprintf("%s/3.0/items/geocode?%s", catalogBaseURL, params.Encode())
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			lastErr = fmt.Errorf("geocode: build request: %w", err)
			continue
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("geocode: do request: %w", err)
			continue
		}

		func() {
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusNotFound {
				lastErr = fmt.Errorf("geocode: 404 not found (query=%q)", query)
				return
			}
			if resp.StatusCode >= 300 {
				// Read small body for diagnostics
				b, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
				lastErr = fmt.Errorf("geocode: http %s: %s", resp.Status, strings.TrimSpace(string(b)))
				return
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
				lastErr = fmt.Errorf("geocode: decode: %w", err)
				return
			}
			if len(payload.Result.Items) == 0 {
				lastErr = fmt.Errorf("geocode: no results (locale=%s typed=%v)", a.locale, a.typed)
				return
			}
			p := payload.Result.Items[0].Point
			if p.Lon == 0 && p.Lat == 0 {
				lastErr = errors.New("geocode: got zero coordinates")
				return
			}
			// Success
			lon, lat := p.Lon, p.Lat
			lastErr = nil
			// Return via panic/recover trick? Better: shadow return.
			// We'll assign to named results instead? Keep it simple:
			// we can't break two nested func scopes, so capture via closure vars:
			query = fmt.Sprintf("%f,%f", lon, lat) // reuse carrier
		}()

		// If we stuffed lon,lat back into query — treat it as success signal
		if strings.Contains(query, ",") {
			if lon, lat, ok := tryParseLonLat(query); ok {
				return lon, lat, nil
			}
		}
	}

	if lastErr == nil {
		lastErr = errors.New("geocode: unknown error")
	}
	return 0, 0, lastErr
}

// RouteMatrix returns distance and duration between two points.
func (c *DGISClient) RouteMatrix(ctx context.Context, fromLon, fromLat, toLon, toLat float64) (int, int, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
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
		req.Header.Set("Accept", "application/json")

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
