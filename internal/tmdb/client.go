package tmdb

import (
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/muesli/smartcrop"
	"github.com/muesli/smartcrop/nfnt"
)

const (
	baseURL     = "https://api.themoviedb.org/3"
	imageBaseURL = "https://image.tmdb.org/t/p"
)

type Client struct {
	apiKey     string
	httpClient *http.Client
	imageDir   string
}

func NewClient(apiKey, imageDir string) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		imageDir: imageDir,
	}
}

// Movie types
type MovieSearchResult struct {
	Page         int           `json:"page"`
	Results      []MovieResult `json:"results"`
	TotalPages   int           `json:"total_pages"`
	TotalResults int           `json:"total_results"`
}

type MovieResult struct {
	ID            int64   `json:"id"`
	Title         string  `json:"title"`
	OriginalTitle string  `json:"original_title"`
	Overview      string  `json:"overview"`
	ReleaseDate   string  `json:"release_date"`
	PosterPath    string  `json:"poster_path"`
	BackdropPath  string  `json:"backdrop_path"`
	VoteAverage   float64 `json:"vote_average"`
	Popularity    float64 `json:"popularity"`
}

// MovieCollection represents the collection a movie belongs to
type MovieCollection struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	PosterPath   string `json:"poster_path"`
	BackdropPath string `json:"backdrop_path"`
}

// CollectionDetails represents full details of a TMDB collection
type CollectionDetails struct {
	ID           int64            `json:"id"`
	Name         string           `json:"name"`
	Overview     string           `json:"overview"`
	PosterPath   string           `json:"poster_path"`
	BackdropPath string           `json:"backdrop_path"`
	Parts        []CollectionPart `json:"parts"`
}

// CollectionPart represents a movie in a collection
type CollectionPart struct {
	ID          int64   `json:"id"`
	Title       string  `json:"title"`
	Overview    string  `json:"overview"`
	ReleaseDate string  `json:"release_date"`
	PosterPath  string  `json:"poster_path"`
	VoteAverage float64 `json:"vote_average"`
}

type MovieDetails struct {
	ID                    int64                 `json:"id"`
	ImdbID                string                `json:"imdb_id"`
	Title                 string                `json:"title"`
	OriginalTitle         string                `json:"original_title"`
	Overview              string                `json:"overview"`
	Tagline               string                `json:"tagline"`
	ReleaseDate           string                `json:"release_date"`
	Runtime               int                   `json:"runtime"`
	VoteAverage           float64               `json:"vote_average"`
	PosterPath            string                `json:"poster_path"`
	BackdropPath          string                `json:"backdrop_path"`
	Genres                []Genre               `json:"genres"`
	Credits               Credits               `json:"credits"`
	Status                string                `json:"status"`
	Budget                int64                 `json:"budget"`
	Revenue               int64                 `json:"revenue"`
	OriginalLanguage      string                `json:"original_language"`
	ProductionCountries   []ProductionCountry   `json:"production_countries"`
	ProductionCompanies   []ProductionCompany   `json:"production_companies"`
	Videos                Videos                `json:"videos"`
	ReleaseDates          ReleaseDatesResult    `json:"release_dates"`
	Recommendations       MovieSearchResult     `json:"recommendations"`
	BelongsToCollection   *MovieCollection      `json:"belongs_to_collection"`
}

type ProductionCountry struct {
	ISO31661 string `json:"iso_3166_1"`
	Name     string `json:"name"`
}

type ProductionCompany struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	LogoPath      string `json:"logo_path"`
	OriginCountry string `json:"origin_country"`
}

type ReleaseDatesResult struct {
	Results []CountryReleaseDates `json:"results"`
}

type CountryReleaseDates struct {
	ISO31661     string        `json:"iso_3166_1"`
	ReleaseDates []ReleaseDate `json:"release_dates"`
}

type ReleaseDate struct {
	Certification string `json:"certification"`
	ISO6391       string `json:"iso_639_1"`
	ReleaseDate   string `json:"release_date"`
	Type          int    `json:"type"` // 1=Premiere, 2=Theatrical (limited), 3=Theatrical, 4=Digital, 5=Physical, 6=TV
	Note          string `json:"note"`
}

type Videos struct {
	Results []Video `json:"results"`
}

type Video struct {
	ID       string `json:"id"`
	Key      string `json:"key"`
	Name     string `json:"name"`
	Site     string `json:"site"`
	Type     string `json:"type"`
	Official bool   `json:"official"`
}

type Genre struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Credits struct {
	Cast []CastMember `json:"cast"`
	Crew []CrewMember `json:"crew"`
}

