// Copyright 2020 ChainSafe Systems
// SPDX-License-Identifier: LGPL-3.0-only

package health

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/ChainSafe/chainbridge-utils/core"
	"github.com/ChainSafe/chainbridge-utils/msg"
	log "github.com/ChainSafe/log15"
)

type httpMetricServer struct {
	port         int
	blockTimeout int // After this duration (seconds) with no change in block height a chain will be considered unhealthy
	chains       map[string]core.Chain
	stats        map[string]*ChainInfo
}

type httpResponse struct {
	Data ChainInfo `json:"data,omitempty"`
	Error  string      `json:"error,omitempty"`
}

type ChainInfo struct {
	ChainId     msg.ChainId `json:"chainId"`
	Height      *big.Int    `json:"height"`
	LastUpdated time.Time   `json:"lastUpdated"`
}

func NewHealthServer(port int, chains []core.Chain, blockTimeout int) *httpMetricServer {
	chainMap := make(map[string]core.Chain)
	for _, c := range chains {
		chainMap[c.Name()] = c
	}

	return &httpMetricServer{
		port:         port,
		chains:       chainMap,
		blockTimeout: blockTimeout,
		stats:        make(map[string]*ChainInfo),
	}
}

// healthStatus is a catch-all update that grabs the latest updates on the running chains
// It assumes that the configuration was set correctly, therefore the relevant chains are
// only those that are in the core.Core registry.
func (s httpMetricServer) HealthStatus(w http.ResponseWriter, r *http.Request) {
	tokens := strings.Split(r.URL.String(), "/")
	// TODO: What if len(tokens) < 1
	chainName := tokens[len(tokens) - 1]
	chain, ok := s.chains[chainName]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	current := chain.LatestBlock()
	prev := s.stats[chainName]
	if s.stats[chainName].Height == nil {
		// First time we've received a block for this chain
		s.stats[chainName] = &ChainInfo{
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
			response := &httpResponse{
				Error:  fmt.Sprintf("chain %d height hasn't changed for %f seconds. Current Height: %s", prev.ChainId, timeDiff.Seconds(), current.Height),
			}
			w.WriteHeader(http.StatusInternalServerError)
			err := json.NewEncoder(w).Encode(response)
			if err != nil {
				log.Error("Failed to write metrics", "err", err)
			}
			return
		} else if current.Height != nil && prev.Height != nil && current.Height.Cmp(prev.Height) == -1 { // Error for having a smaller blockheight than previous
			response := &httpResponse{
				Error:  fmt.Sprintf("unexpected block height. previous = %s current = %s", prev.Height, current.Height),
			}
			w.WriteHeader(http.StatusInternalServerError)
			err := json.NewEncoder(w).Encode(response)
			if err != nil {
				log.Error("Failed to write metrics", "err", err)
			}
			return
		}
	}

	response := &httpResponse{
		Data: *s.stats[chainName],
	}
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		log.Error("Failed to serve metrics")
	}
}
