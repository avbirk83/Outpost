package metadata

import (
	"log"
	"path/filepath"
	"strconv"

	"github.com/outpost/outpost/internal/database"
	"github.com/outpost/outpost/internal/tmdb"
)

type Service struct {
	db       *database.Database
	tmdb     *tmdb.Client
	imageDir string
}

func NewService(db *database.Database, apiKey, imageDir string) *Service {
	return &Service{
		db:       db,
		tmdb:     tmdb.NewClient(apiKey, imageDir),
		imageDir: imageDir,
	}
}

// UpdateAPIKey updates the TMDB client with a new API key
func (s *Service) UpdateAPIKey(apiKey string) {
	s.tmdb = tmdb.NewClient(apiKey, s.imageDir)
}

// GetTMDBClient returns the TMDB client for direct API access
func (s *Service) GetTMDBClient() *tmdb.Client {
	return s.tmdb
}

// FetchMovieMetadata fetches metadata from TMDB for a movie
func (s *Service) FetchMovieMetadata(movie *database.Movie) error {
	// Search for the movie
	searchResult, err := s.tmdb.SearchMovie(movie.Title, movie.Year)
	if err != nil {
		return err
	}

	if len(searchResult.Results) == 0 {
		log.Printf("No TMDB results for movie: %s (%d)", movie.Title, movie.Year)
		return nil
	}

	// Use the first result (best match)
	bestMatch := searchResult.Results[0]

	// Get detailed info
	details, err := s.tmdb.GetMovieDetails(bestMatch.ID)
	if err != nil {
		return err
	}

	// Get content rating
	contentRating, _ := s.tmdb.GetMovieContentRating(bestMatch.ID)

	// Download and cache images
	posterPath, _ := s.tmdb.DownloadImage(details.PosterPath, "w500")
	backdropPath, _ := s.tmdb.DownloadImage(details.BackdropPath, "w1280")

	// Analyze focal point for backdrop
	if backdropPath != "" {
		focalX, focalY, _ := s.tmdb.AnalyzeFocalPoint(backdropPath)
		movie.FocalX = &focalX
		movie.FocalY = &focalY
	}

	// Update movie with metadata
	movie.TmdbID = &details.ID
	if details.ImdbID != "" {
		movie.ImdbID = &details.ImdbID
	}
	if details.OriginalTitle != "" && details.OriginalTitle != details.Title {
		movie.OriginalTitle = &details.OriginalTitle
	}
	if details.Overview != "" {
		movie.Overview = &details.Overview
	}
	if details.Tagline != "" {
		movie.Tagline = &details.Tagline
	}
	if details.Runtime > 0 {
		movie.Runtime = &details.Runtime
	}
	if details.VoteAverage > 0 {
		movie.Rating = &details.VoteAverage
	}
	if contentRating != "" {
		movie.ContentRating = &contentRating
	}
	if len(details.Genres) > 0 {
		genres := tmdb.GenresToJSON(details.Genres)
		movie.Genres = &genres
	}
	if len(details.Credits.Cast) > 0 {
		cast := tmdb.CastToJSON(details.Credits.Cast, 0) // Full cast (0 = no limit)
		movie.Cast = &cast
	}
	if len(details.Credits.Crew) > 0 {
		crew := tmdb.CrewToJSON(details.Credits.Crew, 0) // Full crew (0 = no limit)
		movie.Crew = &crew
	}
	director := tmdb.GetDirector(details.Credits.Crew)
	if director != "" {
		movie.Director = &director
	}
	writer := tmdb.GetWriter(details.Credits.Crew)
	if writer != "" {
		movie.Writer = &writer
	}
	editor := tmdb.GetEditor(details.Credits.Crew)
	if editor != "" {
		movie.Editor = &editor
	}
	producers := tmdb.GetProducers(details.Credits.Crew, 3)
	if producers != "" {
		movie.Producers = &producers
	}
	if details.Status != "" {
		movie.Status = &details.Status
	}
	if details.Budget > 0 {
		movie.Budget = &details.Budget
	}
	if details.Revenue > 0 {
		movie.Revenue = &details.Revenue
	}
	if len(details.ProductionCountries) > 0 {
		country := details.ProductionCountries[0].Name
		movie.Country = &country
	}
	if details.OriginalLanguage != "" {
		movie.OriginalLanguage = &details.OriginalLanguage
	}
	// Extract release dates
	theatrical, digital := tmdb.GetUSReleaseDates(details.ReleaseDates)
	if theatrical != "" {
		movie.TheatricalRelease = &theatrical
	}
	if digital != "" {
		movie.DigitalRelease = &digital
	}
	// Extract studios
	studios := tmdb.GetStudios(details.ProductionCompanies)
	if studios != "" {
		movie.Studios = &studios
	}
	trailers := tmdb.TrailersToJSON(details.Videos)
	if trailers != "" {
		movie.Trailers = &trailers
	}
	if posterPath != "" {
		movie.PosterPath = &posterPath
	}
	if backdropPath != "" {
		movie.BackdropPath = &backdropPath
	}

	// Save to database
	if err := s.db.UpdateMovieMetadata(movie); err != nil {
		return err
	}

	// Process collection if movie belongs to one
	if details.BelongsToCollection != nil {
		s.processMovieCollection(movie, details.BelongsToCollection)
	}

	return nil
}

