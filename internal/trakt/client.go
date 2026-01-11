package trakt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	BaseURL     = "https://api.trakt.tv"
	AuthURL     = "https://trakt.tv/oauth/authorize"
	TokenURL    = "https://api.trakt.tv/oauth/token"
	APIVersion  = "2"
	ContentType = "application/json"
)

// Client handles Trakt API requests
type Client struct {
	ClientID     string
	ClientSecret string
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
	httpClient   *http.Client
}

// NewClient creates a new Trakt client
func NewClient(clientID, clientSecret string) *Client {
	return &Client{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
	}
}

// SetTokens sets the OAuth tokens
func (c *Client) SetTokens(accessToken, refreshToken string, expiresAt time.Time) {
	c.AccessToken = accessToken
	c.RefreshToken = refreshToken
	c.ExpiresAt = expiresAt
}

// TokenResponse represents OAuth token response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
	CreatedAt    int64  `json:"created_at"`
}

// GetAuthURL returns the OAuth authorization URL
func (c *Client) GetAuthURL(redirectURI string) string {
	params := url.Values{}
	params.Set("response_type", "code")
	params.Set("client_id", c.ClientID)
	params.Set("redirect_uri", redirectURI)
	return fmt.Sprintf("%s?%s", AuthURL, params.Encode())
}

// ExchangeCode exchanges an authorization code for tokens
func (c *Client) ExchangeCode(code, redirectURI string) (*TokenResponse, error) {
	body := map[string]string{
		"code":          code,
		"client_id":     c.ClientID,
		"client_secret": c.ClientSecret,
		"redirect_uri":  redirectURI,
		"grant_type":    "authorization_code",
	}

	jsonBody, _ := json.Marshal(body)
	req, err := http.NewRequest("POST", TokenURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", ContentType)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token exchange failed: %s", string(respBody))
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	// Set tokens on client
	expiresAt := time.Unix(tokenResp.CreatedAt, 0).Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	c.SetTokens(tokenResp.AccessToken, tokenResp.RefreshToken, expiresAt)

	return &tokenResp, nil
}

// RefreshAccessToken refreshes the access token using the refresh token
func (c *Client) RefreshAccessToken() (*TokenResponse, error) {
	if c.RefreshToken == "" {
		return nil, fmt.Errorf("no refresh token available")
	}

	body := map[string]string{
		"refresh_token": c.RefreshToken,
		"client_id":     c.ClientID,
		"client_secret": c.ClientSecret,
		"grant_type":    "refresh_token",
	}

	jsonBody, _ := json.Marshal(body)
	req, err := http.NewRequest("POST", TokenURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", ContentType)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token refresh failed: %s", string(respBody))
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	expiresAt := time.Unix(tokenResp.CreatedAt, 0).Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	c.SetTokens(tokenResp.AccessToken, tokenResp.RefreshToken, expiresAt)

	return &tokenResp, nil
}

// NeedsRefresh checks if the token needs to be refreshed
func (c *Client) NeedsRefresh() bool {
	// Refresh if token expires within 1 hour
	return time.Now().Add(time.Hour).After(c.ExpiresAt)
}

// doRequest performs an authenticated API request
func (c *Client) doRequest(method, endpoint string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, BaseURL+endpoint, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", ContentType)
	req.Header.Set("trakt-api-version", APIVersion)
	req.Header.Set("trakt-api-key", c.ClientID)
	if c.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	}

	return c.httpClient.Do(req)
}

// User represents a Trakt user
type User struct {
	Username string `json:"username"`
	Private  bool   `json:"private"`
	Name     string `json:"name"`
	VIP      bool   `json:"vip"`
	IDs      struct {
		Slug string `json:"slug"`
	} `json:"ids"`
}

// UserSettings represents user settings response
type UserSettings struct {
	User User `json:"user"`
}

// GetUserSettings gets the authenticated user's settings
func (c *Client) GetUserSettings() (*UserSettings, error) {
	resp, err := c.doRequest("GET", "/users/settings", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get user settings: %s", string(respBody))
	}

	var settings UserSettings
	if err := json.NewDecoder(resp.Body).Decode(&settings); err != nil {
		return nil, err
	}

	return &settings, nil
}

