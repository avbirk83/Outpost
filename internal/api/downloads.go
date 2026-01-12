package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/outpost/outpost/internal/database"
	"github.com/outpost/outpost/internal/downloadclient"
	"github.com/outpost/outpost/internal/indexer"
)

// Download client handlers

func (s *Server) handleDownloadClients(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		clients, err := s.db.GetDownloadClients()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if clients == nil {
			clients = []database.DownloadClient{}
		}
		// Don't expose passwords in responses
		for i := range clients {
			clients[i].Password = ""
		}
		json.NewEncoder(w).Encode(clients)

	case http.MethodPost:
		var client database.DownloadClient
		if err := json.NewDecoder(r.Body).Decode(&client); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if client.Name == "" || client.Type == "" || client.Host == "" || client.Port == 0 {
			http.Error(w, "Name, type, host, and port are required", http.StatusBadRequest)
			return
		}

		// Validate client type
		validTypes := map[string]bool{"qbittorrent": true, "transmission": true, "sabnzbd": true, "nzbget": true}
		if !validTypes[client.Type] {
			http.Error(w, "Invalid client type", http.StatusBadRequest)
			return
		}

		client.Enabled = true
		if err := s.db.CreateDownloadClient(&client); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		client.Password = "" // Don't expose password
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(client)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleDownloadClient(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Parse path: /api/download-clients/{id} or /api/download-clients/{id}/test
	path := strings.TrimPrefix(r.URL.Path, "/api/download-clients/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Client ID required", http.StatusBadRequest)
		return
	}

	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		http.Error(w, "Invalid client ID", http.StatusBadRequest)
		return
	}

	// Handle test endpoint
	if len(parts) == 2 && parts[1] == "test" {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := s.downloads.TestClient(id); err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   err.Error(),
			})
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Connection successful",
		})
		return
	}

	switch r.Method {
	case http.MethodGet:
		client, err := s.db.GetDownloadClient(id)
		if err != nil {
			http.Error(w, "Client not found", http.StatusNotFound)
			return
		}
		client.Password = ""
		json.NewEncoder(w).Encode(client)

	case http.MethodPut:
		var req database.DownloadClient
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		client, err := s.db.GetDownloadClient(id)
		if err != nil {
			http.Error(w, "Client not found", http.StatusNotFound)
			return
		}

		// Update fields
		if req.Name != "" {
			client.Name = req.Name
		}
		if req.Type != "" {
			client.Type = req.Type
		}
		if req.Host != "" {
			client.Host = req.Host
		}
		if req.Port != 0 {
			client.Port = req.Port
		}
		if req.Username != "" {
			client.Username = req.Username
		}
		if req.Password != "" {
			client.Password = req.Password
		}
		if req.APIKey != "" {
			client.APIKey = req.APIKey
		}
		client.UseTLS = req.UseTLS
		client.Category = req.Category
		client.Priority = req.Priority
		client.Enabled = req.Enabled

		if err := s.db.UpdateDownloadClient(client); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		client.Password = ""
		json.NewEncoder(w).Encode(client)

	case http.MethodDelete:
		if err := s.db.DeleteDownloadClient(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleDownloads(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	downloads, err := s.downloads.GetAllDownloads()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if downloads == nil {
		downloads = []downloadclient.Download{}
	}

	json.NewEncoder(w).Encode(downloads)
}

// Indexer handlers

func (s *Server) handleIndexers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		indexers, err := s.db.GetIndexers()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if indexers == nil {
			indexers = []database.Indexer{}
		}
		// Don't expose API keys in responses
		for i := range indexers {
			indexers[i].APIKey = ""
		}
		json.NewEncoder(w).Encode(indexers)

	case http.MethodPost:
		var idx database.Indexer
		if err := json.NewDecoder(r.Body).Decode(&idx); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if idx.Name == "" || idx.Type == "" || idx.URL == "" {
			http.Error(w, "Name, type, and URL are required", http.StatusBadRequest)
			return
		}

		// Validate indexer type
		validTypes := map[string]bool{"torznab": true, "newznab": true, "prowlarr": true}
		if !validTypes[idx.Type] {
			http.Error(w, "Invalid indexer type", http.StatusBadRequest)
			return
		}

		idx.Enabled = true
		if err := s.db.CreateIndexer(&idx); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Add to manager
		config := &indexer.IndexerConfig{
			ID:         idx.ID,
			Name:       idx.Name,
			Type:       idx.Type,
			URL:        idx.URL,
			APIKey:     idx.APIKey,
			Categories: idx.Categories,
			Priority:   idx.Priority,
			Enabled:    idx.Enabled,
		}
		if err := s.indexers.AddIndexer(config); err != nil {
			// Indexer created but connection failed - still return success
			idx.APIKey = ""
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(idx)
			return
		}

		idx.APIKey = ""
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(idx)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleIndexer(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Parse path: /api/indexers/{id} or /api/indexers/{id}/test
	path := strings.TrimPrefix(r.URL.Path, "/api/indexers/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Indexer ID required", http.StatusBadRequest)
		return
	}

	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		http.Error(w, "Invalid indexer ID", http.StatusBadRequest)
		return
	}

	// Handle test endpoint
	if len(parts) == 2 && parts[1] == "test" {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := s.indexers.TestIndexer(id); err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   err.Error(),
			})
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Connection successful",
		})
		return
	}

	// Handle capabilities endpoint
	if len(parts) == 2 && parts[1] == "capabilities" {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		caps, err := s.indexers.GetCapabilities(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(caps)
		return
	}

	switch r.Method {
	case http.MethodGet:
		idx, err := s.db.GetIndexer(id)
		if err != nil {
			http.Error(w, "Indexer not found", http.StatusNotFound)
			return
		}
		idx.APIKey = ""
		json.NewEncoder(w).Encode(idx)

	case http.MethodPut:
		var req database.Indexer
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		idx, err := s.db.GetIndexer(id)
		if err != nil {
			http.Error(w, "Indexer not found", http.StatusNotFound)
			return
		}

		// Update fields
		if req.Name != "" {
			idx.Name = req.Name
		}
		if req.Type != "" {
			idx.Type = req.Type
		}
		if req.URL != "" {
			idx.URL = req.URL
		}
		if req.APIKey != "" {
			idx.APIKey = req.APIKey
		}
		idx.Categories = req.Categories
		idx.Priority = req.Priority
		idx.Enabled = req.Enabled
		idx.ContentTypes = req.ContentTypes

		if err := s.db.UpdateIndexer(idx); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Update manager
		s.indexers.RemoveIndexer(id)
		if idx.Enabled {
			config := &indexer.IndexerConfig{
				ID:         idx.ID,
				Name:       idx.Name,
				Type:       idx.Type,
				URL:        idx.URL,
				APIKey:     idx.APIKey,
				Categories: idx.Categories,
				Priority:   idx.Priority,
				Enabled:    idx.Enabled,
			}
			s.indexers.AddIndexer(config)
		}

		idx.APIKey = ""
		json.NewEncoder(w).Encode(idx)

	case http.MethodDelete:
		s.indexers.RemoveIndexer(id)
		if err := s.db.DeleteIndexer(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