// FetchMovieMetadataByTmdbID fetches metadata for a specific TMDB ID (manual match)
func (s *Service) FetchMovieMetadataByTmdbID(movie *database.Movie, tmdbID int64) error {
	details, err := s.tmdb.GetMovieDetails(tmdbID)
	if err != nil {
		return err
	}

	contentRating, _ := s.tmdb.GetMovieContentRating(tmdbID)
	posterPath, _ := s.tmdb.DownloadImage(details.PosterPath, "w500")
	backdropPath, _ := s.tmdb.DownloadImage(details.BackdropPath, "w1280")

	// Analyze focal point for backdrop
	if backdropPath != "" {
		focalX, focalY, _ := s.tmdb.AnalyzeFocalPoint(backdropPath)
		movie.FocalX = &focalX
		movie.FocalY = &focalY
	}

	movie.TmdbID = &details.ID
	if details.ImdbID != "" {
		movie.ImdbID = &details.ImdbID
	}
	if details.OriginalTitle != "" && details.OriginalTitle != details.Title {
		movie.OriginalTitle = &details.OriginalTitle
	}
	if details.Overview != "" {
		movie.Overview = &details.Overview
	}
	if details.Tagline != "" {
		movie.Tagline = &details.Tagline
	}
	if details.Runtime > 0 {
		movie.Runtime = &details.Runtime
	}
	if details.VoteAverage > 0 {
		movie.Rating = &details.VoteAverage
	}
	if contentRating != "" {
		movie.ContentRating = &contentRating
	}
	if len(details.Genres) > 0 {
		genres := tmdb.GenresToJSON(details.Genres)
		movie.Genres = &genres
	}
	if len(details.Credits.Cast) > 0 {
		cast := tmdb.CastToJSON(details.Credits.Cast, 0)
		movie.Cast = &cast
	}
	director := tmdb.GetDirector(details.Credits.Crew)
	if director != "" {
		movie.Director = &director
	}
	writer := tmdb.GetWriter(details.Credits.Crew)
	if writer != "" {
		movie.Writer = &writer
	}
	editor := tmdb.GetEditor(details.Credits.Crew)
	if editor != "" {
		movie.Editor = &editor
	}
	producers := tmdb.GetProducers(details.Credits.Crew, 3)
	if producers != "" {
		movie.Producers = &producers
	}
	if details.Status != "" {
		movie.Status = &details.Status
	}
	if details.Budget > 0 {
		movie.Budget = &details.Budget
	}
	if details.Revenue > 0 {
		movie.Revenue = &details.Revenue
	}
	if len(details.ProductionCountries) > 0 {
		country := details.ProductionCountries[0].Name
		movie.Country = &country
	}
	if details.OriginalLanguage != "" {
		movie.OriginalLanguage = &details.OriginalLanguage
	}
	// Extract release dates
	theatrical2, digital2 := tmdb.GetUSReleaseDates(details.ReleaseDates)
	if theatrical2 != "" {
		movie.TheatricalRelease = &theatrical2
	}
	if digital2 != "" {
		movie.DigitalRelease = &digital2
	}
	// Extract studios
	studios2 := tmdb.GetStudios(details.ProductionCompanies)
	if studios2 != "" {
		movie.Studios = &studios2
	}
	trailers := tmdb.TrailersToJSON(details.Videos)
	if trailers != "" {
		movie.Trailers = &trailers
	}
	if posterPath != "" {
		movie.PosterPath = &posterPath
	}
	if backdropPath != "" {
		movie.BackdropPath = &backdropPath
	}

	if err := s.db.UpdateMovieMetadata(movie); err != nil {
		return err
	}

	// Process collection if movie belongs to one
	if details.BelongsToCollection != nil {
		s.processMovieCollection(movie, details.BelongsToCollection)
	}

	return nil
}

