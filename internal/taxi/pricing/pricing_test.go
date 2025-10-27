package pricing

import "testing"

func TestRecommended(t *testing.T) {
    cases := []struct {
        name        string
        distance    int
        pricePerKM  int
        minPrice    int
        want        int
    }{
        {"zero distance", 0, 300, 1200, 1200},
        {"below min", 1000, 100, 1500, 1500},
        {"rounding", 2500, 400, 800, 1000},
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            got := Recommended(tc.distance, tc.pricePerKM, tc.minPrice)
            if got != tc.want {
                t.Fatalf("expected %d got %d", tc.want, got)
            }
        })
    }
}