// IDs represents media identifiers
type IDs struct {
	Trakt  int    `json:"trakt,omitempty"`
	Slug   string `json:"slug,omitempty"`
	IMDB   string `json:"imdb,omitempty"`
	TMDB   int    `json:"tmdb,omitempty"`
	TVDB   int    `json:"tvdb,omitempty"`
}

// Movie represents a Trakt movie
type Movie struct {
	Title string `json:"title"`
	Year  int    `json:"year"`
	IDs   IDs    `json:"ids"`
}

// Show represents a Trakt show
type Show struct {
	Title string `json:"title"`
	Year  int    `json:"year"`
	IDs   IDs    `json:"ids"`
}

// Episode represents a Trakt episode
type Episode struct {
	Season int    `json:"season"`
	Number int    `json:"number"`
	Title  string `json:"title,omitempty"`
	IDs    IDs    `json:"ids,omitempty"`
}

// WatchedMovie represents a watched movie from history
type WatchedMovie struct {
	Plays         int       `json:"plays"`
	LastWatchedAt time.Time `json:"last_watched_at"`
	LastUpdatedAt time.Time `json:"last_updated_at"`
	Movie         Movie     `json:"movie"`
}

// WatchedShow represents a watched show from history
type WatchedShow struct {
	Plays         int       `json:"plays"`
	LastWatchedAt time.Time `json:"last_watched_at"`
	LastUpdatedAt time.Time `json:"last_updated_at"`
	Show          Show      `json:"show"`
	Seasons       []struct {
		Number   int `json:"number"`
		Episodes []struct {
			Number        int       `json:"number"`
			Plays         int       `json:"plays"`
			LastWatchedAt time.Time `json:"last_watched_at"`
		} `json:"episodes"`
	} `json:"seasons"`
}

// GetWatchedMovies gets the user's watched movies
func (c *Client) GetWatchedMovies() ([]WatchedMovie, error) {
	resp, err := c.doRequest("GET", "/sync/watched/movies", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get watched movies: %s", string(respBody))
	}

	var movies []WatchedMovie
	if err := json.NewDecoder(resp.Body).Decode(&movies); err != nil {
		return nil, err
	}

	return movies, nil
}

// GetWatchedShows gets the user's watched shows
func (c *Client) GetWatchedShows() ([]WatchedShow, error) {
	resp, err := c.doRequest("GET", "/sync/watched/shows", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get watched shows: %s", string(respBody))
	}

	var shows []WatchedShow
	if err := json.NewDecoder(resp.Body).Decode(&shows); err != nil {
		return nil, err
	}

	return shows, nil
}

// HistoryItem represents an item to add to history
type HistoryItem struct {
	WatchedAt time.Time `json:"watched_at,omitempty"`
	Movie     *Movie    `json:"movie,omitempty"`
	Show      *Show     `json:"show,omitempty"`
	Episode   *Episode  `json:"episode,omitempty"`
}

// HistoryRequest represents a request to add items to history
type HistoryRequest struct {
	Movies   []HistoryItem `json:"movies,omitempty"`
	Shows    []HistoryItem `json:"shows,omitempty"`
	Episodes []HistoryItem `json:"episodes,omitempty"`
}

// HistoryResponse represents the response from adding history
type HistoryResponse struct {
	Added struct {
		Movies   int `json:"movies"`
		Episodes int `json:"episodes"`
	} `json:"added"`
	NotFound struct {
		Movies   []Movie   `json:"movies"`
		Shows    []Show    `json:"shows"`
		Episodes []Episode `json:"episodes"`
	} `json:"not_found"`
}

// AddToHistory adds items to the user's watch history
func (c *Client) AddToHistory(req *HistoryRequest) (*HistoryResponse, error) {
	resp, err := c.doRequest("POST", "/sync/history", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to add to history: %s", string(respBody))
	}

	var histResp HistoryResponse
	if err := json.NewDecoder(resp.Body).Decode(&histResp); err != nil {
		return nil, err
	}

	return &histResp, nil
}

