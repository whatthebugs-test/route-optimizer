// Package solver builds pickup and delivery routes for a terminal's day.
//
// The planner is a nearest-neighbour construction split into vehicle loads
// by capacity. It is deliberately
// simple: city P&D routes are 8 to 25 stops, and dispatchers re-sequence by
// hand anyway. Quality is measured against the hand-built routes in the
// weekly planning report, not against an exact solver.
package solver

import (
	"errors"
	"math"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geo"
)

// Stop is a pickup or delivery location.
type Stop struct {
	ID        string    `json:"id"`
	Location  orb.Point `json:"location"` // [lng, lat]
	WeightLbs float64   `json:"weight_lbs"`
	// ServiceMinutes is dock time at the stop.
	ServiceMinutes float64 `json:"service_minutes"`
}

// Route is an ordered list of stops that starts and ends at the depot.
type Route struct {
	Stops      []Stop  `json:"stops"`
	DistanceKm float64 `json:"distance_km"`
	LoadLbs    float64 `json:"load_lbs"`
}

// Options control route construction.
type Options struct {
	// CapacityLbs is the per-vehicle payload limit. Zero means unlimited.
	CapacityLbs float64
	// MaxStops caps stops per route. Zero means unlimited.
	MaxStops int
}

var (
	ErrNoStops        = errors.New("solver: no stops")
	ErrStopOverweight = errors.New("solver: stop exceeds vehicle capacity")
)

// Distance returns the great-circle distance in kilometres.
func Distance(a, b orb.Point) float64 {
	return geo.DistanceHaversine(a, b) / 1000
}

// NearestNeighbour sequences stops greedily from the depot, opening a new
// route whenever the next stop would exceed capacity or the stop limit.
func NearestNeighbour(depot orb.Point, stops []Stop, opts Options) ([]Route, error) {
	if len(stops) == 0 {
		return nil, ErrNoStops
	}
	for _, s := range stops {
		if opts.CapacityLbs > 0 && s.WeightLbs > opts.CapacityLbs {
			return nil, ErrStopOverweight
		}
	}

	visited := make([]bool, len(stops))
	remaining := len(stops)
	var routes []Route

	for remaining > 0 {
		route := Route{}
		pos := depot
		for {
			next := -1
			best := math.Inf(1)
			for i, s := range stops {
				if visited[i] {
					continue
				}
				if opts.CapacityLbs > 0 && route.LoadLbs+s.WeightLbs > opts.CapacityLbs {
					continue
				}
				if d := Distance(pos, s.Location); d < best {
					best, next = d, i
				}
			}
			if next < 0 || (opts.MaxStops > 0 && len(route.Stops) >= opts.MaxStops) {
				break
			}
			visited[next] = true
			remaining--
			route.Stops = append(route.Stops, stops[next])
			route.LoadLbs += stops[next].WeightLbs
			pos = stops[next].Location
		}
		route.DistanceKm = TourLength(depot, route.Stops)
		routes = append(routes, route)
	}
	return routes, nil
}

// TourLength is the depot -> stops -> depot distance in kilometres.
func TourLength(depot orb.Point, stops []Stop) float64 {
	if len(stops) == 0 {
		return 0
	}
	total := Distance(depot, stops[0].Location)
	for i := 1; i < len(stops); i++ {
		total += Distance(stops[i-1].Location, stops[i].Location)
	}
	return total + Distance(stops[len(stops)-1].Location, depot)
}
