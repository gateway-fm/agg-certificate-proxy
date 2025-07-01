package certificate

import (
	"encoding/json"
	"fmt"
	"html/template"
	"math/big"
	"net/http"
	"strconv"
	"time"
	"log/slog"
)

// APIServer handles HTTP requests.
type APIServer struct {
	service *Service
}

// NewAPIServer creates a new API server.
func NewAPIServer(service *Service) *APIServer {
	return &APIServer{service: service}
}

// RegisterHandlers registers the HTTP handlers.
func (s *APIServer) RegisterHandlers() {
	http.HandleFunc("/", s.viewCerts)
	http.HandleFunc("/config", s.viewConfig)
	http.HandleFunc("/kill", s.handleKillSwitch)
	http.HandleFunc("/restart", s.handleRestart)
}

const tpl = `
<!DOCTYPE html>
<html>
<head>
    <title>Certificates</title>
    <style>
        body { font-family: sans-serif; margin: 20px; }
        table { border-collapse: collapse; width: 100%; margin-top: 20px; table-layout: fixed; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; vertical-align: top; }
        th { background-color: #f2f2f2; }
        .pending { background-color: #fff3cd; }
        .processed { background-color: #d4edda; }
        .metadata { 
            font-size: 0.75em; 
            max-width: 300px;
            overflow: hidden;
            position: relative;
        }
        .metadata pre {
            margin: 0;
            white-space: pre-wrap;
            word-wrap: break-word;
            max-height: 150px;
            overflow-y: auto;
            background-color: #f5f5f5;
            padding: 5px;
            border-radius: 3px;
            font-family: 'Courier New', monospace;
        }
        .config { margin-bottom: 20px; padding: 10px; background-color: #e9ecef; border-radius: 5px; }
        
        /* Scheduler status */
        .scheduler-status {
            margin-bottom: 20px;
            padding: 10px;
            border-radius: 5px;
            border: 1px solid;
        }
        .scheduler-active {
            background-color: #d4edda;
            border-color: #c3e6cb;
            color: #155724;
        }
        .scheduler-stopped {
            background-color: #f8d7da;
            border-color: #f5c6cb;
            color: #721c24;
        }
        
        /* Held values section */
        .held-values {
            margin-bottom: 20px;
            padding: 10px;
            background-color: #f8f9fa;
            border-radius: 5px;
            border: 1px solid #dee2e6;
        }
        .held-values h3 {
            margin-top: 0;
            margin-bottom: 10px;
            color: #495057;
        }
        .chain-totals {
            width: auto;
            min-width: 400px;
            margin: 0;
        }
        .chain-totals th {
            background-color: #6c757d;
            color: white;
        }
        .chain-totals td {
            background-color: white;
        }
        
        /* Column widths */
        th:nth-child(1), td:nth-child(1) { width: 5%; }  /* ID */
        th:nth-child(2), td:nth-child(2) { width: 8%; }  /* Network ID */
        th:nth-child(3), td:nth-child(3) { width: 8%; }  /* Height */
        th:nth-child(4), td:nth-child(4) { width: 12%; } /* Received At */
        th:nth-child(5), td:nth-child(5) { width: 12%; } /* Will Send At */
        th:nth-child(6), td:nth-child(6) { width: 15%; } /* Status */
        th:nth-child(7), td:nth-child(7) { width: 10%; } /* Total Amount */
        th:nth-child(8), td:nth-child(8) { width: 10%; } /* Exit Count */
        th:nth-child(9), td:nth-child(9) { width: 20%; } /* Metadata */
        
        /* Expandable metadata */
        .metadata-toggle {
            cursor: pointer;
            color: #007bff;
            text-decoration: underline;
            font-size: 0.9em;
        }
        .metadata-full {
            display: none;
            margin-top: 10px;
        }
        .metadata-full.show {
            display: block;
        }
    </style>
    <script>
        function toggleMetadata(id) {
            var element = document.getElementById('metadata-' + id);
            if (element.classList.contains('show')) {
                element.classList.remove('show');
            } else {
                element.classList.add('show');
            }
        }
    </script>
</head>
<body>
    <h1>Agg Certificate Proxy</h1>
    <div class="config">
        <strong>Current Configuration:</strong><br>
        Delay: {{.Config.Delay}}<br>
        Current Time: {{.Config.CurrentTime}}
    </div>
    
    <div class="scheduler-status {{if .SchedulerActive}}scheduler-active{{else}}scheduler-stopped{{end}}">
        <strong>Scheduler Status:</strong> {{if .SchedulerActive}}Active (Processing Certificates){{else}}STOPPED (Kill Switch Activated){{end}}
    </div>
    
    <div class="held-values">
        <h3>Held Certificates Total Value by Chain</h3>
        <table class="chain-totals">
            <tr>
                <th>Chain ID</th>
                <th>Total Held Amount</th>
                <th>Certificate Count</th>
            </tr>
            {{range $chainID, $info := .ChainTotals}}
            <tr>
                <td>{{$chainID}}</td>
                <td>{{$info.TotalAmount}}</td>
                <td>{{$info.CertCount}}</td>
            </tr>
            {{else}}
            <tr>
                <td colspan="3" style="text-align: center; font-style: italic;">No pending certificates</td>
            </tr>
            {{end}}
        </table>
    </div>
    
    <h2>Certificates</h2>
    <table>
        <tr>
            <th>ID</th>
            <th>Network ID</th>
            <th>Height</th>
            <th>Received At</th>
            <th>Will Send At</th>
            <th>Status</th>
            <th>Total Amount</th>
            <th>Exit Count</th>
            <th>Metadata</th>
        </tr>
        {{range .Certificates}}
        <tr class="{{if .ProcessedAt.Valid}}processed{{else}}pending{{end}}">
            <td>{{.ID}}</td>
            <td>{{.NetworkID}}</td>
            <td>{{.Height}}</td>
            <td>{{.ReceivedAt.Format "2006-01-02 15:04:05"}}</td>
            <td>{{.WillSendAt.Format "2006-01-02 15:04:05"}}</td>
            <td>{{if .ProcessedAt.Valid}}Processed at {{.ProcessedAt.Time.Format "2006-01-02 15:04:05"}}{{else}}Pending{{end}}</td>
            <td>{{.TotalAmount}}</td>
            <td>BE: {{.BridgeExitCount}}, IBE: {{.ImportedBridgeExitCount}}</td>
            <td class="metadata">
                <pre>{{.PrettyMetadata}}</pre>
                {{if .HasFullMetadata}}
                <span class="metadata-toggle" onclick="toggleMetadata({{.ID}})">Show full metadata</span>
                <div id="metadata-{{.ID}}" class="metadata-full">
                    <pre>{{.FullPrettyMetadata}}</pre>
                </div>
                {{end}}
            </td>
        </tr>
        {{end}}
    </table>
</body>
</html>
`

