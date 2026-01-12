package scanner

import (
	"bytes"
	"encoding/binary"
	"log"
	"os/exec"
	"strconv"
	"strings"

	"github.com/outpost/outpost/internal/database"
)

// IntroDetector handles audio fingerprinting and intro/credits detection
type IntroDetector struct {
	db *database.Database
}

// NewIntroDetector creates a new intro detector
func NewIntroDetector(db *database.Database) *IntroDetector {
	return &IntroDetector{db: db}
}

// ExtractFingerprint extracts an audio fingerprint from a video file using fpcalc
// Returns the raw fingerprint data and duration
func (d *IntroDetector) ExtractFingerprint(videoPath string, maxDuration float64) ([]uint32, float64, error) {
	// Use fpcalc to generate chromaprint fingerprint
	// -length limits duration, -raw outputs raw integers
	args := []string{
		"-length", strconv.FormatFloat(maxDuration, 'f', 0, 64),
		"-raw",
		videoPath,
	}

	cmd := exec.Command("fpcalc", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Printf("fpcalc error for %s: %s", videoPath, stderr.String())
		return nil, 0, err
	}

	// Parse fpcalc output format:
	// DURATION=123.45
	// FINGERPRINT=12345678,23456789,...
	output := stdout.String()
	lines := strings.Split(output, "\n")

	var duration float64
	var fingerprint []uint32

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "DURATION=") {
			durStr := strings.TrimPrefix(line, "DURATION=")
			duration, _ = strconv.ParseFloat(durStr, 64)
		} else if strings.HasPrefix(line, "FINGERPRINT=") {
			fpStr := strings.TrimPrefix(line, "FINGERPRINT=")
			parts := strings.Split(fpStr, ",")
			fingerprint = make([]uint32, 0, len(parts))
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p == "" {
					continue
				}
				// fpcalc outputs signed integers, parse as int64 then convert
				val, err := strconv.ParseInt(p, 10, 64)
				if err != nil {
					continue
				}
				fingerprint = append(fingerprint, uint32(val))
			}
		}
	}

	if len(fingerprint) == 0 {
		return nil, 0, nil
	}

	return fingerprint, duration, nil
}

// FingerprintToBytes converts uint32 fingerprint to bytes for storage
func FingerprintToBytes(fp []uint32) []byte {
	buf := make([]byte, len(fp)*4)
	for i, v := range fp {
		binary.LittleEndian.PutUint32(buf[i*4:], v)
	}
	return buf
}

// BytesToFingerprint converts stored bytes back to uint32 fingerprint
func BytesToFingerprint(data []byte) []uint32 {
	fp := make([]uint32, len(data)/4)
	for i := 0; i < len(fp); i++ {
		fp[i] = binary.LittleEndian.Uint32(data[i*4:])
	}
	return fp
}

// CompareFingerprints compares two fingerprints and returns similarity score (0-1)
// Uses hamming distance on chromaprint values
func CompareFingerprints(fp1, fp2 []uint32, offset1, offset2, windowSize int) float64 {
	if len(fp1) == 0 || len(fp2) == 0 {
		return 0
	}

	// Ensure we don't go out of bounds
	end1 := offset1 + windowSize
	end2 := offset2 + windowSize
	if end1 > len(fp1) || end2 > len(fp2) {
		return 0
	}

	// Calculate hamming distance for each pair
	totalBits := 0
	matchingBits := 0

	for i := 0; i < windowSize; i++ {
		v1 := fp1[offset1+i]
		v2 := fp2[offset2+i]

		// XOR gives us differing bits
		diff := v1 ^ v2

		// Count matching bits (32 - popcount of diff)
		popcount := bits32(diff)
		matchingBits += 32 - popcount
		totalBits += 32
	}

	if totalBits == 0 {
		return 0
	}

	return float64(matchingBits) / float64(totalBits)
}

// bits32 counts the number of set bits in a 32-bit integer
func bits32(n uint32) int {
	count := 0
	for n != 0 {
		count++
		n &= n - 1
	}
	return count
}

// SegmentMatch represents a detected matching segment between episodes
type SegmentMatch struct {
	StartSeconds1 float64
	StartSeconds2 float64
	EndSeconds1   float64
	EndSeconds2   float64
	Similarity    float64
}