// Rating represents a Trakt rating
type Rating struct {
	RatedAt time.Time `json:"rated_at"`
	Rating  int       `json:"rating"`
	Type    string    `json:"type"`
	Movie   *Movie    `json:"movie,omitempty"`
	Show    *Show     `json:"show,omitempty"`
	Episode *Episode  `json:"episode,omitempty"`
}

// GetRatings gets the user's ratings
func (c *Client) GetRatings(mediaType string) ([]Rating, error) {
	endpoint := "/sync/ratings"
	if mediaType != "" {
		endpoint += "/" + mediaType
	}

	resp, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get ratings: %s", string(respBody))
	}

	var ratings []Rating
	if err := json.NewDecoder(resp.Body).Decode(&ratings); err != nil {
		return nil, err
	}

	return ratings, nil
}

// RatingRequest represents a request to add ratings
type RatingRequest struct {
	Movies   []RatingItem `json:"movies,omitempty"`
	Shows    []RatingItem `json:"shows,omitempty"`
	Episodes []RatingItem `json:"episodes,omitempty"`
}

// RatingItem represents an item with a rating
type RatingItem struct {
	RatedAt time.Time `json:"rated_at,omitempty"`
	Rating  int       `json:"rating"`
	Movie   *Movie    `json:"movie,omitempty"`
	Show    *Show     `json:"show,omitempty"`
	Episode *Episode  `json:"episode,omitempty"`
}

// RatingResponse represents the response from adding ratings
type RatingResponse struct {
	Added struct {
		Movies   int `json:"movies"`
		Shows    int `json:"shows"`
		Episodes int `json:"episodes"`
	} `json:"added"`
	NotFound struct {
		Movies   []Movie   `json:"movies"`
		Shows    []Show    `json:"shows"`
		Episodes []Episode `json:"episodes"`
	} `json:"not_found"`
}

// AddRatings adds ratings for items
func (c *Client) AddRatings(req *RatingRequest) (*RatingResponse, error) {
	resp, err := c.doRequest("POST", "/sync/ratings", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to add ratings: %s", string(respBody))
	}

	var ratingResp RatingResponse
	if err := json.NewDecoder(resp.Body).Decode(&ratingResp); err != nil {
		return nil, err
	}

	return &ratingResp, nil
}

// WatchlistItem represents a watchlist item
type WatchlistItem struct {
	Rank      int       `json:"rank"`
	ListedAt  time.Time `json:"listed_at"`
	Type      string    `json:"type"`
	Movie     *Movie    `json:"movie,omitempty"`
	Show      *Show     `json:"show,omitempty"`
}

// GetWatchlist gets the user's watchlist
func (c *Client) GetWatchlist(mediaType string) ([]WatchlistItem, error) {
	endpoint := "/sync/watchlist"
	if mediaType != "" {
		endpoint += "/" + mediaType
	}

	resp, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get watchlist: %s", string(respBody))
	}

	var items []WatchlistItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, err
	}

	return items, nil
}

// WatchlistRequest represents a request to add to watchlist
type WatchlistRequest struct {
	Movies []Movie `json:"movies,omitempty"`
	Shows  []Show  `json:"shows,omitempty"`
}

// WatchlistResponse represents the response from adding to watchlist
type WatchlistResponse struct {
	Added struct {
		Movies int `json:"movies"`
		Shows  int `json:"shows"`
	} `json:"added"`
	Existing struct {
		Movies int `json:"movies"`
		Shows  int `json:"shows"`
	} `json:"existing"`
	NotFound struct {
		Movies []Movie `json:"movies"`
		Shows  []Show  `json:"shows"`
	} `json:"not_found"`
}

// AddToWatchlist adds items to the user's watchlist
func (c *Client) AddToWatchlist(req *WatchlistRequest) (*WatchlistResponse, error) {
	resp, err := c.doRequest("POST", "/sync/watchlist", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to add to watchlist: %s", string(respBody))
	}

	var watchlistResp WatchlistResponse
	if err := json.NewDecoder(resp.Body).Decode(&watchlistResp); err != nil {
		return nil, err
	}

	return &watchlistResp, nil
}

// Test tests the API connection
func (c *Client) Test() error {
	if c.AccessToken == "" {
		return fmt.Errorf("not authenticated")
	}

	_, err := c.GetUserSettings()
	return err
}