// FetchShowMetadata fetches metadata from TMDB for a TV show
func (s *Service) FetchShowMetadata(show *database.Show) error {
	// Search for the show
	searchResult, err := s.tmdb.SearchTV(show.Title, show.Year)
	if err != nil {
		return err
	}

	if len(searchResult.Results) == 0 {
		log.Printf("No TMDB results for show: %s (%d)", show.Title, show.Year)
		return nil
	}

	// Use the first result
	bestMatch := searchResult.Results[0]

	// Get detailed info
	details, err := s.tmdb.GetTVDetails(bestMatch.ID)
	if err != nil {
		return err
	}

	// Get content rating
	contentRating, _ := s.tmdb.GetTVContentRating(bestMatch.ID)

	// Download and cache images
	posterPath, _ := s.tmdb.DownloadImage(details.PosterPath, "w500")
	backdropPath, _ := s.tmdb.DownloadImage(details.BackdropPath, "w1280")

	// Analyze focal point for backdrop
	if backdropPath != "" {
		focalX, focalY, _ := s.tmdb.AnalyzeFocalPoint(backdropPath)
		show.FocalX = &focalX
		show.FocalY = &focalY
	}

	// Update show with metadata
	show.TmdbID = &details.ID
	if details.ExternalIDs.TvdbID > 0 {
		show.TvdbID = &details.ExternalIDs.TvdbID
	}
	if details.ExternalIDs.ImdbID != "" {
		show.ImdbID = &details.ExternalIDs.ImdbID
	}
	if details.OriginalName != "" && details.OriginalName != details.Name {
		show.OriginalTitle = &details.OriginalName
	}
	if details.FirstAirDate != "" && len(details.FirstAirDate) >= 4 {
		year, _ := strconv.Atoi(details.FirstAirDate[:4])
		if year > 0 {
			show.Year = year
		}
	}
	if details.Overview != "" {
		show.Overview = &details.Overview
	}
	if details.Status != "" {
		show.Status = &details.Status
	}
	if details.VoteAverage > 0 {
		show.Rating = &details.VoteAverage
	}
	if contentRating != "" {
		show.ContentRating = &contentRating
	}
	if len(details.Genres) > 0 {
		genres := tmdb.GenresToJSON(details.Genres)
		show.Genres = &genres
	}
	if len(details.Credits.Cast) > 0 {
		cast := tmdb.CastToJSON(details.Credits.Cast, 0)
		show.Cast = &cast
	}
	if len(details.Credits.Crew) > 0 {
		crew := tmdb.CrewToJSON(details.Credits.Crew, 0)
		show.Crew = &crew
	}
	if len(details.Networks) > 0 {
		show.Network = &details.Networks[0].Name
	}
	if posterPath != "" {
		show.PosterPath = &posterPath
	}
	if backdropPath != "" {
		show.BackdropPath = &backdropPath
	}

	// Save show metadata
	if err := s.db.UpdateShowMetadata(show); err != nil {
		return err
	}

	// Fetch season and episode metadata
	return s.fetchSeasonMetadata(show, details.ID)
}

// FetchShowMetadataByTmdbID fetches metadata for a specific TMDB ID (manual match)
func (s *Service) FetchShowMetadataByTmdbID(show *database.Show, tmdbID int64) error {
	details, err := s.tmdb.GetTVDetails(tmdbID)
	if err != nil {
		return err
	}

	contentRating, _ := s.tmdb.GetTVContentRating(tmdbID)
	posterPath, _ := s.tmdb.DownloadImage(details.PosterPath, "w500")
	backdropPath, _ := s.tmdb.DownloadImage(details.BackdropPath, "w1280")

	// Analyze focal point for backdrop
	if backdropPath != "" {
		focalX, focalY, _ := s.tmdb.AnalyzeFocalPoint(backdropPath)
		show.FocalX = &focalX
		show.FocalY = &focalY
	}

	show.TmdbID = &details.ID
	if details.ExternalIDs.TvdbID > 0 {
		show.TvdbID = &details.ExternalIDs.TvdbID
	}
	if details.ExternalIDs.ImdbID != "" {
		show.ImdbID = &details.ExternalIDs.ImdbID
	}
	if details.OriginalName != "" && details.OriginalName != details.Name {
		show.OriginalTitle = &details.OriginalName
	}
	if details.FirstAirDate != "" && len(details.FirstAirDate) >= 4 {
		year, _ := strconv.Atoi(details.FirstAirDate[:4])
		if year > 0 {
			show.Year = year
		}
	}
	if details.Overview != "" {
		show.Overview = &details.Overview
	}
	if details.Status != "" {
		show.Status = &details.Status
	}
	if details.VoteAverage > 0 {
		show.Rating = &details.VoteAverage
	}
	if contentRating != "" {
		show.ContentRating = &contentRating
	}
	if len(details.Genres) > 0 {
		genres := tmdb.GenresToJSON(details.Genres)
		show.Genres = &genres
	}
	if len(details.Credits.Cast) > 0 {
		cast := tmdb.CastToJSON(details.Credits.Cast, 0)
		show.Cast = &cast
	}
	if len(details.Credits.Crew) > 0 {
		crew := tmdb.CrewToJSON(details.Credits.Crew, 0)
		show.Crew = &crew
	}
	if len(details.Networks) > 0 {
		show.Network = &details.Networks[0].Name
	}
	if posterPath != "" {
		show.PosterPath = &posterPath
	}
	if backdropPath != "" {
		show.BackdropPath = &backdropPath
	}

	if err := s.db.UpdateShowMetadata(show); err != nil {
		return err
	}

	return s.fetchSeasonMetadata(show, tmdbID)
}