type CastMember struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Character   string `json:"character"`
	Order       int    `json:"order"`
	ProfilePath string `json:"profile_path"`
}

type CrewMember struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Job         string `json:"job"`
	Department  string `json:"department"`
	ProfilePath string `json:"profile_path"`
}

// TV types
type TVSearchResult struct {
	Page         int        `json:"page"`
	Results      []TVResult `json:"results"`
	TotalPages   int        `json:"total_pages"`
	TotalResults int        `json:"total_results"`
}

type TVResult struct {
	ID            int64   `json:"id"`
	Name          string  `json:"name"`
	OriginalName  string  `json:"original_name"`
	Overview      string  `json:"overview"`
	FirstAirDate  string  `json:"first_air_date"`
	PosterPath    string  `json:"poster_path"`
	BackdropPath  string  `json:"backdrop_path"`
	VoteAverage   float64 `json:"vote_average"`
	Popularity    float64 `json:"popularity"`
}

type TVDetails struct {
	ID                  int64               `json:"id"`
	Name                string              `json:"name"`
	OriginalName        string              `json:"original_name"`
	Overview            string              `json:"overview"`
	FirstAirDate        string              `json:"first_air_date"`
	Status              string              `json:"status"`
	VoteAverage         float64             `json:"vote_average"`
	PosterPath          string              `json:"poster_path"`
	BackdropPath        string              `json:"backdrop_path"`
	Genres              []Genre             `json:"genres"`
	Networks            []Network           `json:"networks"`
	Seasons             []SeasonInfo        `json:"seasons"`
	Credits             Credits             `json:"credits"`
	ExternalIDs         ExternalIDs         `json:"external_ids"`
	Videos              Videos              `json:"videos"`
	Recommendations     TVSearchResult      `json:"recommendations"`
	OriginalLanguage    string              `json:"original_language"`
	ProductionCountries []ProductionCountry `json:"production_countries"`
}

type Network struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type SeasonInfo struct {
	ID           int64  `json:"id"`
	SeasonNumber int    `json:"season_number"`
	Name         string `json:"name"`
	Overview     string `json:"overview"`
	PosterPath   string `json:"poster_path"`
	AirDate      string `json:"air_date"`
	EpisodeCount int    `json:"episode_count"`
}

type ExternalIDs struct {
	ImdbID string `json:"imdb_id"`
	TvdbID int64  `json:"tvdb_id"`
}

type SeasonDetails struct {
	ID           int64           `json:"id"`
	SeasonNumber int             `json:"season_number"`
	Name         string          `json:"name"`
	Overview     string          `json:"overview"`
	PosterPath   string          `json:"poster_path"`
	AirDate      string          `json:"air_date"`
	Episodes     []EpisodeInfo   `json:"episodes"`
}

type EpisodeInfo struct {
	ID            int64   `json:"id"`
	EpisodeNumber int     `json:"episode_number"`
	Name          string  `json:"name"`
	Overview      string  `json:"overview"`
	AirDate       string  `json:"air_date"`
	Runtime       int     `json:"runtime"`
	StillPath     string  `json:"still_path"`
	VoteAverage   float64 `json:"vote_average"`
}

// Content rating types
type ContentRatingsResponse struct {
	Results []ContentRating `json:"results"`
}

type ContentRating struct {
	ISO31661 string `json:"iso_3166_1"`
	Rating   string `json:"rating"`
}

// Person types
type PersonDetails struct {
	ID                 int64  `json:"id"`
	Name               string `json:"name"`
	Biography          string `json:"biography"`
	Birthday           string `json:"birthday"`
	Deathday           string `json:"deathday"`
	PlaceOfBirth       string `json:"place_of_birth"`
	ProfilePath        string `json:"profile_path"`
	KnownForDepartment string `json:"known_for_department"`
	Gender             int    `json:"gender"`
}

type PersonCombinedCredits struct {
	Cast []PersonCreditCast `json:"cast"`
	Crew []PersonCreditCrew `json:"crew"`
}

type PersonCreditCast struct {
	ID           int64   `json:"id"`
	MediaType    string  `json:"media_type"`
	Title        string  `json:"title"`
	Name         string  `json:"name"`
	Character    string  `json:"character"`
	PosterPath   string  `json:"poster_path"`
	ReleaseDate  string  `json:"release_date"`
	FirstAirDate string  `json:"first_air_date"`
	VoteAverage  float64 `json:"vote_average"`
	Popularity   float64 `json:"popularity"`
}