// FindCommonSegments finds common audio segments between two fingerprints
// This is used to detect intros that appear in multiple episodes
func FindCommonSegments(fp1, fp2 []uint32, minSimilarity float64, minDurationSec float64) []SegmentMatch {
	var matches []SegmentMatch

	// Each fingerprint value represents ~0.125 seconds (8 values per second)
	valuesPerSecond := 8.0
	windowSizeValues := int(minDurationSec * valuesPerSecond) // Minimum segment size

	if windowSizeValues < 8 {
		windowSizeValues = 8 // At least 1 second
	}

	// Skip first 20 seconds (network logos, bumpers) - start at 20 seconds
	minOffset := int(20 * valuesPerSecond)

	// Only check first 5 minutes of each fingerprint for intros
	maxOffset := int(300 * valuesPerSecond) // 5 minutes
	if maxOffset > len(fp1)-windowSizeValues {
		maxOffset = len(fp1) - windowSizeValues
	}
	if maxOffset < minOffset {
		return matches
	}

	maxOffset2 := int(300 * valuesPerSecond)
	if maxOffset2 > len(fp2)-windowSizeValues {
		maxOffset2 = len(fp2) - windowSizeValues
	}
	if maxOffset2 < minOffset {
		return matches
	}

	// Sliding window comparison
	// Use smaller step for better accuracy (5 seconds = 40 fingerprint values)
	step := 40
	if step > windowSizeValues/4 {
		step = windowSizeValues / 4
	}
	if step < 8 {
		step = 8
	}

	bestMatch := SegmentMatch{}
	bestSimilarity := float64(0)
	bestDuration := float64(0)

	// Track top 5 matches for debugging
	type matchInfo struct {
		i, j int
		sim  float64
	}
	var topMatches []matchInfo

	log.Printf("FindCommonSegments: searching range %d-%d (%.1f-%.1f sec) with window %d (%.1f sec), step %d",
		minOffset, maxOffset, float64(minOffset)/valuesPerSecond, float64(maxOffset)/valuesPerSecond,
		windowSizeValues, float64(windowSizeValues)/valuesPerSecond, step)

	// Start from minOffset to skip network bumpers/logos at the beginning
	for i := minOffset; i < maxOffset; i += step {
		for j := minOffset; j < maxOffset2; j += step {
			sim := CompareFingerprints(fp1, fp2, i, j, windowSizeValues)

			// Track top matches for debugging
			if len(topMatches) < 5 || sim > topMatches[len(topMatches)-1].sim {
				topMatches = append(topMatches, matchInfo{i, j, sim})
				// Sort by similarity (descending)
				for k := len(topMatches) - 1; k > 0 && topMatches[k].sim > topMatches[k-1].sim; k-- {
					topMatches[k], topMatches[k-1] = topMatches[k-1], topMatches[k]
				}
				if len(topMatches) > 5 {
					topMatches = topMatches[:5]
				}
			}

			// Prefer matches where both segments are at similar positions (intros tend to be consistent)
			// Also prefer matches in the 60-240 second range where intros typically occur
			time1 := float64(i) / valuesPerSecond
			time2 := float64(j) / valuesPerSecond
			timeDiff := time1 - time2
			if timeDiff < 0 {
				timeDiff = -timeDiff
			}

			// Bonus for matches where both are in typical intro range (60-240 sec) and close in time
			inIntroRange := (time1 >= 60 && time1 <= 240) && (time2 >= 60 && time2 <= 240)
			closeInTime := timeDiff < 60 // Within 60 seconds of each other

			// Effective similarity includes position bonus
			effectiveSim := sim
			if inIntroRange && closeInTime {
				effectiveSim += 0.05 // 5% bonus for being in typical intro range and close in time
			} else if inIntroRange {
				effectiveSim += 0.02 // 2% bonus for just being in intro range
			}

			duration := float64(windowSizeValues) / valuesPerSecond
			if effectiveSim >= minSimilarity && (effectiveSim > bestSimilarity || (effectiveSim == bestSimilarity && duration > bestDuration)) {
				bestSimilarity = effectiveSim
				bestDuration = duration
				bestMatch = SegmentMatch{
					StartSeconds1: float64(i) / valuesPerSecond,
					StartSeconds2: float64(j) / valuesPerSecond,
					EndSeconds1:   float64(i+windowSizeValues) / valuesPerSecond,
					EndSeconds2:   float64(j+windowSizeValues) / valuesPerSecond,
					Similarity:    sim, // Store raw similarity for logging
				}
			}
		}
	}

	// Log top matches for debugging
	log.Printf("FindCommonSegments: top 5 matches (threshold %.2f):", minSimilarity)
	for _, m := range topMatches {
		log.Printf("  %.1f-%.1f sec <-> %.1f-%.1f sec: sim=%.3f",
			float64(m.i)/valuesPerSecond, float64(m.i+windowSizeValues)/valuesPerSecond,
			float64(m.j)/valuesPerSecond, float64(m.j+windowSizeValues)/valuesPerSecond, m.sim)
	}

	if bestSimilarity >= minSimilarity {
		// Refine the match by extending it
		refined := refineMatch(fp1, fp2, bestMatch, minSimilarity, valuesPerSecond)
		log.Printf("FindCommonSegments: refined match %.1f-%.1f <-> %.1f-%.1f",
			refined.StartSeconds1, refined.EndSeconds1, refined.StartSeconds2, refined.EndSeconds2)
		matches = append(matches, refined)
	} else {
		log.Printf("FindCommonSegments: no match above threshold (best was %.3f)", bestSimilarity)
	}

	return matches
}

