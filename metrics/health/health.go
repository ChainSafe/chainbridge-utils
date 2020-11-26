// Copyright 2020 ChainSafe Systems
// SPDX-License-Identifier: LGPL-3.0-only

package health

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"path"
	"time"

	"github.com/ChainSafe/chainbridge-utils/core"
	"github.com/ChainSafe/chainbridge-utils/msg"
	log "github.com/ChainSafe/log15"
)

type httpMetricServer struct {
	port         int
	blockTimeout int // After this duration (seconds) with no change in block height a chain will be considered unhealthy
	chains       map[string]core.Chain
	stats        map[string]*ChainStats
}

type ChainStats struct {
	ChainId     msg.ChainId `json:"chainId"`
	Height      *big.Int    `json:"height"`
	LastUpdated time.Time   `json:"lastUpdated"`
}

// NewHealthServer creates a new server with unique endpoints for each chain based on chain name.
func NewHealthServer(port int, chains []core.Chain, blockTimeout int) *httpMetricServer {
	chainMap := make(map[string]core.Chain)
	for _, c := range chains {
		chainMap[c.Name()] = c
	}

	return &httpMetricServer{
		port:         port,
		chains:       chainMap,
		blockTimeout: blockTimeout,
		stats:        make(map[string]*ChainStats),
	}
}

// healthStatus is a catch-all that grabs the latest update for a given chain.
// The last segment of the URL is used to identify the chain (eg. "health/goerli" will return "goerli").
func (s httpMetricServer) HealthStatus(w http.ResponseWriter, r *http.Request) {
	// Get last segment of URL
	chainName := path.Base(r.URL.String())
	chain, ok := s.chains[chainName]
	if !ok {
		http.Error(w, "Invalid chain name", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	current := chain.LatestBlock()
	prev := s.stats[chainName]
	if s.stats[chainName] == nil {
		// First time we've received a block for this chain
		s.stats[chainName] = &ChainStats{
			ChainId:     chain.Id(),
			Height:      current.Height,
			LastUpdated: current.LastUpdated,
		}
	} else {
		now := time.Now()
		timeDiff := now.Sub(prev.LastUpdated)
		// If block has changed, update it
		if current.Height.Cmp(prev.Height) == 1 {
			s.stats[chainName].LastUpdated = current.LastUpdated
			s.stats[chainName].Height = current.Height
		} else if int(timeDiff.Seconds()) >= s.blockTimeout { // Error if we exceeded the time limit
			http.Error(w, fmt.Sprintf("chain %d height hasn't changed for %f seconds. Current Height: %s", prev.ChainId, timeDiff.Seconds(), current.Height), http.StatusInternalServerError)
			return
		} else if current.Height != nil && prev.Height != nil && current.Height.Cmp(prev.Height) == -1 { // Error for having a smaller blockheight than previous
			http.Error(w, fmt.Sprintf("unexpected block height. previous = %s current = %s", prev.Height, current.Height), http.StatusInternalServerError)
			return
		}
	}

	err := json.NewEncoder(w).Encode(s.stats[chainName])
	if err != nil {
		log.Error("Failed to serve metrics")
	}
}