// fetchSeasonMetadata fetches metadata for all seasons of a show
func (s *Service) fetchSeasonMetadata(show *database.Show, showTmdbID int64) error {
	seasons, err := s.db.GetSeasonsByShow(show.ID)
	if err != nil {
		return err
	}

	for i := range seasons {
		season := &seasons[i]

		// Fetch season details from TMDB
		seasonDetails, err := s.tmdb.GetSeasonDetails(showTmdbID, season.SeasonNumber)
		if err != nil {
			log.Printf("Failed to fetch season %d metadata: %v", season.SeasonNumber, err)
			continue
		}

		// Download season poster
		posterPath, _ := s.tmdb.DownloadImage(seasonDetails.PosterPath, "w500")

		// Update season
		if seasonDetails.Name != "" {
			season.Name = &seasonDetails.Name
		}
		if seasonDetails.Overview != "" {
			season.Overview = &seasonDetails.Overview
		}
		if posterPath != "" {
			season.PosterPath = &posterPath
		}
		if seasonDetails.AirDate != "" {
			season.AirDate = &seasonDetails.AirDate
		}

		if err := s.db.UpdateSeasonMetadata(season); err != nil {
			log.Printf("Failed to update season %d metadata: %v", season.SeasonNumber, err)
			continue
		}

		// Update episode metadata
		episodes, err := s.db.GetEpisodesBySeason(season.ID)
		if err != nil {
			continue
		}

		for j := range episodes {
			ep := &episodes[j]

			// Find matching TMDB episode
			for _, tmdbEp := range seasonDetails.Episodes {
				if tmdbEp.EpisodeNumber == ep.EpisodeNumber {
					// Download still image
					stillPath, _ := s.tmdb.DownloadImage(tmdbEp.StillPath, "w300")

					if tmdbEp.Name != "" {
						ep.Title = tmdbEp.Name
					}
					if tmdbEp.Overview != "" {
						ep.Overview = &tmdbEp.Overview
					}
					if tmdbEp.AirDate != "" {
						ep.AirDate = &tmdbEp.AirDate
					}
					if tmdbEp.Runtime > 0 {
						ep.Runtime = &tmdbEp.Runtime
					}
					if stillPath != "" {
						ep.StillPath = &stillPath
					}

					if err := s.db.UpdateEpisodeMetadata(ep); err != nil {
						log.Printf("Failed to update episode S%02dE%02d metadata: %v",
							season.SeasonNumber, ep.EpisodeNumber, err)
					}
					break
				}
			}
		}
	}

	return nil
}

// SearchMovies searches TMDB for movies (for manual matching)
func (s *Service) SearchMovies(query string, year int) ([]tmdb.MovieResult, error) {
	result, err := s.tmdb.SearchMovie(query, year)
	if err != nil {
		return nil, err
	}
	return result.Results, nil
}

// SearchTV searches TMDB for TV shows (for manual matching)
func (s *Service) SearchTV(query string, year int) ([]tmdb.TVResult, error) {
	result, err := s.tmdb.SearchTV(query, year)
	if err != nil {
		return nil, err
	}
	return result.Results, nil
}

// GetImageURL returns the full URL for a cached image
func GetImageURL(localPath string) string {
	if localPath == "" {
		return ""
	}
	return "/images/" + filepath.ToSlash(localPath)
}

// DiscoverItem represents a movie or TV show from discover endpoints
type DiscoverItem struct {
	ID           int64   `json:"id"`
	Type         string  `json:"type"` // movie or show
	Title        string  `json:"title"`
	Overview     string  `json:"overview"`
	ReleaseDate  string  `json:"releaseDate"`
	PosterPath   string  `json:"posterPath"`
	BackdropPath string  `json:"backdropPath"`
	Rating       float64 `json:"rating"`
	Popularity   float64  `json:"popularity"`
	FocalX       *float64 `json:"focalX,omitempty"`
	FocalY       *float64 `json:"focalY,omitempty"`
}