// refineMatch extends a match to find the full extent of the similar segment
func refineMatch(fp1, fp2 []uint32, match SegmentMatch, minSimilarity float64, valuesPerSecond float64) SegmentMatch {
	// Convert to value indices
	start1 := int(match.StartSeconds1 * valuesPerSecond)
	start2 := int(match.StartSeconds2 * valuesPerSecond)
	end1 := int(match.EndSeconds1 * valuesPerSecond)
	end2 := int(match.EndSeconds2 * valuesPerSecond)

	// Try to extend backward
	for start1 > 0 && start2 > 0 {
		sim := CompareFingerprints(fp1, fp2, start1-1, start2-1, 8)
		if sim < minSimilarity*0.9 { // Allow slightly lower similarity at edges
			break
		}
		start1--
		start2--
	}

	// Try to extend forward
	for end1 < len(fp1)-8 && end2 < len(fp2)-8 {
		sim := CompareFingerprints(fp1, fp2, end1, end2, 8)
		if sim < minSimilarity*0.9 {
			break
		}
		end1++
		end2++
	}

	return SegmentMatch{
		StartSeconds1: float64(start1) / valuesPerSecond,
		StartSeconds2: float64(start2) / valuesPerSecond,
		EndSeconds1:   float64(end1) / valuesPerSecond,
		EndSeconds2:   float64(end2) / valuesPerSecond,
		Similarity:    match.Similarity,
	}
}

