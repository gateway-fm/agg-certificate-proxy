package certificate

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"log/slog"
	"math/big"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/crypto/bcrypt"
)

//go:embed templates
var templateFolder embed.FS

var homepageTemplate *template.Template

func init() {
	parsed, err := template.ParseFS(templateFolder, "templates/index.tmpl.html")
	if err != nil {
		log.Fatalf("could not parse the homepage template file: %s", err)
	}
	homepageTemplate = parsed
}

// APIServer handles HTTP requests.
type APIServer struct {
	service        *Service
	metricsUpdater MetricsUpdater
}

// NewAPIServer creates a new API server.
func NewAPIServer(service *Service, metricsUpdater MetricsUpdater) *APIServer {
	return &APIServer{service: service, metricsUpdater: metricsUpdater}
}

// RegisterHandlers registers the HTTP handlers.
func (s *APIServer) RegisterHandlers() {
	http.HandleFunc("/", s.viewCerts)
	http.HandleFunc("/config", s.viewConfig)
	http.HandleFunc("/kill", s.handleKillSwitch)
	http.HandleFunc("/restart", s.handleRestart)
	http.HandleFunc("/override", s.handleOverride)
}

// CertificateView extends Certificate with calculated fields for display
type CertificateView struct {
	Certificate
	NetworkID               uint32
	Height                  uint64
	WillSendAt              time.Time
	Tokens                  []TokenExit
	BridgeExitCount         int
	ImportedBridgeExitCount int
	PrettyMetadata          string
	FullPrettyMetadata      string
	HasFullMetadata         bool
}

type TokenExit struct {
	TokenAddress    string
	Amount          *big.Int
	AmountFormatted string
}

// prettyPrintJSON pretty-prints JSON data with indentation
func prettyPrintJSON(jsonStr string) string {
	var data interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return jsonStr // Return original if parsing fails
	}

	pretty, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return jsonStr // Return original if formatting fails
	}

	return string(pretty)
}

// truncateJSON creates a smart truncated version of JSON, showing key structure
func truncateJSON(data map[string]interface{}, maxLines int) (string, bool) {
	// Create a summary version with counts for arrays
	summary := make(map[string]interface{})
	lineCount := 0

	for key, value := range data {
		if lineCount >= maxLines {
			summary["..."] = "more fields truncated"
			break
		}

		switch v := value.(type) {
		case []interface{}:
			if len(v) > 0 {
				summary[key+"_count"] = len(v)
				// Show first item as example if it's not too large
				if len(v) > 0 && lineCount < maxLines-2 {
					summary[key+"_example"] = v[0]
				}
				lineCount += 2
			} else {
				summary[key] = v
				lineCount++
			}
		case map[string]interface{}:
			// For nested objects, just show keys
			keys := make([]string, 0, len(v))
			for k := range v {
				keys = append(keys, k)
			}
			summary[key+"_keys"] = keys
			lineCount++
		default:
			summary[key] = value
			lineCount++
		}
	}

	summaryJSON, _ := json.MarshalIndent(summary, "", "  ")
	return string(summaryJSON), lineCount < len(data)
}

// formatAmount converts wei to a human-readable format with units
func formatAmount(wei *big.Int) string {
	if wei == nil {
		return "0 wei"
	}

	// Define unit thresholds
	oneGwei := big.NewInt(1e9)       // 1 Gwei = 10^9 wei
	thousandGwei := big.NewInt(1e12) // 1000 Gwei = 10^12 wei (0.001 ETH)
	oneEth := big.NewInt(1e18)       // 1 ETH = 10^18 wei

	// For very small amounts, show in wei
	if wei.Cmp(oneGwei) < 0 {
		return fmt.Sprintf("%s wei", wei.String())
	}

	// For medium amounts (less than 1000 Gwei), show in Gwei
	if wei.Cmp(thousandGwei) < 0 {
		gwei := new(big.Float).SetInt(wei)
		gwei.Quo(gwei, new(big.Float).SetInt(oneGwei))
		return fmt.Sprintf("%.3f Gwei", gwei)
	}

	// For large amounts (1000 Gwei or more), show in ETH
	eth := new(big.Float).SetInt(wei)
	eth.Quo(eth, new(big.Float).SetInt(oneEth))
	return fmt.Sprintf("%.6f ETH", eth)
}