// DiscoverResult represents paginated discover results
type DiscoverResult struct {
	Page         int            `json:"page"`
	TotalPages   int            `json:"totalPages"`
	TotalResults int            `json:"totalResults"`
	Results      []DiscoverItem `json:"results"`
}

// GetTrendingMovies returns trending movies
func (s *Service) GetTrendingMovies(page int) (*DiscoverResult, error) {
	result, err := s.tmdb.GetTrendingMovies(page)
	if err != nil {
		return nil, err
	}
	return s.convertMovieResults(result), nil
}

// GetTrendingTV returns trending TV shows
func (s *Service) GetTrendingTV(page int) (*DiscoverResult, error) {
	result, err := s.tmdb.GetTrendingTV(page)
	if err != nil {
		return nil, err
	}
	return s.convertTVResults(result), nil
}

// GetPopularMovies returns popular movies
func (s *Service) GetPopularMovies(page int) (*DiscoverResult, error) {
	result, err := s.tmdb.GetPopularMovies(page)
	if err != nil {
		return nil, err
	}
	return s.convertMovieResults(result), nil
}

// GetPopularTV returns popular TV shows
func (s *Service) GetPopularTV(page int) (*DiscoverResult, error) {
	result, err := s.tmdb.GetPopularTV(page)
	if err != nil {
		return nil, err
	}
	return s.convertTVResults(result), nil
}

// GetUpcomingMovies returns upcoming movies
func (s *Service) GetUpcomingMovies(page int) (*DiscoverResult, error) {
	result, err := s.tmdb.GetUpcomingMovies(page)
	if err != nil {
		return nil, err
	}
	return s.convertMovieResults(result), nil
}

// GetTheatricalReleases returns upcoming theatrical releases with date filtering
func (s *Service) GetTheatricalReleases(region string, page int) (*DiscoverResult, error) {
	result, err := s.tmdb.DiscoverTheatricalReleases(region, page)
	if err != nil {
		return nil, err
	}
	return s.convertMovieResults(result), nil
}

// GetUpcomingTV returns upcoming TV shows
func (s *Service) GetUpcomingTV(page int) (*DiscoverResult, error) {
	result, err := s.tmdb.DiscoverUpcomingTV(page)
	if err != nil {
		return nil, err
	}
	return s.convertTVResults(result), nil
}

// GetTopRatedMovies returns top rated movies
func (s *Service) GetTopRatedMovies(page int) (*DiscoverResult, error) {
	result, err := s.tmdb.GetTopRatedMovies(page)
	if err != nil {
		return nil, err
	}
	return s.convertMovieResults(result), nil
}

// GetTopRatedTV returns top rated TV shows
func (s *Service) GetTopRatedTV(page int) (*DiscoverResult, error) {
	result, err := s.tmdb.GetTopRatedTV(page)
	if err != nil {
		return nil, err
	}
	return s.convertTVResults(result), nil
}

// GetMovieGenres returns all movie genres from TMDB
func (s *Service) GetMovieGenres() ([]tmdb.Genre, error) {
	return s.tmdb.GetMovieGenres()
}

// GetTVGenres returns all TV genres from TMDB
func (s *Service) GetTVGenres() ([]tmdb.Genre, error) {
	return s.tmdb.GetTVGenres()
}

// GetMoviesByGenre returns movies by genre
func (s *Service) GetMoviesByGenre(genreID int, page int) (*DiscoverResult, error) {
	result, err := s.tmdb.GetMoviesByGenre(genreID, page)
	if err != nil {
		return nil, err
	}
	return s.convertMovieResults(result), nil
}

// GetTVByGenre returns TV shows by genre
func (s *Service) GetTVByGenre(genreID int, page int) (*DiscoverResult, error) {
	result, err := s.tmdb.GetTVByGenre(genreID, page)
	if err != nil {
		return nil, err
	}
	return s.convertTVResults(result), nil
}

func (s *Service) convertMovieResults(result *tmdb.DiscoverMovieResult) *DiscoverResult {
	items := make([]DiscoverItem, len(result.Results))
	for i, r := range result.Results {
		items[i] = DiscoverItem{
			ID:           r.ID,
			Type:         "movie",
			Title:        r.Title,
			Overview:     r.Overview,
			ReleaseDate:  r.ReleaseDate,
			PosterPath:   r.PosterPath,
			BackdropPath: r.BackdropPath,
			Rating:       r.VoteAverage,
			Popularity:   r.Popularity,
		}
	}
	return &DiscoverResult{
		Page:         result.Page,
		TotalPages:   result.TotalPages,
		TotalResults: result.TotalResults,
		Results:      items,
	}
}