// DetectIntroForSeason analyzes all episodes in a season and detects common intro segments
func (d *IntroDetector) DetectIntroForSeason(seasonID int64) error {
	// Get all fingerprints for this season
	fingerprints, err := d.db.GetSeasonFingerprints(seasonID)
	if err != nil {
		return err
	}

	if len(fingerprints) < 2 {
		log.Printf("Season %d has fewer than 2 fingerprints, skipping intro detection", seasonID)
		return nil
	}

	log.Printf("Analyzing %d episodes for intro detection in season %d", len(fingerprints), seasonID)

	// Compare each pair of consecutive episodes
	minSimilarity := 0.65 // 65% similarity threshold - higher to reduce false positives
	minDuration := 25.0   // Minimum 25 second intro

	// Track detected intros for each episode
	introStarts := make(map[int64][]float64)
	introEnds := make(map[int64][]float64)

	for i := 0; i < len(fingerprints)-1; i++ {
		fp1 := BytesToFingerprint(fingerprints[i].Fingerprint)
		fp2 := BytesToFingerprint(fingerprints[i+1].Fingerprint)

		matches := FindCommonSegments(fp1, fp2, minSimilarity, minDuration)

		if len(matches) == 0 {
			log.Printf("No matching segments found between episode %d and %d", fingerprints[i].EpisodeID, fingerprints[i+1].EpisodeID)
		}

		for _, match := range matches {
			// Store detected intro times
			introStarts[fingerprints[i].EpisodeID] = append(introStarts[fingerprints[i].EpisodeID], match.StartSeconds1)
			introEnds[fingerprints[i].EpisodeID] = append(introEnds[fingerprints[i].EpisodeID], match.EndSeconds1)
			introStarts[fingerprints[i+1].EpisodeID] = append(introStarts[fingerprints[i+1].EpisodeID], match.StartSeconds2)
			introEnds[fingerprints[i+1].EpisodeID] = append(introEnds[fingerprints[i+1].EpisodeID], match.EndSeconds2)

			log.Printf("Found common segment: Episode %d (%.1f-%.1f) <-> Episode %d (%.1f-%.1f), similarity: %.2f",
				fingerprints[i].EpisodeID, match.StartSeconds1, match.EndSeconds1,
				fingerprints[i+1].EpisodeID, match.StartSeconds2, match.EndSeconds2,
				match.Similarity)
		}
	}

	// For each episode with detected intros, save the median intro time
	for episodeID, starts := range introStarts {
		if len(starts) == 0 {
			continue
		}

		ends := introEnds[episodeID]
		if len(ends) == 0 {
			continue
		}

		// Use median start/end times
		medianStart := median(starts)
		medianEnd := median(ends)

		// Skip if intro is too short or too long
		duration := medianEnd - medianStart
		if duration < 10 || duration > 180 {
			continue
		}

		// Create media segment
		segment := &database.MediaSegment{
			EpisodeID:    episodeID,
			SegmentType:  "intro",
			StartSeconds: medianStart,
			EndSeconds:   medianEnd,
			Confidence:   0.8, // Fingerprint-based detection confidence
			Source:       "fingerprint",
		}

		if err := d.db.CreateMediaSegment(segment); err != nil {
			log.Printf("Failed to save intro segment for episode %d: %v", episodeID, err)
		} else {
			log.Printf("Saved intro segment for episode %d: %.1f-%.1f", episodeID, medianStart, medianEnd)
		}
	}

	return nil
}

// median calculates the median of a slice of float64
func median(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	if len(values) == 1 {
		return values[0]
	}

	// Simple average for small slices
	sum := float64(0)
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// AnalyzeEpisode extracts and saves the fingerprint for a single episode
func (d *IntroDetector) AnalyzeEpisode(episode *database.Episode) error {
	if episode.Path == "" {
		return nil
	}

	log.Printf("Extracting fingerprint for episode %d: %s", episode.ID, episode.Path)

	// Extract first 5 minutes for intro detection
	fp, duration, err := d.ExtractFingerprint(episode.Path, 300)
	if err != nil {
		return err
	}

	if len(fp) == 0 {
		log.Printf("No fingerprint data extracted for episode %d", episode.ID)
		return nil
	}

	// Save fingerprint
	audioFp := &database.AudioFingerprint{
		EpisodeID:   episode.ID,
		Fingerprint: FingerprintToBytes(fp),
		Duration:    duration,
	}

	return d.db.SaveAudioFingerprint(audioFp)
}

// AnalyzeSeason analyzes all episodes in a season
func (d *IntroDetector) AnalyzeSeason(seasonID int64) error {
	// Get episodes without fingerprints
	episodes, err := d.db.GetEpisodesWithoutFingerprints(seasonID, 100)
	if err != nil {
		return err
	}

	// Extract fingerprints for each episode
	for _, ep := range episodes {
		if err := d.AnalyzeEpisode(&ep); err != nil {
			log.Printf("Failed to analyze episode %d: %v", ep.ID, err)
			continue
		}
	}

	// Now detect intros using the fingerprints
	return d.DetectIntroForSeason(seasonID)
}

// CheckFFmpegChromaprint checks if fpcalc (chromaprint) is available
func CheckFFmpegChromaprint() bool {
	cmd := exec.Command("fpcalc", "-v")
	err := cmd.Run()
	return err == nil
}
