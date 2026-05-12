package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"

	"github.com/kestrelfreight/route-optimizer/internal/cache"
	"github.com/kestrelfreight/route-optimizer/internal/solver"
	"github.com/kestrelfreight/route-optimizer/internal/store"
)

type Server struct {
	Store *store.Store
	Cache *cache.Cache
}

type optimizeRequest struct {
	Terminals   []string `json:"terminals" binding:"required,min=1,max=20,dive,len=3,numeric"`
	Day         string   `json:"day" binding:"required,datetime=2006-01-02"`
	CapacityLbs float64  `json:"capacity_lbs" binding:"gte=0,lte=48000"`
	MaxStops    int      `json:"max_stops" binding:"gte=0,lte=60"`
}

func (s *Server) Routes(r *gin.Engine, secret []byte) {
	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	r.GET("/readyz", s.ready)

	v1 := r.Group("/v1", RequireServiceToken(secret, "routes:plan"))
	v1.POST("/routes/optimize", s.optimize)
	v1.GET("/routes/:id", s.getPlan)
}

func (s *Server) ready(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()
	if err := s.Store.Ping(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "db unavailable"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}

func (s *Server) optimize(c *gin.Context) {
	var req optimizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	day, _ := time.Parse("2006-01-02", req.Day)
	opts := solver.Options{CapacityLbs: req.CapacityLbs, MaxStops: req.MaxStops, Improve: true}

	plans := make([]store.Plan, len(req.Terminals))
	g, ctx := errgroup.WithContext(c.Request.Context())
	g.SetLimit(4)
	for i, terminal := range req.Terminals {
		g.Go(func() error {
			plan, err := s.planTerminal(ctx, terminal, day, opts)
			plans[i] = plan
			return err
		})
	}
	if err := g.Wait(); err != nil {
		switch {
		case errors.Is(err, store.ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "unknown terminal"})
		case errors.Is(err, solver.ErrStopOverweight):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		default:
			log.Error().Err(err).Msg("optimize failed")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"plans": plans})
}

func (s *Server) planTerminal(ctx context.Context, terminal string, day time.Time, opts solver.Options) (store.Plan, error) {
	depot, err := s.Store.Depot(ctx, terminal)
	if err != nil {
		return store.Plan{}, err
	}
	stops, err := s.Store.PickupStops(ctx, terminal, day)
	if err != nil {
		return store.Plan{}, err
	}
	plan := store.Plan{ID: newPlanID(), Terminal: terminal, Day: day.Format("2006-01-02"), CreatedAt: time.Now().UTC()}
	if len(stops) == 0 {
		return plan, nil
	}

	key := cache.Key(terminal, stops, opts)
	if routes, ok := s.Cache.Get(ctx, key); ok {
		plan.Routes = routes
		return plan, nil
	}
	start := time.Now()
	routes, err := solver.NearestNeighbour(depot, stops, opts)
	if err != nil {
		return plan, err
	}
	log.Info().Str("terminal", terminal).Int("stops", len(stops)).Int("routes", len(routes)).
		Dur("took", time.Since(start)).Msg("solved")
	plan.Routes = routes
	if err := s.Cache.Put(ctx, key, routes); err != nil {
		log.Warn().Err(err).Msg("cache put failed")
	}
	return plan, s.Store.SavePlan(ctx, plan)
}

func (s *Server) getPlan(c *gin.Context) {
	plan, err := s.Store.Plan(c.Request.Context(), c.Param("id"))
	if errors.Is(err, store.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, plan)
}

func newPlanID() string {
	b := make([]byte, 10)
	_, _ = rand.Read(b)
	return "rp_" + hex.EncodeToString(b)
}
