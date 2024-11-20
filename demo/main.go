package main

import (
	"context"
	"demo/pkg/logger"
	"demo/pkg/metrics"
	"demo/pkg/traces"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/exp/rand"
)

func Observe() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// context correlation
		correlationID := uuid.New().String()
		reqLog := logger.G(c.Request.Context()).WithFields(logrus.Fields{
			"correlation_id": correlationID,
			"method":         c.Request.Method,
			"path":           c.Request.URL.Path,
			"remote_ip":      c.ClientIP(),
		})
		c.Request = c.Request.WithContext(logger.WithLogger(c.Request.Context(), reqLog))

		defer func() {
			latency := time.Since(start)

			// logging
			reqLog := logger.G(c.Request.Context())
			reqLog.WithFields(logrus.Fields{
				"latency_ms":  latency.Milliseconds(),
				"status_code": c.Writer.Status(),
				"status":      c.Writer.Status(),
			}).Info("request completed")

			// metrics
			metrics.HttpRequestsTotal.WithLabelValues(
				fmt.Sprintf("%d", c.Writer.Status()),
				c.Request.Method,
				c.Request.URL.Path,
			).Inc()

			metrics.HttpRequestDuration.WithLabelValues(
				fmt.Sprintf("%d", c.Writer.Status()),
				c.Request.Method,
				c.Request.URL.Path,
			).Observe(latency.Seconds())
		}()
		c.Next()
	}
}

type Svc struct {
	Client *http.Client
}

func (s *Svc) Healthz() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "OK"})
	}
}

func (s *Svc) Hello() gin.HandlerFunc {
	return func(c *gin.Context) {
		naughty := c.Query("naughty") == "true"
		ctx, span := traces.Tracer.Start(c.Request.Context(), "demo.Hello", trace.WithAttributes(
			attribute.String("demo.hello", "world"),
			attribute.Bool("demo.naughty", naughty),
		))

		defer span.End()

		reqLog := logger.G(ctx)
		reqLog.Info("Hello World")

		if err := s.makeReqToGoogle(ctx); err != nil {
			span.SetStatus(codes.Error, err.Error())
			reqLog.WithError(err).Error("failed to make http request to google")
			c.AbortWithError(http.StatusInternalServerError, errors.New("something goes wrong"))
			return
		}

		if err := doSomethingNaughty(ctx, naughty); err != nil {
			span.SetStatus(codes.Error, err.Error())
			reqLog.WithError(err).Error("failed to do something naughty")
			c.AbortWithError(http.StatusInternalServerError, errors.New("something goes wrong"))
			return
		}

		reqLog.Info("request to google completed")

		c.JSON(http.StatusOK, gin.H{"message": "Hello, World!"})
	}
}

func (s *Svc) makeReqToGoogle(ctx context.Context) error {
	ctx, span := traces.Tracer.Start(ctx, "demo.makeReqToGoogle")
	defer span.End()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.google.com", nil)
	if err != nil {
		return err
	}

	resp, err := s.Client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	return nil
}

func doSomethingNaughty(ctx context.Context, naughty bool) error {
	_, span := traces.Tracer.Start(ctx, "demo.doSomethingNaughty")
	defer span.End()

	if !naughty {
		if span.IsRecording() {
			span.SetAttributes(attribute.String("demo.naughty", "skipped"))
		}
		return nil
	}

	// Simulate CPU spike by performing a large number of calculations

	for j := 0; j < 1e8; j++ {
		r1 := rand.Intn(100)
		r2 := rand.Intn(100)
		_ = r1 * r2
	}

	return nil
}

func NewRouter() *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery(), otelgin.Middleware("demo"), Observe())
	svc := &Svc{
		Client: &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)},
	}
	router.GET("/healthz", svc.Healthz())
	router.GET("/readyz", svc.Healthz())
	router.GET("/hello", svc.Hello())

	return router
}

func main() {
	ctx := context.Background()

	metrics.Init(ctx)

	// Set up OpenTelemetry.
	otelShutdown, err := traces.SetupOTelSDK(ctx)
	if err != nil {
		logger.G(ctx).Warn("Cannot setup Otel")
	}

	// Handle shutdown properly so nothing leaks.
	defer func() {
		if err := otelShutdown(ctx); err != nil {
			logger.G(ctx).Error("error with shutting down the otel collector")
		}
	}()

	// Start HTTP server.
	router := NewRouter()
	router.Run(":8080")
}
