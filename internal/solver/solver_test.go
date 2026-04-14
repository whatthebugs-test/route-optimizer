package solver

import (
	"errors"
	"testing"

	"github.com/paulmach/orb"
)

// Memphis terminal and a handful of real-ish customer locations around it.
var depot = orb.Point{-89.9767, 35.0421}

func memphisStops() []Stop {
	return []Stop{
		{ID: "olive-branch", Location: orb.Point{-89.8295, 34.9618}, WeightLbs: 1800},
		{ID: "southaven", Location: orb.Point{-90.0126, 34.9889}, WeightLbs: 2400},
		{ID: "germantown", Location: orb.Point{-89.8101, 35.0868}, WeightLbs: 900},
		{ID: "bartlett", Location: orb.Point{-89.8740, 35.2045}, WeightLbs: 3100},
		{ID: "downtown", Location: orb.Point{-90.0490, 35.1495}, WeightLbs: 650},
		{ID: "collierville", Location: orb.Point{-89.6645, 35.0420}, WeightLbs: 4200},
	}
}

func TestNearestNeighbourVisitsEveryStopOnce(t *testing.T) {
	routes, err := NearestNeighbour(depot, memphisStops(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(routes) != 1 {
		t.Fatalf("expected one route without capacity limits, got %d", len(routes))
	}
	seen := map[string]bool{}
	for _, s := range routes[0].Stops {
		if seen[s.ID] {
			t.Fatalf("stop %s visited twice", s.ID)
		}
		seen[s.ID] = true
	}
	if len(seen) != len(memphisStops()) {
		t.Fatalf("visited %d of %d stops", len(seen), len(memphisStops()))
	}
}

func TestNearestNeighbourStartsWithClosestStop(t *testing.T) {
	routes, _ := NearestNeighbour(depot, memphisStops(), Options{})
	if got := routes[0].Stops[0].ID; got != "southaven" {
		t.Fatalf("first stop = %s, want southaven", got)
	}
}

func TestCapacitySplitsRoutes(t *testing.T) {
	routes, err := NearestNeighbour(depot, memphisStops(), Options{CapacityLbs: 6000})
	if err != nil {
		t.Fatal(err)
	}
	if len(routes) < 3 {
		t.Fatalf("13,050 lb over 6,000 lb trucks needs at least 3 routes, got %d", len(routes))
	}
	for i, r := range routes {
		if r.LoadLbs > 6000 {
			t.Errorf("route %d carries %.0f lb", i, r.LoadLbs)
		}
	}
}

func TestMaxStops(t *testing.T) {
	routes, _ := NearestNeighbour(depot, memphisStops(), Options{MaxStops: 2})
	for i, r := range routes {
		if len(r.Stops) > 2 {
			t.Errorf("route %d has %d stops", i, len(r.Stops))
		}
	}
}

func TestOverweightStopIsRejected(t *testing.T) {
	_, err := NearestNeighbour(depot, memphisStops(), Options{CapacityLbs: 4000})
	if !errors.Is(err, ErrStopOverweight) {
		t.Fatalf("err = %v, want ErrStopOverweight", err)
	}
}

func TestNoStops(t *testing.T) {
	if _, err := NearestNeighbour(depot, nil, Options{}); !errors.Is(err, ErrNoStops) {
		t.Fatalf("err = %v, want ErrNoStops", err)
	}
}

func TestDistanceMemphisToNashville(t *testing.T) {
	d := Distance(orb.Point{-90.0490, 35.1495}, orb.Point{-86.7816, 36.1627})
	if d < 310 || d > 320 {
		t.Fatalf("distance = %.1f km, want ~315", d)
	}
}
