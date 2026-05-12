// Package store reads stops from and writes route plans to the dispatch Postgres.
package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/paulmach/orb"

	"github.com/kestrelfreight/route-optimizer/internal/solver"
)

var ErrNotFound = errors.New("store: not found")

type Store struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, databaseURL string) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	cfg.MaxConns = 8
	cfg.MaxConnIdleTime = 5 * time.Minute
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() { s.pool.Close() }

func (s *Store) Ping(ctx context.Context) error { return s.pool.Ping(ctx) }

// Depot returns the terminal's dock location.
func (s *Store) Depot(ctx context.Context, terminal string) (orb.Point, error) {
	var lat, lng float64
	err := s.pool.QueryRow(ctx, `SELECT lat, lng FROM terminals WHERE code = $1`, terminal).Scan(&lat, &lng)
	if errors.Is(err, pgx.ErrNoRows) {
		return orb.Point{}, ErrNotFound
	}
	return orb.Point{lng, lat}, err
}

// PickupStops returns unassigned and offered pickups for a terminal and day.
func (s *Store) PickupStops(ctx context.Context, terminal string, day time.Time) ([]solver.Stop, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT shipment_id, pickup_lat, pickup_lng, weight_lbs
		FROM assignments
		WHERE terminal = $1 AND pickup_date = $2 AND status IN ('UNASSIGNED', 'OFFERED', 'ACCEPTED')
		ORDER BY shipment_id`, terminal, day.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (solver.Stop, error) {
		var st solver.Stop
		var lat, lng float64
		err := row.Scan(&st.ID, &lat, &lng, &st.WeightLbs)
		st.Location = orb.Point{lng, lat}
		st.ServiceMinutes = 20
		return st, err
	})
}

// Plan is a saved set of routes for one terminal-day.
type Plan struct {
	ID        string         `json:"id"`
	Terminal  string         `json:"terminal"`
	Day       string         `json:"day"`
	Routes    []solver.Route `json:"routes"`
	CreatedAt time.Time      `json:"created_at"`
}

func (s *Store) SavePlan(ctx context.Context, p Plan) error {
	body, err := json.Marshal(p.Routes)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO route_plans (id, terminal, day, routes, created_at)
		VALUES ($1, $2, $3, $4, $5)`, p.ID, p.Terminal, p.Day, body, p.CreatedAt)
	return err
}

func (s *Store) Plan(ctx context.Context, id string) (Plan, error) {
	var p Plan
	var body []byte
	var day time.Time
	err := s.pool.QueryRow(ctx, `SELECT id, terminal, day, routes, created_at FROM route_plans WHERE id = $1`, id).
		Scan(&p.ID, &p.Terminal, &day, &body, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, ErrNotFound
	}
	if err != nil {
		return p, err
	}
	p.Day = day.Format("2006-01-02")
	return p, json.Unmarshal(body, &p.Routes)
}
