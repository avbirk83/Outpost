package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/outpost/outpost/internal/auth"
	"github.com/outpost/outpost/internal/database"
)

// Auth handlers

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	session, user, err := s.auth.Login(req.Username, req.Password)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    session.Token,
		Path:     "/",
		HttpOnly: true,
		Expires:  session.ExpiresAt,
		SameSite: http.SameSiteLaxMode,
	})

	json.NewEncoder(w).Encode(map[string]interface{}{
		"token": session.Token,
		"user": map[string]interface{}{
			"id":       user.ID,
			"username": user.Username,
			"role":     user.Role,
		},
	})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token := s.getSessionToken(r)
	if token != "" {
		s.auth.Logout(token)
	}

	// Clear cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	json.NewEncoder(w).Encode(map[string]string{"status": "logged out"})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token := s.getSessionToken(r)
	if token == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := s.auth.ValidateSession(token)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check if user has PIN elevation
	isElevated := false
	elevationToken := s.getElevationToken(r)
	if elevationToken != "" {
		elevation, err := s.db.GetPinElevationByToken(elevationToken)
		if err == nil && elevation.UserID == user.ID {
			isElevated = true
		}
	}

	response := map[string]interface{}{
		"id":                 user.ID,
		"username":           user.Username,
		"role":               user.Role,
		"contentRatingLimit": user.ContentRatingLimit,
		"requirePin":         user.RequirePin,
		"isElevated":         isElevated,
		"hasPin":             user.PinHash != nil && *user.PinHash != "",
	}

	json.NewEncoder(w).Encode(response)
}

// getElevationToken extracts the elevation token from request headers or cookies
func (s *Server) getElevationToken(r *http.Request) string {
	// Check header first
	token := r.Header.Get("X-Elevation-Token")
	if token != "" {
		return token
	}

	// Check cookie
	cookie, err := r.Cookie("elevation_token")
	if err == nil {
		return cookie.Value
	}

	return ""
}