func (s *Service) convertTVResults(result *tmdb.DiscoverTVResult) *DiscoverResult {
	items := make([]DiscoverItem, len(result.Results))
	for i, r := range result.Results {
		items[i] = DiscoverItem{
			ID:           r.ID,
			Type:         "show",
			Title:        r.Name,
			Overview:     r.Overview,
			ReleaseDate:  r.FirstAirDate,
			PosterPath:   r.PosterPath,
			BackdropPath: r.BackdropPath,
			Rating:       r.VoteAverage,
			Popularity:   r.Popularity,
		}
	}
	return &DiscoverResult{
		Page:         result.Page,
		TotalPages:   result.TotalPages,
		TotalResults: result.TotalResults,
		Results:      items,
	}
}

// DiscoverMovieDetail contains detailed info for a movie from TMDB
type DiscoverMovieDetail struct {
	ID                  int64              `json:"id"`
	Title               string             `json:"title"`
	Overview            string             `json:"overview"`
	Tagline             string             `json:"tagline"`
	ReleaseDate         string             `json:"releaseDate"`
	Runtime             int                `json:"runtime"`
	Rating              float64            `json:"rating"`
	ContentRating       string             `json:"contentRating,omitempty"`
	PosterPath          string             `json:"posterPath"`
	BackdropPath        string             `json:"backdropPath"`
	Genres              []string           `json:"genres"`
	Cast                []CastMember       `json:"cast"`
	Crew                []CrewMember       `json:"crew"`
	Director            string             `json:"director"`
	IMDbID              string             `json:"imdbId,omitempty"`
	Status              string             `json:"status"`
	Budget              int64              `json:"budget,omitempty"`
	Revenue             int64              `json:"revenue,omitempty"`
	OriginalLanguage    string             `json:"originalLanguage,omitempty"`
	ProductionCountries []string           `json:"productionCountries,omitempty"`
	ProductionCompanies []string           `json:"productionCompanies,omitempty"`
	TrailerKey          string             `json:"trailerKey,omitempty"`
	Recommendations     []RecommendedItem  `json:"recommendations,omitempty"`
}

// SeasonSummary contains summary info for a TV season
type SeasonSummary struct {
	SeasonNumber int    `json:"season_number"`
	Name         string `json:"name"`
	Overview     string `json:"overview,omitempty"`
	PosterPath   string `json:"poster_path,omitempty"`
	AirDate      string `json:"air_date,omitempty"`
	EpisodeCount int    `json:"episode_count"`
}

// DiscoverShowDetail contains detailed info for a TV show from TMDB
type DiscoverShowDetail struct {
	ID                  int64              `json:"id"`
	Title               string             `json:"title"`
	Overview            string             `json:"overview"`
	FirstAirDate        string             `json:"firstAirDate"`
	Status              string             `json:"status"`
	Rating              float64            `json:"rating"`
	ContentRating       string             `json:"contentRating,omitempty"`
	PosterPath          string             `json:"posterPath"`
	BackdropPath        string             `json:"backdropPath"`
	Genres              []string           `json:"genres"`
	Networks            []string           `json:"networks"`
	Seasons             int                `json:"seasons"`
	Episodes            int                `json:"episodes"`
	SeasonDetails       []SeasonSummary    `json:"seasonDetails,omitempty"`
	Cast                []CastMember       `json:"cast"`
	Crew                []CrewMember       `json:"crew"`
	IMDbID              string             `json:"imdbId,omitempty"`
	OriginalLanguage    string             `json:"originalLanguage,omitempty"`
	ProductionCountries []string           `json:"productionCountries,omitempty"`
	TrailerKey          string             `json:"trailerKey,omitempty"`
	Recommendations     []RecommendedItem  `json:"recommendations,omitempty"`
}

type CastMember struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Character string `json:"character"`
	Photo     string `json:"photo"`
}

type CrewMember struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Job   string `json:"job"`
	Photo string `json:"photo"`
}

type RecommendedItem struct {
	ID            int64   `json:"id"`
	Title         string  `json:"title"`
	PosterPath    string  `json:"posterPath"`
	ReleaseDate   string  `json:"releaseDate"`
	Rating        float64 `json:"rating"`
	MediaType     string  `json:"mediaType"`
	Runtime       int     `json:"runtime,omitempty"`
	ContentRating string  `json:"contentRating,omitempty"`
}

