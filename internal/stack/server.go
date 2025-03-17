package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/edgeflare/pgo/pkg/httputil"
)

const (
	StatusHealthy   = "healthy"
	StatusUnhealthy = "unhealthy"
	StatusUnknown   = "unknown"
)

type ServiceHealth struct {
	Status    string    `json:"status"`
	LastCheck time.Time `json:"lastCheck"`
	Error     string    `json:"error,omitempty"`
}

type SystemHealth struct {
	Status    string                   `json:"status"`
	Services  map[string]ServiceHealth `json:"services"`
	Timestamp time.Time                `json:"timestamp"`
}

// HealthChecker manages health checks for various services
type HealthChecker struct {
	services      map[string]func() (bool, error)
	healthStatus  map[string]ServiceHealth
	mutex         sync.RWMutex
	checkInterval time.Duration
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(checkInterval time.Duration) *HealthChecker {
	return &HealthChecker{
		services:      make(map[string]func() (bool, error)),
		healthStatus:  make(map[string]ServiceHealth),
		checkInterval: checkInterval,
	}
}

// RegisterService adds a service to be monitored
func (hc *HealthChecker) RegisterService(name string, checkFn func() (bool, error)) {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()

	hc.services[name] = checkFn
	hc.healthStatus[name] = ServiceHealth{
		Status:    StatusUnknown,
		LastCheck: time.Now(),
	}
}

// CheckHealth checks the health of all registered services
func (hc *HealthChecker) CheckHealth() {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()

	for name, checkFn := range hc.services {
		healthy, err := checkFn()
		now := time.Now()

		status := StatusUnhealthy
		errMsg := ""

		if healthy {
			status = StatusHealthy
		} else if err != nil {
			errMsg = err.Error()
		}

		hc.healthStatus[name] = ServiceHealth{
			Status:    status,
			LastCheck: now,
			Error:     errMsg,
		}
	}
}

// CheckServiceHealth checks the health of a specific service
func (hc *HealthChecker) CheckServiceHealth(name string) (ServiceHealth, bool) {
	hc.mutex.RLock()
	checkFn, exists := hc.services[name]
	hc.mutex.RUnlock()

	if !exists {
		return ServiceHealth{}, false
	}

	hc.mutex.Lock()
	defer hc.mutex.Unlock()

	healthy, err := checkFn()
	now := time.Now()

	status := StatusUnhealthy
	errMsg := ""

	if healthy {
		status = StatusHealthy
	} else if err != nil {
		errMsg = err.Error()
	}

	health := ServiceHealth{
		Status:    status,
		LastCheck: now,
		Error:     errMsg,
	}

	hc.healthStatus[name] = health

	return health, true
}

// GetServiceHealth returns the health status of a specific service
// If the service is unhealthy, it performs an immediate health check before returning
func (hc *HealthChecker) GetServiceHealth(name string) (ServiceHealth, bool) {
	hc.mutex.RLock()
	health, exists := hc.healthStatus[name]
	hc.mutex.RUnlock()

	if !exists {
		return ServiceHealth{}, false
	}

	// If the service is unhealthy, perform an immediate health check
	if health.Status != StatusHealthy {
		return hc.CheckServiceHealth(name)
	}

	return health, true
}

// GetSystemHealth returns the health status of all services
func (hc *HealthChecker) GetSystemHealth() SystemHealth {
	servicesCopy := make(map[string]ServiceHealth)
	unhealthyServices := []string{}

	hc.mutex.RLock()
	for name, health := range hc.healthStatus {
		if health.Status != StatusHealthy {
			unhealthyServices = append(unhealthyServices, name)
		} else {
			servicesCopy[name] = health
		}
	}
	hc.mutex.RUnlock()

	for _, name := range unhealthyServices {
		health, exists := hc.CheckServiceHealth(name)
		if exists {
			servicesCopy[name] = health
		}
	}

	systemStatus := StatusHealthy
	for _, v := range servicesCopy {
		if v.Status != StatusHealthy {
			systemStatus = StatusUnhealthy
			break
		}
	}

	return SystemHealth{
		Status:    systemStatus,
		Services:  servicesCopy,
		Timestamp: time.Now(),
	}
}

// StartChecking begins periodic health checking
func (hc *HealthChecker) StartChecking() {
	ticker := time.NewTicker(hc.checkInterval)
	go func() {
		for {
			<-ticker.C
			hc.CheckHealth()
		}
	}()
}

// registerZitadelCheck adds Zitadel health check to the health checker
func registerZitadelCheck(hc *HealthChecker, host string) {
	hc.RegisterService("zitadel", func() (bool, error) {
		resp, err := http.Get(fmt.Sprintf("http://%s/debug/healthz", host))
		if err != nil {
			return false, err
		}
		defer resp.Body.Close()

		return resp.StatusCode == http.StatusOK, nil
	})
}

// serve starts the health check API server
func serve(port int, zitadelHost string) {
	healthChecker := NewHealthChecker(30 * time.Second)
	registerZitadelCheck(healthChecker, zitadelHost)
	healthChecker.StartChecking()

	mux := http.NewServeMux()

	// Overall health endpoint
	mux.Handle("GET /healthz", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		health := healthChecker.GetSystemHealth()

		if health.Status != StatusHealthy {
			httputil.JSON(w, http.StatusServiceUnavailable, nil)
		}
		httputil.JSON(w, http.StatusOK, health)
	}))

	// Individual service health endpoints
	mux.HandleFunc("GET /healthz/", func(w http.ResponseWriter, r *http.Request) {
		serviceName := r.URL.Path[len("/healthz/"):]
		if serviceName == "" {
			http.NotFound(w, r)
			return
		}

		health, exists := healthChecker.GetServiceHealth(serviceName)
		if !exists {
			http.NotFound(w, r)
			return
		}

		if health.Status != StatusHealthy {
			httputil.JSON(w, http.StatusServiceUnavailable, nil)
		}
		httputil.JSON(w, http.StatusOK, health)
	})

	// Start the HTTP server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	log.Printf("Starting health check service on :%d\n", port)
	log.Fatal(server.ListenAndServe())
}

// checkHealth checks the health endpoint and returns proper exit code
func checkHealth(endpoint string) error {
	resp, err := http.Get(endpoint)
	if err != nil {
		return fmt.Errorf("error checking health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unhealthy status: %s", string(body))
	}

	fmt.Println("Health check passed!")
	return nil
}
