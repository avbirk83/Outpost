package api

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/outpost/outpost/internal/database"
	"github.com/outpost/outpost/internal/scanner"
)

// Media handlers

// MovieWithWatchState extends Movie with watch state
type MovieWithWatchState struct {
	database.Movie
	WatchState string  `json:"watchState,omitempty"`
	Progress   float64 `json:"progress,omitempty"`
}

func (s *Server) handleMovies(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	movies, err := s.db.GetMovies()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if movies == nil {
		movies = []database.Movie{}
	}

	// Filter based on user's content rating limit
	user := s.getCurrentUser(r)
	if user != nil && user.ContentRatingLimit != nil {
		var filtered []database.Movie
		for _, m := range movies {
			if s.isContentAllowed(user, m.ContentRating, r) {
				filtered = append(filtered, m)
			}
		}
		movies = filtered
		if movies == nil {
			movies = []database.Movie{}
		}
	}

	// Get watch states
	watchStates, _ := s.db.GetAllMovieWatchStates()

	// Build response with watch states
	result := make([]MovieWithWatchState, len(movies))
	for i, m := range movies {
		result[i] = MovieWithWatchState{Movie: m}
		if state, ok := watchStates[m.ID]; ok {
			result[i].WatchState = state.WatchState
			result[i].Progress = state.Progress
		}
	}

	json.NewEncoder(w).Encode(result)
}

// ShowWithWatchState extends Show with watch state and episode progress
type ShowWithWatchState struct {
	database.Show
	WatchState      string `json:"watchState,omitempty"`
	WatchedEpisodes int    `json:"watchedEpisodes,omitempty"`
	TotalEpisodes   int    `json:"totalEpisodes,omitempty"`
}

func (s *Server) handleShows(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	shows, err := s.db.GetShows()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if shows == nil {
		shows = []database.Show{}
	}

	// Filter based on user's content rating limit
	user := s.getCurrentUser(r)
	if user != nil && user.ContentRatingLimit != nil {
		var filtered []database.Show
		for _, sh := range shows {
			if s.isContentAllowed(user, sh.ContentRating, r) {
				filtered = append(filtered, sh)
			}
		}
		shows = filtered
		if shows == nil {
			shows = []database.Show{}
		}
	}

	// Get watch states
	watchStates, _ := s.db.GetAllShowWatchStates()

	// Build response with watch states
	result := make([]ShowWithWatchState, len(shows))
	for i, sh := range shows {
		result[i] = ShowWithWatchState{Show: sh}
		if state, ok := watchStates[sh.ID]; ok {
			result[i].WatchState = state.WatchState
			result[i].WatchedEpisodes = state.WatchedEpisodes
			result[i].TotalEpisodes = state.TotalEpisodes
		}
	}

	json.NewEncoder(w).Encode(result)
}