type PersonCreditCrew struct {
	ID           int64   `json:"id"`
	MediaType    string  `json:"media_type"`
	Title        string  `json:"title"`
	Name         string  `json:"name"`
	Job          string  `json:"job"`
	Department   string  `json:"department"`
	PosterPath   string  `json:"poster_path"`
	ReleaseDate  string  `json:"release_date"`
	FirstAirDate string  `json:"first_air_date"`
	Popularity   float64 `json:"popularity"`
}

// API methods

func (c *Client) get(endpoint string, params map[string]string) ([]byte, error) {
	u, err := url.Parse(baseURL + endpoint)
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("api_key", c.apiKey)
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	resp, err := c.httpClient.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TMDB API error: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// SearchMovie searches for movies by title and optional year
func (c *Client) SearchMovie(title string, year int) (*MovieSearchResult, error) {
	params := map[string]string{"query": title}
	if year > 0 {
		params["year"] = strconv.Itoa(year)
	}

	data, err := c.get("/search/movie", params)
	if err != nil {
		return nil, err
	}

	var result MovieSearchResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetMovieDetails gets detailed info about a movie including credits
func (c *Client) GetMovieDetails(tmdbID int64) (*MovieDetails, error) {
	data, err := c.get(fmt.Sprintf("/movie/%d", tmdbID), map[string]string{
		"append_to_response": "credits,videos,release_dates,recommendations",
	})
	if err != nil {
		return nil, err
	}

	var result MovieDetails
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetMovieContentRating gets the US content rating for a movie
func (c *Client) GetMovieContentRating(tmdbID int64) (string, error) {
	data, err := c.get(fmt.Sprintf("/movie/%d/release_dates", tmdbID), nil)
	if err != nil {
		return "", err
	}

	var result ReleaseDatesResult
	if err := json.Unmarshal(data, &result); err != nil {
		return "", err
	}

	// Look for US rating
	for _, r := range result.Results {
		if r.ISO31661 == "US" {
			for _, rd := range r.ReleaseDates {
				if rd.Certification != "" {
					return rd.Certification, nil
				}
			}
		}
	}

	return "", nil
}

// GetMovieRecommendations gets recommended movies similar to a given movie
func (c *Client) GetMovieRecommendations(tmdbID int64) (*MovieSearchResult, error) {
	data, err := c.get(fmt.Sprintf("/movie/%d/recommendations", tmdbID), nil)
	if err != nil {
		return nil, err
	}

	var result MovieSearchResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// EnrichedMovieResult extends MovieResult with runtime and content rating
type EnrichedMovieResult struct {
	MovieResult
	Runtime       int    `json:"runtime"`
	ContentRating string `json:"content_rating"`
}

// EnrichedMovieSearchResult is like MovieSearchResult but with enriched results
type EnrichedMovieSearchResult struct {
	Page         int                   `json:"page"`
	Results      []EnrichedMovieResult `json:"results"`
	TotalPages   int                   `json:"total_pages"`
	TotalResults int                   `json:"total_results"`
}

// GetEnrichedMovieRecommendations gets recommendations with runtime and content rating
func (c *Client) GetEnrichedMovieRecommendations(tmdbID int64, limit int) (*EnrichedMovieSearchResult, error) {
	// First get basic recommendations
	basic, err := c.GetMovieRecommendations(tmdbID)
	if err != nil {
		return nil, err
	}

	// Limit the number we enrich to avoid too many API calls
	if limit <= 0 || limit > len(basic.Results) {
		limit = len(basic.Results)
	}
	if limit > 20 {
		limit = 20 // Cap at 20 to avoid excessive API calls
	}

	enriched := &EnrichedMovieSearchResult{
		Page:         basic.Page,
		TotalPages:   basic.TotalPages,
		TotalResults: basic.TotalResults,
		Results:      make([]EnrichedMovieResult, 0, limit),
	}

	// Fetch details for each recommendation (in parallel would be better but keeping simple)
	for i := 0; i < limit && i < len(basic.Results); i++ {
		rec := basic.Results[i]
		enrichedRec := EnrichedMovieResult{MovieResult: rec}

		// Try to get details for runtime and certification
		details, err := c.GetMovieDetails(rec.ID)
		if err == nil {
			enrichedRec.Runtime = details.Runtime
			enrichedRec.ContentRating = c.extractUSCertification(details)
		}

		enriched.Results = append(enriched.Results, enrichedRec)
	}

	return enriched, nil
}

// extractUSCertification gets the US content rating from movie details
func (c *Client) extractUSCertification(details *MovieDetails) string {
	for _, country := range details.ReleaseDates.Results {
		if country.ISO31661 == "US" {
			for _, release := range country.ReleaseDates {
				if release.Certification != "" {
					return release.Certification
				}
			}
		}
	}
	return ""
}

// SearchTV searches for TV shows by title and optional year
func (c *Client) SearchTV(title string, year int) (*TVSearchResult, error) {
	params := map[string]string{"query": title}
	if year > 0 {
		params["first_air_date_year"] = strconv.Itoa(year)
	}

	data, err := c.get("/search/tv", params)
	if err != nil {
		return nil, err
	}

	var result TVSearchResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetTVDetails gets detailed info about a TV show including credits, external IDs, videos, and recommendations
func (c *Client) GetTVDetails(tmdbID int64) (*TVDetails, error) {
	data, err := c.get(fmt.Sprintf("/tv/%d", tmdbID), map[string]string{
		"append_to_response": "credits,external_ids,videos,recommendations",
	})
	if err != nil {
		return nil, err
	}

	var result TVDetails
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetTVContentRating gets the US content rating for a TV show
func (c *Client) GetTVContentRating(tmdbID int64) (string, error) {
	data, err := c.get(fmt.Sprintf("/tv/%d/content_ratings", tmdbID), nil)
	if err != nil {
		return "", err
	}

	var result ContentRatingsResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return "", err
	}

	// Look for US rating
	for _, r := range result.Results {
		if r.ISO31661 == "US" {
			return r.Rating, nil
		}
	}

	return "", nil
}

// GetSeasonDetails gets detailed info about a season including episodes
func (c *Client) GetSeasonDetails(showTmdbID int64, seasonNumber int) (*SeasonDetails, error) {
	data, err := c.get(fmt.Sprintf("/tv/%d/season/%d", showTmdbID, seasonNumber), nil)
	if err != nil {
		return nil, err
	}

	var result SeasonDetails
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// DownloadImage downloads an image from TMDB and caches it locally
// Returns the local path relative to the images directory
func (c *Client) DownloadImage(tmdbPath string, size string) (string, error) {
	if tmdbPath == "" {
		return "", nil
	}

	// Create filename from TMDB path
	filename := strings.TrimPrefix(tmdbPath, "/")
	localPath := filepath.Join(size, filename)
	fullPath := filepath.Join(c.imageDir, localPath)

	// Check if already cached
	if _, err := os.Stat(fullPath); err == nil {
		return localPath, nil
	}

	// Create directory if needed
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return "", err
	}

	// Download image
	imageURL := fmt.Sprintf("%s/%s%s", imageBaseURL, size, tmdbPath)
	resp, err := c.httpClient.Get(imageURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download image: %d", resp.StatusCode)
	}

	// Save to file
	file, err := os.Create(fullPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if _, err := io.Copy(file, resp.Body); err != nil {
		return "", err
	}

	return localPath, nil
}

// AnalyzeFocalPoint analyzes an image and returns the focal point as percentages (0.0-1.0)
// Returns (focalX, focalY, error) where (0.5, 0.5) is center
func (c *Client) AnalyzeFocalPoint(imagePath string) (float64, float64, error) {
	fullPath := filepath.Join(c.imageDir, imagePath)

	file, err := os.Open(fullPath)
	if err != nil {
		return 0.5, 0.25, nil // Default to center-top on error
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return 0.5, 0.25, nil // Default to center-top on error
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Use smartcrop to find the best crop area
	// We'll use a 16:9 aspect ratio for backdrop analysis
	resizer := nfnt.NewDefaultResizer()
	analyzer := smartcrop.NewAnalyzer(resizer)

	// Find best crop for a square to find the focal point
	cropWidth := width
	cropHeight := width // Square crop to find focal area
	if cropHeight > height {
		cropHeight = height
		cropWidth = height
	}

	crop, err := analyzer.FindBestCrop(img, cropWidth, cropHeight)
	if err != nil {
		return 0.5, 0.25, nil // Default to center-top on error
	}

	// Calculate focal point as percentage of image dimensions
	// The crop rectangle gives us where the "interesting" content is
	// image.Rectangle uses Min/Max points, so we calculate center from those
	cropCenterX := (crop.Min.X + crop.Max.X) / 2
	cropCenterY := (crop.Min.Y + crop.Max.Y) / 2
	focalX := float64(cropCenterX) / float64(width)
	focalY := float64(cropCenterY) / float64(height)

	// Clamp values between 0 and 1
	if focalX < 0 {
		focalX = 0
	} else if focalX > 1 {
		focalX = 1
	}
	if focalY < 0 {
		focalY = 0
	} else if focalY > 1 {
		focalY = 1
	}

	return focalX, focalY, nil
}

// AnalyzeFocalPointFromURL fetches an image from a URL and analyzes it for focal point
func (c *Client) AnalyzeFocalPointFromURL(backdropPath string) (float64, float64, error) {
	if backdropPath == "" {
		return 0.5, 0.25, nil
	}
	imageURL := fmt.Sprintf("%s/w780%s", imageBaseURL, backdropPath)
	resp, err := c.httpClient.Get(imageURL)
	if err != nil {
		return 0.5, 0.25, nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0.5, 0.25, nil
	}
	img, _, err := image.Decode(resp.Body)
	if err != nil {
		return 0.5, 0.25, nil
	}
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	resizer := nfnt.NewDefaultResizer()
	analyzer := smartcrop.NewAnalyzer(resizer)
	cropWidth := width
	cropHeight := width
	if cropHeight > height {
		cropHeight = height
		cropWidth = height
	}
	crop, err := analyzer.FindBestCrop(img, cropWidth, cropHeight)
	if err != nil {
		return 0.5, 0.25, nil
	}
	cropCenterX := (crop.Min.X + crop.Max.X) / 2
	cropCenterY := (crop.Min.Y + crop.Max.Y) / 2
	focalX := float64(cropCenterX) / float64(width)
	focalY := float64(cropCenterY) / float64(height)
	if focalX < 0 {
		focalX = 0
	} else if focalX > 1 {
		focalX = 1
	}
	if focalY < 0 {
		focalY = 0
	} else if focalY > 1 {
		focalY = 1
	}
	// Bias Y towards top for movie/TV backdrops - faces are typically in upper third
	focalY = focalY*0.6 + 0.25*0.4
	return focalX, focalY, nil
}


// GetPersonDetails fetches detailed info about a person
func (c *Client) GetPersonDetails(personID int64) (*PersonDetails, error) {
	data, err := c.get(fmt.Sprintf("/person/%d", personID), nil)
	if err != nil {
		return nil, err
	}
	var result PersonDetails
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetPersonCombinedCredits fetches all movie and TV credits for a person
func (c *Client) GetPersonCombinedCredits(personID int64) (*PersonCombinedCredits, error) {
	data, err := c.get(fmt.Sprintf("/person/%d/combined_credits", personID), nil)
	if err != nil {
		return nil, err
	}
	var result PersonCombinedCredits
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Helper functions

func GenresToJSON(genres []Genre) string {
	names := make([]string, len(genres))
	for i, g := range genres {
		names[i] = g.Name
	}
	data, _ := json.Marshal(names)
	return string(data)
}

func CastToJSON(cast []CastMember, limit int) string {
	if limit > 0 && len(cast) > limit {
		cast = cast[:limit]
	}
	// Marshal cast directly to preserve TMDB's snake_case format
	data, _ := json.Marshal(cast)
	return string(data)
}

// CrewToJSON converts crew members to JSON, keeping key roles with photos
func CrewToJSON(crew []CrewMember, limit int) string {
	// Filter to important roles only
	importantJobs := map[string]bool{
		"Director":           true,
		"Writer":             true,
		"Screenplay":         true,
		"Story":              true,
		"Producer":           true,
		"Executive Producer": true,
		"Editor":             true,
		"Director of Photography": true,
		"Cinematographer":    true,
		"Original Music Composer": true,
		"Composer":           true,
		"Costume Design":     true,
		"Production Design":  true,
	}

	var filtered []CrewMember
	for _, c := range crew {
		if importantJobs[c.Job] {
			filtered = append(filtered, c)
		}
	}

	if limit > 0 && len(filtered) > limit {
		filtered = filtered[:limit]
	}

	data, _ := json.Marshal(filtered)
	return string(data)
}

func GetDirector(crew []CrewMember) string {
	for _, c := range crew {
		if c.Job == "Director" {
			return c.Name
		}
	}
	return ""
}

func GetWriter(crew []CrewMember) string {
	// Look for Screenplay first, then Writer
	for _, c := range crew {
		if c.Job == "Screenplay" {
			return c.Name
		}
	}
	for _, c := range crew {
		if c.Job == "Writer" {
			return c.Name
		}
	}
	return ""
}

func GetEditor(crew []CrewMember) string {
	for _, c := range crew {
		if c.Job == "Editor" {
			return c.Name
		}
	}
	return ""
}

func GetProducers(crew []CrewMember, limit int) string {
	var producers []string
	for _, c := range crew {
		if c.Job == "Producer" || c.Job == "Executive Producer" {
			producers = append(producers, c.Name)
			if limit > 0 && len(producers) >= limit {
				break
			}
		}
	}
	if len(producers) == 0 {
		return ""
	}
	return strings.Join(producers, ", ")
}

func TrailersToJSON(videos Videos) string {
	var trailers []map[string]string
	for _, v := range videos.Results {
		if v.Site == "YouTube" && (v.Type == "Trailer" || v.Type == "Teaser") {
			trailers = append(trailers, map[string]string{
				"key":  v.Key,
				"name": v.Name,
				"type": v.Type,
			})
		}
	}
	if len(trailers) == 0 {
		return ""
	}
	data, _ := json.Marshal(trailers)
	return string(data)
}

func GetYear(dateStr string) int {
	if len(dateStr) >= 4 {
		year, _ := strconv.Atoi(dateStr[:4])
		return year
	}
	return 0
}

// GetUSReleaseDates extracts theatrical and digital release dates for US
func GetUSReleaseDates(releaseDates ReleaseDatesResult) (theatrical, digital string) {
	for _, country := range releaseDates.Results {
		if country.ISO31661 == "US" {
			for _, rd := range country.ReleaseDates {
				// Type 3 = Theatrical
				if rd.Type == 3 && theatrical == "" {
					theatrical = rd.ReleaseDate
				}
				// Type 4 = Digital
				if rd.Type == 4 && digital == "" {
					digital = rd.ReleaseDate
				}
			}
			break
		}
	}
	return theatrical, digital
}

// GetStudios returns production company names as JSON array
func GetStudios(companies []ProductionCompany) string {
	if len(companies) == 0 {
		return ""
	}
	var names []string
	for _, c := range companies {
		names = append(names, c.Name)
	}
	data, _ := json.Marshal(names)
	return string(data)
}

// Discover types
type DiscoverResult struct {
	Page         int           `json:"page"`
	TotalPages   int           `json:"total_pages"`
	TotalResults int           `json:"total_results"`
}

type DiscoverMovieResult struct {
	DiscoverResult
	Results []MovieResult `json:"results"`
}

type DiscoverTVResult struct {
	DiscoverResult
	Results []TVResult `json:"results"`
}

// GetTrendingMovies returns trending movies for the week
func (c *Client) GetTrendingMovies(page int) (*DiscoverMovieResult, error) {
	params := map[string]string{}
	if page > 0 {
		params["page"] = strconv.Itoa(page)
	}

	data, err := c.get("/trending/movie/week", params)
	if err != nil {
		return nil, err
	}

	var result DiscoverMovieResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetTrendingTV returns trending TV shows for the week
func (c *Client) GetTrendingTV(page int) (*DiscoverTVResult, error) {
	params := map[string]string{}
	if page > 0 {
		params["page"] = strconv.Itoa(page)
	}

	data, err := c.get("/trending/tv/week", params)
	if err != nil {
		return nil, err
	}

	var result DiscoverTVResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetPopularMovies returns popular movies
func (c *Client) GetPopularMovies(page int) (*DiscoverMovieResult, error) {
	params := map[string]string{}
	if page > 0 {
		params["page"] = strconv.Itoa(page)
	}

	data, err := c.get("/movie/popular", params)
	if err != nil {
		return nil, err
	}

	var result DiscoverMovieResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetPopularTV returns popular TV shows
func (c *Client) GetPopularTV(page int) (*DiscoverTVResult, error) {
	params := map[string]string{}
	if page > 0 {
		params["page"] = strconv.Itoa(page)
	}

	data, err := c.get("/tv/popular", params)
	if err != nil {
		return nil, err
	}

	var result DiscoverTVResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetUpcomingMovies returns upcoming movies
func (c *Client) GetUpcomingMovies(page int) (*DiscoverMovieResult, error) {
	params := map[string]string{}
	if page > 0 {
		params["page"] = strconv.Itoa(page)
	}

	data, err := c.get("/movie/upcoming", params)
	if err != nil {
		return nil, err
	}

	var result DiscoverMovieResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// DiscoverTheatricalReleases returns upcoming movies using discover endpoint
func (c *Client) DiscoverTheatricalReleases(region string, page int) (*DiscoverMovieResult, error) {
	// Get today's date and 1 year out for a full year of upcoming releases
	today := time.Now().Format("2006-01-02")
	oneYearOut := time.Now().AddDate(1, 0, 0).Format("2006-01-02")

	// Filter by upcoming date range, sort by popularity
	// Popularity naturally surfaces major anticipated releases
	params := map[string]string{
		"primary_release_date.gte": today,
		"primary_release_date.lte": oneYearOut,
		"sort_by":                  "popularity.desc",
		"include_adult":            "false",
		"include_video":            "false",
	}
	if region != "" {
		params["region"] = region
	}
	if page > 0 {
		params["page"] = strconv.Itoa(page)
	}

	data, err := c.get("/discover/movie", params)
	if err != nil {
		return nil, err
	}

	var result DiscoverMovieResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// DiscoverUpcomingTV returns upcoming TV shows using discover endpoint
func (c *Client) DiscoverUpcomingTV(page int) (*DiscoverTVResult, error) {
	// Get today's date and 1 year out for upcoming shows
	today := time.Now().Format("2006-01-02")
	oneYearOut := time.Now().AddDate(1, 0, 0).Format("2006-01-02")

	params := map[string]string{
		"first_air_date.gte": today,
		"first_air_date.lte": oneYearOut,
		"sort_by":            "popularity.desc",
		"include_adult":      "false",
	}
	if page > 0 {
		params["page"] = strconv.Itoa(page)
	}

	data, err := c.get("/discover/tv", params)
	if err != nil {
		return nil, err
	}

	var result DiscoverTVResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetTopRatedMovies returns top rated movies
func (c *Client) GetTopRatedMovies(page int) (*DiscoverMovieResult, error) {
	params := map[string]string{}
	if page > 0 {
		params["page"] = strconv.Itoa(page)
	}

	data, err := c.get("/movie/top_rated", params)
	if err != nil {
		return nil, err
	}

	var result DiscoverMovieResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetTopRatedTV returns top rated TV shows
func (c *Client) GetTopRatedTV(page int) (*DiscoverTVResult, error) {
	params := map[string]string{}
	if page > 0 {
		params["page"] = strconv.Itoa(page)
	}

	data, err := c.get("/tv/top_rated", params)
	if err != nil {
		return nil, err
	}

	var result DiscoverTVResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GenreList represents the response from genre list endpoints
type GenreList struct {
	Genres []Genre `json:"genres"`
}

// GetMovieGenres returns the list of movie genres
func (c *Client) GetMovieGenres() ([]Genre, error) {
	data, err := c.get("/genre/movie/list", nil)
	if err != nil {
		return nil, err
	}

	var result GenreList
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result.Genres, nil
}

// GetTVGenres returns the list of TV genres
func (c *Client) GetTVGenres() ([]Genre, error) {
	data, err := c.get("/genre/tv/list", nil)
	if err != nil {
		return nil, err
	}

	var result GenreList
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result.Genres, nil
}

// GetMoviesByGenre returns movies filtered by genre
func (c *Client) GetMoviesByGenre(genreID int, page int) (*DiscoverMovieResult, error) {
	params := map[string]string{
		"with_genres":    strconv.Itoa(genreID),
		"sort_by":        "popularity.desc",
		"include_adult":  "false",
		"include_video":  "false",
	}
	if page > 0 {
		params["page"] = strconv.Itoa(page)
	}

	data, err := c.get("/discover/movie", params)
	if err != nil {
		return nil, err
	}

	var result DiscoverMovieResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetTVByGenre returns TV shows filtered by genre
func (c *Client) GetTVByGenre(genreID int, page int) (*DiscoverTVResult, error) {
	params := map[string]string{
		"with_genres":   strconv.Itoa(genreID),
		"sort_by":       "popularity.desc",
		"include_adult": "false",
	}
	if page > 0 {
		params["page"] = strconv.Itoa(page)
	}

	data, err := c.get("/discover/tv", params)
	if err != nil {
		return nil, err
	}

	var result DiscoverTVResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// DiscoverMoviesByGenres returns popular movies matching any of the given genre IDs
func (c *Client) DiscoverMoviesByGenres(genreIDs []int, page int) (*DiscoverMovieResult, error) {
	// Convert genre IDs to comma-separated string (OR logic in TMDB)
	genreStrs := make([]string, len(genreIDs))
	for i, id := range genreIDs {
		genreStrs[i] = strconv.Itoa(id)
	}

	params := map[string]string{
		"with_genres":      strings.Join(genreStrs, "|"), // | = OR, , = AND
		"sort_by":          "popularity.desc",
		"include_adult":    "false",
		"include_video":    "false",
		"vote_count.gte":   "100", // Only movies with decent vote count
		"vote_average.gte": "6.0", // Minimum rating
	}
	if page > 0 {
		params["page"] = strconv.Itoa(page)
	}

	data, err := c.get("/discover/movie", params)
	if err != nil {
		return nil, err
	}

	var result DiscoverMovieResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetGenreNameToIDMap returns a map of genre names to TMDB IDs
func (c *Client) GetGenreNameToIDMap() (map[string]int, error) {
	genres, err := c.GetMovieGenres()
	if err != nil {
		return nil, err
	}

	genreMap := make(map[string]int)
	for _, g := range genres {
		genreMap[g.Name] = g.ID
	}
	return genreMap, nil
}

// GetTVGenreNameToIDMap returns a map of TV genre names to TMDB IDs
func (c *Client) GetTVGenreNameToIDMap() (map[string]int, error) {
	genres, err := c.GetTVGenres()
	if err != nil {
		return nil, err
	}

	genreMap := make(map[string]int)
	for _, g := range genres {
		genreMap[g.Name] = g.ID
	}
	return genreMap, nil
}

// DiscoverTVByGenres returns popular TV shows matching any of the given genre IDs
func (c *Client) DiscoverTVByGenres(genreIDs []int, page int) (*DiscoverTVResult, error) {
	// Convert genre IDs to comma-separated string (OR logic in TMDB)
	genreStrs := make([]string, len(genreIDs))
	for i, id := range genreIDs {
		genreStrs[i] = strconv.Itoa(id)
	}

	params := map[string]string{
		"with_genres":      strings.Join(genreStrs, "|"), // | = OR, , = AND
		"sort_by":          "popularity.desc",
		"include_adult":    "false",
		"vote_count.gte":   "100", // Only shows with decent vote count
		"vote_average.gte": "6.0", // Minimum rating
	}
	if page > 0 {
		params["page"] = strconv.Itoa(page)
	}

	data, err := c.get("/discover/tv", params)
	if err != nil {
		return nil, err
	}

	var result DiscoverTVResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetTVRecommendations returns TV show recommendations for a given show
func (c *Client) GetTVRecommendations(tmdbID int64) (*DiscoverTVResult, error) {
	endpoint := fmt.Sprintf("/tv/%d/recommendations", tmdbID)
	data, err := c.get(endpoint, nil)
	if err != nil {
		return nil, err
	}

	var result DiscoverTVResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// EnrichedTVResult extends TVResult with episode runtime and content rating
type EnrichedTVResult struct {
	TVResult
	EpisodeRuntime int    `json:"episode_runtime"`
	ContentRating  string `json:"content_rating"`
}

// EnrichedTVSearchResult is like DiscoverTVResult but with enriched results
type EnrichedTVSearchResult struct {
	Page         int                `json:"page"`
	Results      []EnrichedTVResult `json:"results"`
	TotalPages   int                `json:"total_pages"`
	TotalResults int                `json:"total_results"`
}

// TVDetailsMinimal is a minimal response for getting episode runtime
type TVDetailsMinimal struct {
	ID             int64 `json:"id"`
	EpisodeRunTime []int `json:"episode_run_time"`
}

// GetEnrichedTVRecommendations gets recommendations with runtime and content rating
func (c *Client) GetEnrichedTVRecommendations(tmdbID int64, limit int) (*EnrichedTVSearchResult, error) {
	// First get basic recommendations
	basic, err := c.GetTVRecommendations(tmdbID)
	if err != nil {
		return nil, err
	}

	// Limit the number we enrich to avoid too many API calls
	if limit <= 0 || limit > len(basic.Results) {
		limit = len(basic.Results)
	}
	if limit > 20 {
		limit = 20 // Cap at 20 to avoid excessive API calls
	}

	enriched := &EnrichedTVSearchResult{
		Page:         basic.Page,
		TotalPages:   basic.TotalPages,
		TotalResults: basic.TotalResults,
		Results:      make([]EnrichedTVResult, 0, limit),
	}

	// Fetch details for each recommendation
	for i := 0; i < limit && i < len(basic.Results); i++ {
		rec := basic.Results[i]
		enrichedRec := EnrichedTVResult{TVResult: rec}

		// Get episode runtime from TV details
		data, err := c.get(fmt.Sprintf("/tv/%d", rec.ID), nil)
		if err == nil {
			var details TVDetailsMinimal
			if json.Unmarshal(data, &details) == nil && len(details.EpisodeRunTime) > 0 {
				enrichedRec.EpisodeRuntime = details.EpisodeRunTime[0]
			}
		}

		// Get content rating
		rating, err := c.GetTVContentRating(rec.ID)
		if err == nil {
			enrichedRec.ContentRating = rating
		}

		enriched.Results = append(enriched.Results, enrichedRec)
	}

	return enriched, nil
}

// GetCollectionDetails fetches full details of a TMDB collection including all parts
func (c *Client) GetCollectionDetails(collectionID int64) (*CollectionDetails, error) {
	data, err := c.get(fmt.Sprintf("/collection/%d", collectionID), nil)
	if err != nil {
		return nil, err
	}

	var details CollectionDetails
	if err := json.Unmarshal(data, &details); err != nil {
		return nil, err
	}

	return &details, nil
}