// GetMovieDetail gets detailed info for a movie from TMDB
func (s *Service) GetMovieDetail(tmdbID int64) (*DiscoverMovieDetail, error) {
	details, err := s.tmdb.GetMovieDetails(tmdbID)
	if err != nil {
		return nil, err
	}

	genres := make([]string, len(details.Genres))
	for i, g := range details.Genres {
		genres[i] = g.Name
	}

	cast := make([]CastMember, 0)
	for _, c := range details.Credits.Cast {
		cast = append(cast, CastMember{
			ID:        c.ID,
			Name:      c.Name,
			Character: c.Character,
			Photo:     c.ProfilePath,
		})
	}

	director := ""
	crew := make([]CrewMember, 0)
	importantJobs := map[string]bool{
		"Director": true, "Producer": true, "Executive Producer": true,
		"Writer": true, "Screenplay": true, "Director of Photography": true,
		"Original Music Composer": true, "Editor": true, "Cinematographer": true,
		"Costume Design": true, "Production Design": true, "Composer": true,
	}
	for _, c := range details.Credits.Crew {
		if c.Job == "Director" && director == "" {
			director = c.Name
		}
		if importantJobs[c.Job] {
			crew = append(crew, CrewMember{
				ID:    c.ID,
				Name:  c.Name,
				Job:   c.Job,
				Photo: c.ProfilePath,
			})
		}
	}

	// Production countries
	countries := make([]string, len(details.ProductionCountries))
	for i, c := range details.ProductionCountries {
		countries[i] = c.ISO31661
	}

	// Production companies
	companies := make([]string, 0)
	for i, c := range details.ProductionCompanies {
		if i >= 3 {
			break
		}
		companies = append(companies, c.Name)
	}

	// Get trailer
	trailerKey := ""
	for _, v := range details.Videos.Results {
		if v.Type == "Trailer" && v.Site == "YouTube" {
			trailerKey = v.Key
			break
		}
	}

	// Get content rating
	contentRating, _ := s.tmdb.GetMovieContentRating(tmdbID)

	// Get recommendations from embedded response (basic info only)
	recommendations := make([]RecommendedItem, 0)
	for _, r := range details.Recommendations.Results {
		recommendations = append(recommendations, RecommendedItem{
			ID:          r.ID,
			Title:       r.Title,
			PosterPath:  r.PosterPath,
			ReleaseDate: r.ReleaseDate,
			Rating:      r.VoteAverage,
			MediaType:   "movie",
		})
	}

	return &DiscoverMovieDetail{
		ID:                  details.ID,
		Title:               details.Title,
		Overview:            details.Overview,
		Tagline:             details.Tagline,
		ReleaseDate:         details.ReleaseDate,
		Runtime:             details.Runtime,
		Rating:              details.VoteAverage,
		ContentRating:       contentRating,
		PosterPath:          details.PosterPath,
		BackdropPath:        details.BackdropPath,
		Genres:              genres,
		Cast:                cast,
		Crew:                crew,
		Director:            director,
		IMDbID:              details.ImdbID,
		Status:              details.Status,
		Budget:              details.Budget,
		Revenue:             details.Revenue,
		OriginalLanguage:    details.OriginalLanguage,
		ProductionCountries: countries,
		ProductionCompanies: companies,
		TrailerKey:          trailerKey,
		Recommendations:     recommendations,
	}, nil
}

