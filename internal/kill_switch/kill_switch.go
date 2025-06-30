package kill_switch

import (
	"log"
	"os"
	"sync"
	"time"
)

// Config holds the kill switch configuration.
type Config struct {
	KillTimeout   time.Duration // How long to wait before killing
	Key           string        // Secret key to authorize kill
	KillThreshold int           // Number of kill requests in 1 minute to trigger kill
}

type KillSwitch struct {
	cfg      Config
	mu       sync.Mutex
	requests []time.Time
}

// New creates a new KillSwitch with the given config.
func New(cfg Config) *KillSwitch {
	return &KillSwitch{cfg: cfg}
}

// RegisterKillRequest records a kill request. If the threshold is met, triggers kill.
func (k *KillSwitch) RegisterKillRequest(key string) {
	if key != k.cfg.Key {
		return // Ignore unauthorized
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	now := time.Now()
	// Remove requests older than 1 minute
	cutoff := now.Add(-1 * time.Minute)
	filtered := k.requests[:0]
	for _, t := range k.requests {
		if t.After(cutoff) {
			filtered = append(filtered, t)
		}
	}
	k.requests = append(filtered, now)
	if len(k.requests) >= k.cfg.KillThreshold {
		go k.TriggerKill()
	}
}

// TriggerKill initiates the kill process and exits the process after KillTimeout.
func (k *KillSwitch) TriggerKill() {
	log.Printf("Kill switch triggered! Exiting in %s...", k.cfg.KillTimeout)
	time.Sleep(k.cfg.KillTimeout)
	os.Exit(1)
}
