package importpkg

import (
	"database/sql"
	"path/filepath"
	"strings"

	"github.com/outpost/outpost/internal/parser"
)

// EpisodeMatch represents a matched file to episode
type EpisodeMatch struct {
	FilePath   string
	ShowID     int64
	SeasonID   int64
	EpisodeID  int64
	Season     int
	Episode    int
	Confidence float64 // 0.0 to 1.0
	ParsedInfo *parser.ParsedRelease
}

// EpisodeMatcher matches video files to database episodes
type EpisodeMatcher struct {
	db *sql.DB
}

// NewEpisodeMatcher creates a new episode matcher
func NewEpisodeMatcher(db *sql.DB) *EpisodeMatcher {
	return &EpisodeMatcher{db: db}
}

// MatchFiles attempts to match video files to episodes for a show
func (m *EpisodeMatcher) MatchFiles(showID int64, files []string) ([]EpisodeMatch, error) {
	// Get all seasons and episodes for this show
	episodes, err := m.getShowEpisodes(showID)
	if err != nil {
		return nil, err
	}

	var matches []EpisodeMatch

	for _, file := range files {
		// Parse the filename
		parsed := parser.Parse(filepath.Base(file))

		// Try to match to an episode
		match := m.matchFile(showID, file, parsed, episodes)
		if match != nil {
			matches = append(matches, *match)
		}
	}

	return matches, nil
}

// MatchSingleFile matches a single file to an episode
func (m *EpisodeMatcher) MatchSingleFile(showID int64, file string, parsed *parser.ParsedRelease) (*EpisodeMatch, error) {
	episodes, err := m.getShowEpisodes(showID)
	if err != nil {
		return nil, err
	}

	return m.matchFile(showID, file, parsed, episodes), nil
}

// episodeInfo holds episode data from database
type episodeInfo struct {
	EpisodeID     int64
	SeasonID      int64
	SeasonNumber  int
	EpisodeNumber int
	Title         string
}

// getShowEpisodes retrieves all episodes for a show
func (m *EpisodeMatcher) getShowEpisodes(showID int64) ([]episodeInfo, error) {
	rows, err := m.db.Query(`
		SELECT e.id, e.season_id, s.season_number, e.episode_number, e.title
		FROM episodes e
		JOIN seasons s ON s.id = e.season_id
		WHERE s.show_id = ?
		ORDER BY s.season_number, e.episode_number`, showID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var episodes []episodeInfo
	for rows.Next() {
		var ep episodeInfo
		if err := rows.Scan(&ep.EpisodeID, &ep.SeasonID, &ep.SeasonNumber, &ep.EpisodeNumber, &ep.Title); err != nil {
			return nil, err
		}
		episodes = append(episodes, ep)
	}

	return episodes, nil
}

// matchFile matches a single file to an episode
func (m *EpisodeMatcher) matchFile(showID int64, file string, parsed *parser.ParsedRelease, episodes []episodeInfo) *EpisodeMatch {
	// If parser extracted season/episode, use that
	if parsed.Season > 0 && parsed.Episode > 0 {
		for _, ep := range episodes {
			if ep.SeasonNumber == parsed.Season && ep.EpisodeNumber == parsed.Episode {
				return &EpisodeMatch{
					FilePath:   file,
					ShowID:     showID,
					SeasonID:   ep.SeasonID,
					EpisodeID:  ep.EpisodeID,
					Season:     ep.SeasonNumber,
					Episode:    ep.EpisodeNumber,
					Confidence: 0.95,
					ParsedInfo: parsed,
				}
			}
		}
	}

	// Try to match by episode title in filename
	filename := strings.ToLower(filepath.Base(file))
	for _, ep := range episodes {
		if ep.Title != "" && strings.Contains(filename, strings.ToLower(ep.Title)) {
			return &EpisodeMatch{
				FilePath:   file,
				ShowID:     showID,
				SeasonID:   ep.SeasonID,
				EpisodeID:  ep.EpisodeID,
				Season:     ep.SeasonNumber,
				Episode:    ep.EpisodeNumber,
				Confidence: 0.7,
				ParsedInfo: parsed,
			}
		}
	}

	return nil
}

// MatchSeasonPack matches all files from a season pack
func (m *EpisodeMatcher) MatchSeasonPack(showID int64, seasonNum int, files []string) ([]EpisodeMatch, []string, error) {
	episodes, err := m.getShowEpisodes(showID)
	if err != nil {
		return nil, nil, err
	}

	// Filter to just this season
	var seasonEps []episodeInfo
	for _, ep := range episodes {
		if ep.SeasonNumber == seasonNum {
			seasonEps = append(seasonEps, ep)
		}
	}

	var matches []EpisodeMatch
	var unmatched []string

	for _, file := range files {
		parsed := parser.Parse(filepath.Base(file))
		match := m.matchFile(showID, file, parsed, seasonEps)
		if match != nil {
			matches = append(matches, *match)
		} else {
			unmatched = append(unmatched, file)
		}
	}

	return matches, unmatched, nil
}

// IsSeasonPack checks if a parsed release is a season pack
func IsSeasonPack(parsed *parser.ParsedRelease) bool {
	return parsed.IsSeasonPack || (parsed.Season > 0 && parsed.Episode == 0)
}

// GetSeasonPackFiles returns all video files grouped by episode
func (m *EpisodeMatcher) GetSeasonPackFiles(seasonPath string) (map[int]string, error) {
	files, err := findVideoFiles(seasonPath)
	if err != nil {
		return nil, err
	}

	result := make(map[int]string)
	for _, file := range files {
		parsed := parser.Parse(filepath.Base(file))
		if parsed.Episode > 0 {
			result[parsed.Episode] = file
		}
	}

	return result, nil
}