// GetShowDetail gets detailed info for a TV show from TMDB
func (s *Service) GetShowDetail(tmdbID int64) (*DiscoverShowDetail, error) {
	details, err := s.tmdb.GetTVDetails(tmdbID)
	if err != nil {
		return nil, err
	}

	genres := make([]string, len(details.Genres))
	for i, g := range details.Genres {
		genres[i] = g.Name
	}

	networks := make([]string, len(details.Networks))
	for i, n := range details.Networks {
		networks[i] = n.Name
	}

	cast := make([]CastMember, 0)
	for _, c := range details.Credits.Cast {
		cast = append(cast, CastMember{
			ID:        c.ID,
			Name:      c.Name,
			Character: c.Character,
			Photo:     c.ProfilePath,
		})
	}

	// Crew
	crew := make([]CrewMember, 0)
	importantJobs := map[string]bool{
		"Creator": true, "Executive Producer": true, "Producer": true,
		"Director": true, "Writer": true, "Showrunner": true,
		"Director of Photography": true, "Original Music Composer": true,
		"Editor": true, "Cinematographer": true, "Composer": true,
	}
	for _, c := range details.Credits.Crew {
		if importantJobs[c.Job] {
			crew = append(crew, CrewMember{
				ID:    c.ID,
				Name:  c.Name,
				Job:   c.Job,
				Photo: c.ProfilePath,
			})
		}
	}

	// Production countries
	countries := make([]string, len(details.ProductionCountries))
	for i, c := range details.ProductionCountries {
		countries[i] = c.ISO31661
	}

	// Get trailer
	trailerKey := ""
	for _, v := range details.Videos.Results {
		if v.Type == "Trailer" && v.Site == "YouTube" {
			trailerKey = v.Key
			break
		}
	}

	// Get content rating
	contentRating, _ := s.tmdb.GetTVContentRating(tmdbID)

	// Get recommendations from embedded response (basic info only)
	recommendations := make([]RecommendedItem, 0)
	for _, r := range details.Recommendations.Results {
		recommendations = append(recommendations, RecommendedItem{
			ID:          r.ID,
			Title:       r.Name,
			PosterPath:  r.PosterPath,
			ReleaseDate: r.FirstAirDate,
			Rating:      r.VoteAverage,
			MediaType:   "tv",
		})
	}

	// Count total episodes and build season details
	totalEpisodes := 0
	seasonDetails := make([]SeasonSummary, 0)
	for _, s := range details.Seasons {
		totalEpisodes += s.EpisodeCount
		seasonDetails = append(seasonDetails, SeasonSummary{
			SeasonNumber: s.SeasonNumber,
			Name:         s.Name,
			Overview:     s.Overview,
			PosterPath:   s.PosterPath,
			AirDate:      s.AirDate,
			EpisodeCount: s.EpisodeCount,
		})
	}

	return &DiscoverShowDetail{
		ID:                  details.ID,
		Title:               details.Name,
		Overview:            details.Overview,
		FirstAirDate:        details.FirstAirDate,
		Status:              details.Status,
		Rating:              details.VoteAverage,
		ContentRating:       contentRating,
		PosterPath:          details.PosterPath,
		BackdropPath:        details.BackdropPath,
		Genres:              genres,
		Networks:            networks,
		Seasons:             len(details.Seasons),
		Episodes:            totalEpisodes,
		SeasonDetails:       seasonDetails,
		Cast:                cast,
		Crew:                crew,
		IMDbID:              details.ExternalIDs.ImdbID,
		OriginalLanguage:    details.OriginalLanguage,
		ProductionCountries: countries,
		TrailerKey:          trailerKey,
		Recommendations:     recommendations,
	}, nil
}

// processMovieCollection creates or updates a collection when a movie belongs to a TMDB collection
func (s *Service) processMovieCollection(movie *database.Movie, tmdbColl *tmdb.MovieCollection) {
	if tmdbColl == nil || movie.TmdbID == nil {
		return
	}

	// Check if collection already exists
	existingColl, err := s.db.GetCollectionByTmdbID(tmdbColl.ID)
	if err == nil && existingColl != nil {
		// Collection exists, just update the media_id for this movie
		s.db.UpdateCollectionItemMediaID(*movie.TmdbID, "movie", movie.ID)
		return
	}

	// Collection doesn't exist, fetch full details from TMDB
	collDetails, err := s.tmdb.GetCollectionDetails(tmdbColl.ID)
	if err != nil {
		log.Printf("Failed to fetch collection details for %s: %v", tmdbColl.Name, err)
		return
	}

	// Download collection images
	posterPath, _ := s.tmdb.DownloadImage(collDetails.PosterPath, "w500")
	backdropPath, _ := s.tmdb.DownloadImage(collDetails.BackdropPath, "w1280")

	// Create the collection
	tmdbID := collDetails.ID
	coll := &database.Collection{
		Name:             collDetails.Name,
		TmdbCollectionID: &tmdbID,
		IsAuto:           true,
		SortOrder:        "release",
	}
	if collDetails.Overview != "" {
		coll.Description = &collDetails.Overview
	}
	if posterPath != "" {
		coll.PosterPath = &posterPath
	}
	if backdropPath != "" {
		coll.BackdropPath = &backdropPath
	}

	if err := s.db.CreateCollection(coll); err != nil {
		log.Printf("Failed to create collection %s: %v", collDetails.Name, err)
		return
	}

	log.Printf("Created collection: %s with %d movies", collDetails.Name, len(collDetails.Parts))

	// Add all parts as collection items
	for i, part := range collDetails.Parts {
		year := 0
		if len(part.ReleaseDate) >= 4 {
			year, _ = strconv.Atoi(part.ReleaseDate[:4])
		}

		// Download poster for the part
		partPoster, _ := s.tmdb.DownloadImage(part.PosterPath, "w500")

		item := &database.CollectionItem{
			CollectionID: coll.ID,
			MediaType:    "movie",
			TmdbID:       part.ID,
			Title:        part.Title,
			Year:         year,
			SortOrder:    i,
		}
		if partPoster != "" {
			item.PosterPath = &partPoster
		}

		// Check if this movie is already in the library
		existingMovie, err := s.db.GetMovieByTmdb(part.ID)
		if err == nil && existingMovie != nil {
			item.MediaID = &existingMovie.ID
		}

		if err := s.db.AddCollectionItem(item); err != nil {
			log.Printf("Failed to add collection item %s: %v", part.Title, err)
		}
	}
}