// CertificateView extends Certificate with calculated fields for display
type CertificateView struct {
	Certificate
	NetworkID               uint32
	Height                  uint64
	WillSendAt              time.Time
	TotalAmount             string
	BridgeExitCount         int
	ImportedBridgeExitCount int
	PrettyMetadata          string
	FullPrettyMetadata      string
	HasFullMetadata         bool
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
func formatAmount(weiStr string) string {
	wei, ok := new(big.Int).SetString(weiStr, 10)
	if !ok || wei.Sign() == 0 {
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

func (s *APIServer) viewCerts(w http.ResponseWriter, r *http.Request) {
	certs, err := s.service.GetCertificates()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get certificates: %v", err), http.StatusInternalServerError)
		return
	}

	// Get scheduler status
	schedulerActive, err := s.service.db.GetSchedulerStatus()
	if err != nil {
		slog.Error("failed to get scheduler status", "err", err)
		schedulerActive = true // Default to active if error
	}

	// Get delay configuration
	delayStr, err := s.service.GetConfigValue("delay_seconds")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		if _, err := w.Write([]byte("failed to get delay_seconds config value")); err != nil {
			slog.Error("failed to write error message to response", "err", err)
			return
		}
		return
	}
	delaySeconds, _ := strconv.Atoi(delayStr)
	delayDuration := time.Duration(delaySeconds) * time.Second

	// Track chain totals for pending certificates
	type ChainInfo struct {
		TotalAmount string
		CertCount   int
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
				totalAmount := uint64(0)

				// Count and sum bridge exits
				if bridgeExits, ok := meta["bridge_exits"].([]interface{}); ok {
					view.BridgeExitCount = len(bridgeExits)
					for _, be := range bridgeExits {
						if beMap, ok := be.(map[string]interface{}); ok {
							if amountStr, ok := beMap["amount"].(string); ok {
								if amount, err := strconv.ParseUint(amountStr, 10, 64); err == nil {
									totalAmount += amount
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
							if amountStr, ok := ibeMap["amount"].(string); ok {
								if amount, err := strconv.ParseUint(amountStr, 10, 64); err == nil {
									totalAmount += amount
								}
							}
						}
					}
				}
				if ibeCount, ok := meta["imported_bridge_exits_count"].(float64); ok {
					view.ImportedBridgeExitCount = int(ibeCount)
				}

				view.TotalAmount = formatAmount(fmt.Sprintf("%d", totalAmount))

				// Create truncated version for display
				truncated, wasTruncated := truncateJSON(meta, 8)
				view.PrettyMetadata = truncated
				view.HasFullMetadata = wasTruncated || len(view.FullPrettyMetadata) > 500

				// Add to chain totals if this is a pending certificate
				if !cert.ProcessedAt.Valid && view.NetworkID > 0 {
					if chainTotals[view.NetworkID] == nil {
						chainTotals[view.NetworkID] = &ChainInfo{TotalAmount: "0", CertCount: 0}
					}
					chainInfo := chainTotals[view.NetworkID]
					chainInfo.CertCount++

					// Parse and add to total
					currentTotal, _ := strconv.ParseUint(chainInfo.TotalAmount, 10, 64)
					currentTotal += totalAmount
					chainInfo.TotalAmount = fmt.Sprintf("%d", currentTotal)
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
		info.TotalAmount = formatAmount(info.TotalAmount)
	}

	// Prepare template data
	data := struct {
		Config struct {
			DelayHours  string
			Delay       string
			CurrentTime string
		}
		SchedulerActive bool
		ChainTotals     map[uint32]*ChainInfo
		Certificates    []CertificateView
	}{
		Config: struct {
			DelayHours  string
			Delay       string
			CurrentTime string
		}{
			DelayHours:  delayStr, // Keep for backward compatibility in template
			Delay:       formatDuration(delaySeconds),
			CurrentTime: time.Now().Format("2006-01-02 15:04:05"),
		},
		SchedulerActive: schedulerActive,
		ChainTotals:     chainTotals,
		Certificates:    certViews,
	}

	t, err := template.New("webpage").Parse(tpl)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to parse template: %v", err), http.StatusInternalServerError)
		return
	}

	err = t.Execute(w, data)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to execute template: %v", err), http.StatusInternalServerError)
	}
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

// Start starts the HTTP server.
func (s *APIServer) Start(addr string) error {
	slog.Info("http server listening", "address", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		return err
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

	if storedKey == "" || apiKey != storedKey {
		http.Error(w, "invalid API key", http.StatusUnauthorized)
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

	if storedKey == "" || apiKey != storedKey {
		http.Error(w, "invalid API key", http.StatusUnauthorized)
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