func (s *Server) handleVerifyPin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := s.getCurrentUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Pin string `json:"pin"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Pin == "" || len(req.Pin) != 4 {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"valid": false,
			"error": "PIN must be 4 digits",
		})
		return
	}

	// Check if user has a PIN set
	if user.PinHash == nil || *user.PinHash == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"valid": false,
			"error": "No PIN set for this user",
		})
		return
	}

	// Verify PIN
	if !auth.CheckPassword(req.Pin, *user.PinHash) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"valid": false,
			"error": "Incorrect PIN",
		})
		return
	}

	// Create elevation token (valid for 1 hour)
	elevationToken, err := auth.GenerateToken()
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}
	elevation := &database.PinElevation{
		UserID:    user.ID,
		Token:     elevationToken,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	if err := s.db.CreatePinElevation(elevation); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Set elevation token as cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "elevation_token",
		Value:    elevationToken,
		Path:     "/",
		Expires:  elevation.ExpiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	json.NewEncoder(w).Encode(map[string]interface{}{
		"valid": true,
		"token": elevationToken,
	})
}

func (s *Server) handleSetup(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Check if any users exist
	count, err := s.db.CountUsers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.Method == http.MethodGet {
		// Return setup status
		json.NewEncoder(w).Encode(map[string]interface{}{
			"setupRequired": count == 0,
		})
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Only allow setup if no users exist
	if count > 0 {
		http.Error(w, "Setup already completed", http.StatusForbidden)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		http.Error(w, "Username and password required", http.StatusBadRequest)
		return
	}

	// Create admin user
	user, err := s.auth.CreateUser(req.Username, req.Password, "admin")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":       user.ID,
		"username": user.Username,
		"role":     user.Role,
	})
}

// Setup wizard handlers

func (s *Server) handleSetupStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Check setup_completed setting first
	setupCompleted, _ := s.db.GetSetting("setup_completed")
	if setupCompleted == "true" {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"needsSetup":     false,
			"setupCompleted": true,
		})
		return
	}

	// Check each step
	steps := map[string]bool{
		"adminCreated":             false,
		"libraryAdded":             false,
		"downloadClientConfigured": false,
		"indexerConfigured":        false,
		"qualityProfileSet":        false,
	}

	// 1. Check if admin exists
	userCount, _ := s.db.CountUsers()
	steps["adminCreated"] = userCount > 0

	// 2. Check if any library exists
	libraries, _ := s.db.GetLibraries()
	steps["libraryAdded"] = len(libraries) > 0

	// 3. Check if download client is configured
	downloadClients, _ := s.db.GetDownloadClients()
	for _, dc := range downloadClients {
		if dc.Enabled {
			steps["downloadClientConfigured"] = true
			break
		}
	}

	// 4. Check if indexers are configured (direct or via Prowlarr)
	enabledIndexers, _ := s.db.GetEnabledIndexers()
	prowlarrConfig, _ := s.db.GetProwlarrConfig()
	steps["indexerConfigured"] = len(enabledIndexers) > 0 || (prowlarrConfig != nil && prowlarrConfig.URL != "")

	// 5. Check if quality presets exist
	presets, _ := s.db.GetQualityPresets()
	for _, p := range presets {
		if p.Enabled {
			steps["qualityProfileSet"] = true
			break
		}
	}

	// Determine if setup is needed (admin must exist, but other steps can be skipped)
	needsSetup := !steps["adminCreated"] || !steps["libraryAdded"]

	// All steps except admin can be skipped
	canSkip := steps["adminCreated"]

	json.NewEncoder(w).Encode(map[string]interface{}{
		"needsSetup":     needsSetup,
		"setupCompleted": false,
		"steps":          steps,
		"canSkip":        canSkip,
	})
}

func (s *Server) handleSetupComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Set setup_completed flag
	if err := s.db.SetSetting("setup_completed", "true"); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// User management handlers

func (s *Server) handleUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		users, err := s.db.GetUsers()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if users == nil {
			users = []database.User{}
		}
		json.NewEncoder(w).Encode(users)

	case http.MethodPost:
		var req struct {
			Username           string  `json:"username"`
			Password           string  `json:"password"`
			Role               string  `json:"role"`
			ContentRatingLimit *string `json:"contentRatingLimit"`
			RequirePin         bool    `json:"requirePin"`
			Pin                string  `json:"pin"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Username == "" || req.Password == "" {
			http.Error(w, "Username and password required", http.StatusBadRequest)
			return
		}

		if req.Role == "" {
			req.Role = "user"
		}

		// For kid role, default to PG if no limit set
		if req.Role == "kid" && req.ContentRatingLimit == nil {
			pg := "PG"
			req.ContentRatingLimit = &pg
		}

		user, err := s.auth.CreateUser(req.Username, req.Password, req.Role)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Set parental controls
		user.ContentRatingLimit = req.ContentRatingLimit
		user.RequirePin = req.RequirePin
		if err := s.db.UpdateUser(user); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Set PIN if provided
		if req.Pin != "" && len(req.Pin) == 4 {
			pinHash, err := auth.HashPassword(req.Pin)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if err := s.db.UpdateUserPin(user.ID, &pinHash); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(user)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Parse path: /api/users/{id}
	path := strings.TrimPrefix(r.URL.Path, "/api/users/")
	id, err := strconv.ParseInt(path, 10, 64)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		user, err := s.db.GetUserByID(id)
		if err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(user)

	case http.MethodPut:
		var req struct {
			Username           string  `json:"username"`
			Password           string  `json:"password"`
			Role               string  `json:"role"`
			ContentRatingLimit *string `json:"contentRatingLimit"`
			RequirePin         *bool   `json:"requirePin"`
			Pin                string  `json:"pin"`
			ClearPin           bool    `json:"clearPin"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		user, err := s.db.GetUserByID(id)
		if err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		if req.Username != "" {
			user.Username = req.Username
		}
		if req.Role != "" {
			user.Role = req.Role
			// For kid role, default to PG if no limit set
			if req.Role == "kid" && user.ContentRatingLimit == nil && req.ContentRatingLimit == nil {
				pg := "PG"
				user.ContentRatingLimit = &pg
			}
		}

		// Handle content rating limit - allow setting to nil to remove limit
		// The request sends the field, so we update it
		user.ContentRatingLimit = req.ContentRatingLimit

		if req.RequirePin != nil {
			user.RequirePin = *req.RequirePin
		}

		if err := s.db.UpdateUser(user); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Update password if provided
		if req.Password != "" {
			hash, err := auth.HashPassword(req.Password)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if err := s.db.UpdateUserPassword(id, hash); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		// Update or clear PIN
		if req.ClearPin {
			if err := s.db.UpdateUserPin(id, nil); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else if req.Pin != "" && len(req.Pin) == 4 {
			pinHash, err := auth.HashPassword(req.Pin)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if err := s.db.UpdateUserPin(user.ID, &pinHash); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		json.NewEncoder(w).Encode(user)

	case http.MethodDelete:
		// Don't allow deleting yourself
		currentUser := s.getCurrentUser(r)
		if currentUser != nil && currentUser.ID == id {
			http.Error(w, "Cannot delete yourself", http.StatusBadRequest)
			return
		}

		// Delete user sessions first
		s.db.DeleteUserSessions(id)

		if err := s.db.DeleteUser(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Profile handlers

func (s *Server) handleProfiles(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	user := s.getCurrentUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case http.MethodGet:
		profiles, err := s.db.GetProfilesByUser(user.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if profiles == nil {
			profiles = []database.Profile{}
		}
		json.NewEncoder(w).Encode(profiles)

	case http.MethodPost:
		var req struct {
			Name               string  `json:"name"`
			AvatarURL          *string `json:"avatarUrl"`
			IsKid              bool    `json:"isKid"`
			ContentRatingLimit *string `json:"contentRatingLimit"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Name == "" {
			http.Error(w, "Profile name is required", http.StatusBadRequest)
			return
		}

		// Check profile limit (max 5 per user)
		count, err := s.db.CountProfilesByUser(user.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if count >= 5 {
			http.Error(w, "Maximum 5 profiles per user", http.StatusBadRequest)
			return
		}

		profile := &database.Profile{
			UserID:             user.ID,
			Name:               req.Name,
			AvatarURL:          req.AvatarURL,
			IsKid:              req.IsKid,
			ContentRatingLimit: req.ContentRatingLimit,
		}

		if err := s.db.CreateProfile(profile); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(profile)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleProfile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	user := s.getCurrentUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse path: /api/profiles/{id} or /api/profiles/{id}/select
	path := strings.TrimPrefix(r.URL.Path, "/api/profiles/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Profile ID required", http.StatusBadRequest)
		return
	}

	// Check for special "active" endpoint
	if parts[0] == "active" {
		s.handleActiveProfile(w, r, user)
		return
	}

	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		http.Error(w, "Invalid profile ID", http.StatusBadRequest)
		return
	}

	// Check if this is a select action
	if len(parts) >= 2 && parts[1] == "select" {
		s.handleProfileSelect(w, r, user, id)
		return
	}

	// Get profile and verify ownership
	profile, err := s.db.GetProfile(id)
	if err != nil {
		http.Error(w, "Profile not found", http.StatusNotFound)
		return
	}

	if profile.UserID != user.ID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	switch r.Method {
	case http.MethodGet:
		json.NewEncoder(w).Encode(profile)

	case http.MethodPut:
		var req struct {
			Name               string  `json:"name"`
			AvatarURL          *string `json:"avatarUrl"`
			IsKid              *bool   `json:"isKid"`
			ContentRatingLimit *string `json:"contentRatingLimit"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Name != "" {
			profile.Name = req.Name
		}
		if req.AvatarURL != nil {
			profile.AvatarURL = req.AvatarURL
		}
		if req.IsKid != nil {
			profile.IsKid = *req.IsKid
		}
		profile.ContentRatingLimit = req.ContentRatingLimit

		if err := s.db.UpdateProfile(profile); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(profile)

	case http.MethodDelete:
		// Don't allow deleting the only profile
		count, err := s.db.CountProfilesByUser(user.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if count <= 1 {
			http.Error(w, "Cannot delete the only profile", http.StatusBadRequest)
			return
		}

		// Don't allow deleting default profile
		if profile.IsDefault {
			http.Error(w, "Cannot delete default profile", http.StatusBadRequest)
			return
		}

		if err := s.db.DeleteProfile(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleProfileSelect(w http.ResponseWriter, r *http.Request, user *database.User, profileID int64) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Verify profile belongs to user
	profile, err := s.db.GetProfile(profileID)
	if err != nil {
		http.Error(w, "Profile not found", http.StatusNotFound)
		return
	}

	if profile.UserID != user.ID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Get session token and update active profile
	token := s.getSessionToken(r)
	if token == "" {
		http.Error(w, "No session", http.StatusBadRequest)
		return
	}

	if err := s.db.SetActiveProfile(token, profileID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(profile)
}

func (s *Server) handleActiveProfile(w http.ResponseWriter, r *http.Request, user *database.User) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	profileID := s.getActiveProfileID(r)
	if profileID == nil {
		// No active profile, return null
		w.Write([]byte("null"))
		return
	}

	profile, err := s.db.GetProfile(*profileID)
	if err != nil {
		w.Write([]byte("null"))
		return
	}

	// Verify profile still belongs to user
	if profile.UserID != user.ID {
		w.Write([]byte("null"))
		return
	}

	json.NewEncoder(w).Encode(profile)
}