func (s *Server) handleShow(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Parse path: /api/shows/{id} or /api/shows/{id}/refresh or /api/shows/{id}/match
	path := strings.TrimPrefix(r.URL.Path, "/api/shows/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Show ID required", http.StatusBadRequest)
		return
	}

	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		http.Error(w, "Invalid show ID", http.StatusBadRequest)
		return
	}

	show, err := s.db.GetShow(id)
	if err != nil {
		http.Error(w, "Show not found", http.StatusNotFound)
		return
	}

	// Check content rating restriction
	user := s.getCurrentUser(r)
	if user != nil && user.ContentRatingLimit != nil && !s.isContentAllowed(user, show.ContentRating, r) {
		// Content is restricted - check if PIN is required
		if user.RequirePin {
			http.Error(w, "Content restricted - PIN required", http.StatusForbidden)
		} else {
			http.Error(w, "Content not available", http.StatusForbidden)
		}
		return
	}

	// Handle refresh endpoint
	if len(parts) == 2 && parts[1] == "refresh" {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if s.metadata != nil {
			if err := s.metadata.FetchShowMetadata(show); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			show, _ = s.db.GetShow(id)
		}
		s.sendShowDetail(w, show)
		return
	}

	// Handle match endpoint
	if len(parts) == 2 && parts[1] == "match" {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			TmdbID int64 `json:"tmdbId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if s.metadata != nil {
			if err := s.metadata.FetchShowMetadataByTmdbID(show, req.TmdbID); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			show, _ = s.db.GetShow(id)
		}
		s.sendShowDetail(w, show)
		return
	}

	// Handle missing episodes endpoint
	if len(parts) == 2 && parts[1] == "missing" {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.handleMissingEpisodes(w, r, show)
		return
	}

	// Handle request-missing endpoint
	if len(parts) == 2 && parts[1] == "request-missing" {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.handleRequestMissingEpisodes(w, r, show)
		return
	}

	// Handle detect-intros endpoint
	if len(parts) >= 2 && parts[1] == "detect-intros" {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		// Check admin permission
		user := r.Context().Value(userContextKey).(*database.User)
		if user.Role != "admin" {
			http.Error(w, "Admin access required", http.StatusForbidden)
			return
		}
		s.handleDetectIntros(w, r, show, parts[2:])
		return
	}

	// Default: GET show
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.sendShowDetail(w, show)
}

func (s *Server) sendShowDetail(w http.ResponseWriter, show *database.Show) {
	seasons, err := s.db.GetSeasonsByShow(show.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type SeasonWithEpisodes struct {
		database.Season
		Episodes []database.Episode `json:"episodes"`
	}

	type ShowDetail struct {
		database.Show
		Seasons []SeasonWithEpisodes `json:"seasons"`
	}

	detail := ShowDetail{Show: *show}
	for _, season := range seasons {
		episodes, _ := s.db.GetEpisodesBySeason(season.ID)
		if episodes == nil {
			episodes = []database.Episode{}
		}
		detail.Seasons = append(detail.Seasons, SeasonWithEpisodes{
			Season:   season,
			Episodes: episodes,
		})
	}

	json.NewEncoder(w).Encode(detail)
}

func (s *Server) handleEpisode(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Parse path: /api/episodes/{id} or /api/episodes/{id}/segments
	path := strings.TrimPrefix(r.URL.Path, "/api/episodes/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Episode ID required", http.StatusBadRequest)
		return
	}

	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		http.Error(w, "Invalid episode ID", http.StatusBadRequest)
		return
	}

	// Handle sub-paths
	if len(parts) > 1 {
		switch parts[1] {
		case "segments":
			s.handleEpisodeSegments(w, r, id, parts[2:])
			return
		}
	}

	episode, err := s.db.GetEpisode(id)
	if err != nil {
		http.Error(w, "Episode not found", http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Get showId for the episode
		showID, err := s.db.GetShowIDForEpisode(id)
		if err != nil {
			log.Printf("Failed to get show ID for episode %d: %v", id, err)
		}
		response := struct {
			*database.Episode
			ShowID int64 `json:"showId"`
		}{
			Episode: episode,
			ShowID:  showID,
		}
		json.NewEncoder(w).Encode(response)
	case http.MethodDelete:
		// Admin only
		user := r.Context().Value(userContextKey).(*database.User)
		if user.Role != "admin" {
			http.Error(w, "Admin access required", http.StatusForbidden)
			return
		}
		// Delete the file if it exists
		if episode.Path != "" {
			if err := os.Remove(episode.Path); err != nil && !os.IsNotExist(err) {
				log.Printf("Failed to delete episode file: %v", err)
			}
		}
		// Delete from database
		if err := s.db.DeleteEpisode(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleEpisodeSegments handles /api/episodes/{id}/segments
func (s *Server) handleEpisodeSegments(w http.ResponseWriter, r *http.Request, episodeID int64, subParts []string) {
	switch r.Method {
	case http.MethodGet:
		// GET /api/episodes/{id}/segments - Get all segments for episode
		log.Printf("GET segments for episode %d", episodeID)
		segments, err := s.db.GetMediaSegments(episodeID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if segments == nil {
			segments = []database.MediaSegment{}
		}
		log.Printf("Returning %d segments", len(segments))
		json.NewEncoder(w).Encode(segments)

	case http.MethodPost:
		// POST /api/episodes/{id}/segments - Create/update a segment (admin only)
		user := r.Context().Value(userContextKey).(*database.User)
		if user.Role != "admin" {
			http.Error(w, "Admin access required", http.StatusForbidden)
			return
		}

		var req struct {
			SegmentType  string  `json:"segmentType"`
			StartSeconds float64 `json:"startSeconds"`
			EndSeconds   float64 `json:"endSeconds"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate segment type
		validTypes := map[string]bool{"intro": true, "credits": true, "recap": true, "preview": true}
		if !validTypes[req.SegmentType] {
			http.Error(w, "Invalid segment type. Must be: intro, credits, recap, or preview", http.StatusBadRequest)
			return
		}

		segment := &database.MediaSegment{
			EpisodeID:    episodeID,
			SegmentType:  req.SegmentType,
			StartSeconds: req.StartSeconds,
			EndSeconds:   req.EndSeconds,
			Confidence:   1.0,
			Source:       "user", // Manual user entry
		}

		if err := s.db.CreateMediaSegment(segment); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(segment)

	case http.MethodDelete:
		// DELETE /api/episodes/{id}/segments/{segmentId} - Delete a specific segment (admin only)
		user := r.Context().Value(userContextKey).(*database.User)
		if user.Role != "admin" {
			http.Error(w, "Admin access required", http.StatusForbidden)
			return
		}

		if len(subParts) == 0 {
			// Delete all segments for episode
			if err := s.db.DeleteMediaSegmentsByEpisode(episodeID); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}

		segmentID, err := strconv.ParseInt(subParts[0], 10, 64)
		if err != nil {
			http.Error(w, "Invalid segment ID", http.StatusBadRequest)
			return
		}

		if err := s.db.DeleteMediaSegment(segmentID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Single movie handler
func (s *Server) handleMovie(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Parse path: /api/movies/{id} or /api/movies/{id}/refresh or /api/movies/{id}/match
	path := strings.TrimPrefix(r.URL.Path, "/api/movies/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Movie ID required", http.StatusBadRequest)
		return
	}

	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		http.Error(w, "Invalid movie ID", http.StatusBadRequest)
		return
	}

	movie, err := s.db.GetMovie(id)
	if err != nil {
		http.Error(w, "Movie not found", http.StatusNotFound)
		return
	}

	// Check content rating restriction
	user := s.getCurrentUser(r)
	if user != nil && user.ContentRatingLimit != nil && !s.isContentAllowed(user, movie.ContentRating, r) {
		// Content is restricted - check if PIN is required
		if user.RequirePin {
			http.Error(w, "Content restricted - PIN required", http.StatusForbidden)
		} else {
			http.Error(w, "Content not available", http.StatusForbidden)
		}
		return
	}

	// Handle refresh endpoint
	if len(parts) == 2 && parts[1] == "refresh" {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if s.metadata != nil {
			if err := s.metadata.FetchMovieMetadata(movie); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			// Reload movie to get updated data
			movie, _ = s.db.GetMovie(id)
		}
		json.NewEncoder(w).Encode(movie)
		return
	}

	// Handle match endpoint
	if len(parts) == 2 && parts[1] == "match" {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			TmdbID int64 `json:"tmdbId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if s.metadata != nil {
			if err := s.metadata.FetchMovieMetadataByTmdbID(movie, req.TmdbID); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			movie, _ = s.db.GetMovie(id)
		}
		json.NewEncoder(w).Encode(movie)
		return
	}

	// Handle DELETE
	if r.Method == http.MethodDelete {
		// Delete the file if it exists
		if movie.Path != "" {
			if err := os.Remove(movie.Path); err != nil && !os.IsNotExist(err) {
				log.Printf("Failed to delete movie file: %v", err)
			}
		}
		// Delete from database
		if err := s.db.DeleteMovie(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Default: GET movie
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	json.NewEncoder(w).Encode(movie)
}

// handleDetectIntros handles POST /api/shows/{id}/detect-intros
// Triggers audio fingerprint analysis for intro detection across all seasons
func (s *Server) handleDetectIntros(w http.ResponseWriter, r *http.Request, show *database.Show, subParts []string) {
	// Get all seasons for this show
	seasons, err := s.db.GetSeasonsByShow(show.ID)
	if err != nil {
		http.Error(w, "Failed to get seasons: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if len(seasons) == 0 {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "No seasons found for this show",
			"results": []interface{}{},
		})
		return
	}

	// Create intro detector
	detector := scanner.NewIntroDetector(s.db)

	// Check if FFmpeg has chromaprint support
	if !scanner.CheckFFmpegChromaprint() {
		http.Error(w, "FFmpeg chromaprint support not available. Please ensure FFmpeg is compiled with chromaprint.", http.StatusInternalServerError)
		return
	}

	// Analyze each season
	type SeasonResult struct {
		SeasonID     int64  `json:"seasonId"`
		SeasonNumber int    `json:"seasonNumber"`
		Status       string `json:"status"`
		Error        string `json:"error,omitempty"`
	}

	results := make([]SeasonResult, 0, len(seasons))

	for _, season := range seasons {
		result := SeasonResult{
			SeasonID:     season.ID,
			SeasonNumber: season.SeasonNumber,
		}

		if err := detector.AnalyzeSeason(season.ID); err != nil {
			result.Status = "failed"
			result.Error = err.Error()
			log.Printf("Intro detection failed for show %d season %d: %v", show.ID, season.SeasonNumber, err)
		} else {
			result.Status = "completed"
			log.Printf("Intro detection completed for show %d season %d", show.ID, season.SeasonNumber)
		}

		results = append(results, result)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Intro detection completed",
		"results": results,
	})
}