// formatDuration converts seconds to human-readable duration
func formatDuration(seconds int) string {
	duration := time.Duration(seconds) * time.Second
	return duration.String()
}

type CertificateData struct {
	Config          Config                `json:"config"`
	SchedulerActive bool                  `json:"scheduler_active"`
	ChainTotals     map[uint32]*ChainInfo `json:"chain_totals"`
	Certificates    []CertificateView     `json:"certificates"`
}

type Config struct {
	DelayHours  string `json:"delay_hours"`
	Delay       string `json:"delay"`
	CurrentTime string `json:"current_time"`
}

type ChainInfo struct {
	TotalAmount    *big.Int `json:"total_amount"`
	FormattedTotal string   `json:"formatted_total"`
	CertCount      int      `json:"cert_count"`
}

func (s *APIServer) viewCerts(w http.ResponseWriter, r *http.Request) {
	// lets check for an api key here
	apiKey := r.URL.Query().Get("key")
	if apiKey == "" {
		slog.Error("missing API key")
		http.Error(w, "missing API key", http.StatusUnauthorized)
		return
	}

	// check if the api key is valid
	storedKey, err := s.service.db.GetCredential("data_key")
	if err != nil {
		slog.Error("error retrieving data key", "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(storedKey), []byte(apiKey))
	if err != nil {
		slog.Error("invalid API key", "err", err)
		http.Error(w, "invalid API key", http.StatusUnauthorized)
		return
	}

	data, err := s.loadCertificateData()
	if err != nil {
		slog.Error("failed to load certificate data", "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// lets check what kind of content we should return here
	contentType := r.Header.Get("Accept")

	switch contentType {
	case "application/json":
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(data); err != nil {
			slog.Error("failed to encode certificate data", "err", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
	default:
		err = homepageTemplate.Execute(w, data)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to execute template: %v", err), http.StatusInternalServerError)
		}
	}
}

func (s *APIServer) loadCertificateData() (CertificateData, error) {
	schedulerActive, err := s.service.db.GetSchedulerStatus()
	if err != nil {
		slog.Error("failed to get scheduler status", "err", err)
		schedulerActive = true // Default to active if error
	}

	delayStr, err := s.service.GetConfigValue("delay_seconds")
	if err != nil {
		return CertificateData{}, err
	}
	delaySeconds, _ := strconv.Atoi(delayStr)
	delayDuration := time.Duration(delaySeconds) * time.Second

	certs, err := s.service.GetCertificates()
	if err != nil {
		slog.Error("failed to get certificates", "err", err)
		return CertificateData{}, err
	}

	chainTotals := make(map[uint32]*ChainInfo)

	// Create views with calculated fields
	certViews := make([]CertificateView, 0, len(certs))
	for _, cert := range certs {
		view := CertificateView{
			Certificate: cert,
			WillSendAt:  cert.ReceivedAt.Add(delayDuration),
		}

		// Parse metadata to extract fields
		if cert.Metadata != "" {
			// Pretty print the full metadata
			view.FullPrettyMetadata = prettyPrintJSON(cert.Metadata)

			var meta map[string]interface{}
			if err := json.Unmarshal([]byte(cert.Metadata), &meta); err == nil {
				// Extract basic fields
				if networkID, ok := meta["network_id"].(float64); ok {
					view.NetworkID = uint32(networkID)
				}
				if height, ok := meta["height"].(float64); ok {
					view.Height = uint64(height)
				}

				// Calculate total amount from bridge exits and imported bridge exits
				tokenTotals := make(map[string]*TokenExit)

				// Count and sum bridge exits
				if bridgeExits, ok := meta["bridge_exits"].([]interface{}); ok {
					view.BridgeExitCount = len(bridgeExits)
					for _, be := range bridgeExits {
						if beMap, ok := be.(map[string]interface{}); ok {
							token := ""
							if tokenAddress, ok := beMap["token_address"].(string); ok {
								token = tokenAddress
							}
							if amountStr, ok := beMap["amount"].(string); ok {
								if amount, err := strconv.ParseUint(amountStr, 10, 64); err == nil {
									asBig := big.NewInt(0).SetUint64(amount)
									if t, ok := tokenTotals[token]; !ok {
										tokenTotals[token] = &TokenExit{TokenAddress: token, Amount: asBig, AmountFormatted: formatAmount(asBig)}
									} else {
										t.Amount.Add(t.Amount, asBig)
										t.AmountFormatted = formatAmount(t.Amount)
									}
								}
							}
						}
					}
				}
				if beCount, ok := meta["bridge_exits_count"].(float64); ok {
					view.BridgeExitCount = int(beCount)
				}

				// Count and sum imported bridge exits
				if importedExits, ok := meta["imported_bridge_exits"].([]interface{}); ok {
					view.ImportedBridgeExitCount = len(importedExits)
					for _, ibe := range importedExits {
						if ibeMap, ok := ibe.(map[string]interface{}); ok {
							token := ""
							if tokenAddress, ok := ibeMap["token_address"].(string); ok {
								token = tokenAddress
							}
							if amountStr, ok := ibeMap["amount"].(string); ok {
								if amount, err := strconv.ParseUint(amountStr, 10, 64); err == nil {
									asBig := big.NewInt(0).SetUint64(amount)
									if t, ok := tokenTotals[token]; !ok {
										tokenTotals[token] = &TokenExit{TokenAddress: token, Amount: asBig, AmountFormatted: formatAmount(asBig)}
									} else {
										t.Amount.Add(t.Amount, asBig)
										t.AmountFormatted = formatAmount(t.Amount)
									}
								}
							}
						}
					}
				}
				if ibeCount, ok := meta["imported_bridge_exits_count"].(float64); ok {
					view.ImportedBridgeExitCount = int(ibeCount)
				}

				view.Tokens = make([]TokenExit, 0)
				for _, t := range tokenTotals {
					view.Tokens = append(view.Tokens, *t)
				}

				// Create truncated version for display
				truncated, wasTruncated := truncateJSON(meta, 8)
				view.PrettyMetadata = truncated
				view.HasFullMetadata = wasTruncated || len(view.FullPrettyMetadata) > 500

				// Add to chain totals if this is a pending certificate
				if !cert.ProcessedAt.Valid && view.NetworkID > 0 {
					if chainTotals[view.NetworkID] == nil {
						chainTotals[view.NetworkID] = &ChainInfo{TotalAmount: big.NewInt(0), CertCount: 0}
					}
					chainInfo := chainTotals[view.NetworkID]
					chainInfo.CertCount++

					// Parse and add to total
					for _, t := range tokenTotals {
						chainInfo.TotalAmount.Add(chainInfo.TotalAmount, t.Amount)
					}
				}
			} else {
				// If parsing fails, just show the raw metadata
				view.PrettyMetadata = cert.Metadata
			}
		}

		certViews = append(certViews, view)
	}

	// Format chain totals after accumulation
	for _, info := range chainTotals {
		info.FormattedTotal = formatAmount(info.TotalAmount)
	}

	// Prepare template data
	data := CertificateData{
		Config: Config{
			DelayHours:  delayStr, // Keep for backward compatibility in template
			Delay:       formatDuration(delaySeconds),
			CurrentTime: time.Now().Format("2006-01-02 15:04:05"),
		},
		SchedulerActive: schedulerActive,
		ChainTotals:     chainTotals,
		Certificates:    certViews,
	}

	return data, nil
}

func (s *APIServer) viewConfig(w http.ResponseWriter, r *http.Request) {
	delay, err := s.service.GetConfigValue("delay_seconds")
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get config: %v", err), http.StatusInternalServerError)
		return
	}

	config := map[string]string{"delay_seconds": delay}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(config); err != nil {
		http.Error(w, fmt.Sprintf("failed to encode config: %v", err), http.StatusInternalServerError)
	}
}

// handleKillSwitch handles the kill switch endpoint
func (s *APIServer) handleKillSwitch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get the API key from query parameter
	apiKey := r.URL.Query().Get("key")
	if apiKey == "" {
		http.Error(w, "missing API key", http.StatusUnauthorized)
		return
	}

	// Check if the API key matches the stored kill switch key
	storedKey, err := s.service.db.GetCredential("kill_switch_api_key")
	if err != nil {
		slog.Error("retrieving kill switch API key", "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(storedKey), []byte(apiKey))
	if err != nil {
		slog.Error("generating hashed kill switch API key", "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Record the kill attempt
	if err := s.service.db.RecordKillSwitchAttempt("kill"); err != nil {
		slog.Error("error recording kill switch attempt", "err", err)
	}

	// Clean up old attempts (older than 5 minutes)
	if err := s.service.db.CleanupOldKillSwitchAttempts(5 * time.Minute); err != nil {
		slog.Error("error cleaning up old kill switch attempts", "err", err)
	}

	// Check if we have 3 attempts in the last minute
	count, err := s.service.db.GetRecentKillSwitchAttempts("kill", time.Minute)
	if err != nil {
		slog.Error("error checking recent kill switch attempts", "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if count >= 3 {
		// Kill the scheduler
		if err := s.service.db.SetSchedulerStatus(false); err != nil {
			slog.Error("setting scheduler status", "err", err)
			http.Error(w, "failed to kill scheduler", http.StatusInternalServerError)
			return
		}

		slog.Info("kill switch activated - scheduler stopped")
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{
			"status":  "killing scheduler",
			"message": "Scheduler has been stopped",
		}); err != nil {
			slog.Error("encoding kill switch response", "err", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	// Not enough attempts yet
	attemptsRemaining := 3 - count
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status":             "attempt recorded",
		"attempts":           count,
		"attempts_remaining": attemptsRemaining,
		"message":            fmt.Sprintf("Need %d more attempts within 1 minute to kill scheduler", attemptsRemaining),
	}); err != nil {
		slog.Error("encoding kill switch response", "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

// handleRestart handles the restart endpoint
func (s *APIServer) handleRestart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get the API key from query parameter
	apiKey := r.URL.Query().Get("key")
	if apiKey == "" {
		http.Error(w, "Missing API key", http.StatusUnauthorized)
		return
	}

	// Check if the API key matches the stored restart key
	storedKey, err := s.service.db.GetCredential("kill_restart_api_key")
	if err != nil {
		slog.Error("error retrieving restart API key", "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(storedKey), []byte(apiKey))
	if err != nil {
		slog.Error("error generating hashed restart API key", "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Record the restart attempt
	if err := s.service.db.RecordKillSwitchAttempt("restart"); err != nil {
		slog.Error("recording restart attempt", "err", err)
	}

	// Clean up old attempts (older than 5 minutes)
	if err := s.service.db.CleanupOldKillSwitchAttempts(5 * time.Minute); err != nil {
		slog.Error("cleaning up old restart attempts", "err", err)
	}

	// Check if we have 3 attempts in the last minute
	count, err := s.service.db.GetRecentKillSwitchAttempts("restart", time.Minute)
	if err != nil {
		slog.Error("checking recent restart attempts", "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if count >= 3 {
		// Restart the scheduler
		if err := s.service.db.SetSchedulerStatus(true); err != nil {
			slog.Error("setting scheduler status", "err", err)
			http.Error(w, "failed to restart scheduler", http.StatusInternalServerError)
			return
		}

		slog.Info("scheduler restarted via restart endpoint")
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{
			"status":  "restarting scheduler",
			"message": "Scheduler has been restarted",
		}); err != nil {
			slog.Error("encoding restart response", "err", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	// Not enough attempts yet
	attemptsRemaining := 3 - count
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status":             "attempt recorded",
		"attempts":           count,
		"attempts_remaining": attemptsRemaining,
		"message":            fmt.Sprintf("Need %d more attempts within 1 minute to restart scheduler", attemptsRemaining),
	}); err != nil {
		slog.Error("encoding restart response", "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func (s *APIServer) handleOverride(w http.ResponseWriter, r *http.Request) {
	defer s.metricsUpdater.Trigger()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	certId := r.URL.Query().Get("cert_id")
	if certId == "" {
		http.Error(w, "missing certificate ID", http.StatusBadRequest)
		return
	}

	// ensure the certId is numeric
	certIdInt, err := strconv.ParseInt(certId, 10, 64)
	if err != nil {
		http.Error(w, "invalid certificate ID", http.StatusBadRequest)
		return
	}

	key := r.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "missing API key", http.StatusUnauthorized)
		return
	}

	storedKey, err := s.service.db.GetCredential("certificate_override_key")
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(storedKey), []byte(key))
	if err != nil {
		http.Error(w, "Invalid API key", http.StatusUnauthorized)
		return
	}

	err = s.service.db.MarkCertificateOverrideSent(certIdInt)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{
		"status": "override sent",
	}); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
}
