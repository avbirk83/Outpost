package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// Streaming handlers

// VideoStream represents a video stream in a media file
type VideoStream struct {
	Index       int    `json:"index"`
	Codec       string `json:"codec"`
	Profile     string `json:"profile,omitempty"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	AspectRatio string `json:"aspectRatio,omitempty"`
	FrameRate   string `json:"frameRate,omitempty"`
	BitRate     int64  `json:"bitRate,omitempty"`
	PixelFormat string `json:"pixelFormat,omitempty"`
	Default     bool   `json:"default"`
}

// AudioStream represents an audio stream in a media file
type AudioStream struct {
	Index         int    `json:"index"`
	Codec         string `json:"codec"`
	Channels      int    `json:"channels"`
	ChannelLayout string `json:"channelLayout,omitempty"`
	SampleRate    int    `json:"sampleRate,omitempty"`
	BitRate       int64  `json:"bitRate,omitempty"`
	Language      string `json:"language,omitempty"`
	Title         string `json:"title,omitempty"`
	Default       bool   `json:"default"`
}

// handleStream handles streaming with transcoding support
func (s *Server) handleStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse path: /api/stream/{type}/{id}
	path := strings.TrimPrefix(r.URL.Path, "/api/stream/")
	parts := strings.Split(path, "/")

	if len(parts) != 2 {
		http.Error(w, "Invalid stream path", http.StatusBadRequest)
		return
	}

	mediaType := parts[0]
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var filePath string

	switch mediaType {
	case "movie":
		movie, err := s.db.GetMovie(id)
		if err != nil {
			http.Error(w, "Movie not found", http.StatusNotFound)
			return
		}
		filePath = movie.Path
	case "episode":
		episode, err := s.db.GetEpisode(id)
		if err != nil {
			http.Error(w, "Episode not found", http.StatusNotFound)
			return
		}
		filePath = episode.Path
	case "track":
		track, err := s.db.GetTrack(id)
		if err != nil {
			http.Error(w, "Track not found", http.StatusNotFound)
			return
		}
		filePath = track.Path
		// Serve audio files directly
		s.serveFileDirectly(w, r, filePath)
		return
	case "book":
		book, err := s.db.GetBook(id)
		if err != nil {
			http.Error(w, "Book not found", http.StatusNotFound)
			return
		}
		filePath = book.Path
		// Serve books directly
		s.serveFileDirectly(w, r, filePath)
		return
	default:
		http.Error(w, "Invalid media type", http.StatusBadRequest)
		return
	}

	// Check file exists
	if _, err := os.Stat(filePath); err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// Check if file is browser-compatible (direct play)
	ext := strings.ToLower(filepath.Ext(filePath))
	canDirectPlay := ext == ".mp4" || ext == ".webm" || ext == ".m4v"

	// Direct play for compatible files (browser handles seeking via Range requests)
	if canDirectPlay {
		s.serveFileDirectly(w, r, filePath)
		return
	}

	// Transcode for non-compatible files (MKV, AVI, etc.)
	s.serveTranscodedVideo(w, r, filePath)
}

// serveFileDirectly serves a file without transcoding
func (s *Server) serveFileDirectly(w http.ResponseWriter, r *http.Request, filePath string) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "Cannot open file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(filePath))
	contentType := "application/octet-stream"
	switch ext {
	case ".mp4":
		contentType = "video/mp4"
	case ".mkv":
		contentType = "video/x-matroska"
	case ".webm":
		contentType = "video/webm"
	case ".avi":
		contentType = "video/x-msvideo"
	case ".mov":
		contentType = "video/quicktime"
	case ".mp3":
		contentType = "audio/mpeg"
	case ".flac":
		contentType = "audio/flac"
	case ".m4a", ".aac":
		contentType = "audio/mp4"
	case ".ogg":
		contentType = "audio/ogg"
	case ".pdf":
		contentType = "application/pdf"
	case ".epub":
		contentType = "application/epub+zip"
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Accept-Ranges", "bytes")
	http.ServeContent(w, r, filepath.Base(filePath), fileInfo.ModTime(), file)
}

// handleMediaInfo returns media information including duration
func (s *Server) handleMediaInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse path: /api/media-info/{type}/{id}
	path := strings.TrimPrefix(r.URL.Path, "/api/media-info/")
	parts := strings.Split(path, "/")

	if len(parts) != 2 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	mediaType := parts[0]
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var filePath string

	switch mediaType {
	case "movie":
		movie, err := s.db.GetMovie(id)
		if err != nil {
			http.Error(w, "Movie not found", http.StatusNotFound)
			return
		}
		filePath = movie.Path
	case "episode":
		episode, err := s.db.GetEpisode(id)
		if err != nil {
			http.Error(w, "Episode not found", http.StatusNotFound)
			return
		}
		filePath = episode.Path
	default:
		http.Error(w, "Invalid media type", http.StatusBadRequest)
		return
	}

	// Get full media info using ffprobe
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filePath,
	)

	output, err := cmd.Output()
	if err != nil {
		http.Error(w, "Failed to get media info", http.StatusInternalServerError)
		return
	}

	// Parse ffprobe output
	var probeResult struct {
		Format struct {
			FormatName string `json:"format_name"`
			Duration   string `json:"duration"`
			Size       string `json:"size"`
			BitRate    string `json:"bit_rate"`
		} `json:"format"`
		Streams []struct {
			Index         int               `json:"index"`
			CodecType     string            `json:"codec_type"`
			CodecName     string            `json:"codec_name"`
			Profile       string            `json:"profile"`
			Width         int               `json:"width"`
			Height        int               `json:"height"`
			DisplayAspect string            `json:"display_aspect_ratio"`
			PixelFormat   string            `json:"pix_fmt"`
			FrameRate     string            `json:"r_frame_rate"`
			AvgFrameRate  string            `json:"avg_frame_rate"`
			BitRate       string            `json:"bit_rate"`
			Channels      int               `json:"channels"`
			ChannelLayout string            `json:"channel_layout"`
			SampleRate    string            `json:"sample_rate"`
			Tags          map[string]string `json:"tags"`
			Disposition   struct {
				Default int `json:"default"`
				Forced  int `json:"forced"`
			} `json:"disposition"`
		} `json:"streams"`
	}

	if err := json.Unmarshal(output, &probeResult); err != nil {
		http.Error(w, "Failed to parse media info", http.StatusInternalServerError)
		return
	}

	// Parse duration
	duration, _ := strconv.ParseFloat(probeResult.Format.Duration, 64)
	fileSize, _ := strconv.ParseInt(probeResult.Format.Size, 10, 64)
	containerBitRate, _ := strconv.ParseInt(probeResult.Format.BitRate, 10, 64)

	// Build response
	videoStreams := []VideoStream{}
	audioStreams := []AudioStream{}
	subtitleTracks := []SubtitleTrack{}
	subtitleIndex := 0

	for _, stream := range probeResult.Streams {
		switch stream.CodecType {
		case "video":
			bitRate, _ := strconv.ParseInt(stream.BitRate, 10, 64)
			// Calculate frame rate from fraction if available
			frameRate := stream.AvgFrameRate
			if frameRate == "" || frameRate == "0/0" {
				frameRate = stream.FrameRate
			}
			// Simplify frame rate (e.g., "24000/1001" -> "23.976")
			if parts := strings.Split(frameRate, "/"); len(parts) == 2 {
				num, _ := strconv.ParseFloat(parts[0], 64)
				den, _ := strconv.ParseFloat(parts[1], 64)
				if den > 0 {
					frameRate = fmt.Sprintf("%.3f", num/den)
				}
			}
			videoStreams = append(videoStreams, VideoStream{
				Index:       stream.Index,
				Codec:       stream.CodecName,
				Profile:     stream.Profile,
				Width:       stream.Width,
				Height:      stream.Height,
				AspectRatio: stream.DisplayAspect,
				FrameRate:   frameRate,
				BitRate:     bitRate,
				PixelFormat: stream.PixelFormat,
				Default:     stream.Disposition.Default == 1,
			})
		case "audio":
			bitRate, _ := strconv.ParseInt(stream.BitRate, 10, 64)
			sampleRate, _ := strconv.Atoi(stream.SampleRate)
			audioStreams = append(audioStreams, AudioStream{
				Index:         stream.Index,
				Codec:         stream.CodecName,
				Channels:      stream.Channels,
				ChannelLayout: stream.ChannelLayout,
				SampleRate:    sampleRate,
				BitRate:       bitRate,
				Language:      stream.Tags["language"],
				Title:         stream.Tags["title"],
				Default:       stream.Disposition.Default == 1,
			})
		case "subtitle":
			subtitleTracks = append(subtitleTracks, SubtitleTrack{
				Index:    subtitleIndex,
				Language: stream.Tags["language"],
				Title:    stream.Tags["title"],
				Codec:    stream.CodecName,
				Default:  stream.Disposition.Default == 1,
				Forced:   stream.Disposition.Forced == 1,
				External: false,
			})
			subtitleIndex++
		}
	}

	// Also get external subtitles
	externalTracks := s.findExternalSubtitles(filePath, subtitleIndex)
	subtitleTracks = append(subtitleTracks, externalTracks...)

	// Get container format (e.g., "matroska,webm" -> "MKV")
	container := probeResult.Format.FormatName
	containerMap := map[string]string{
		"matroska,webm":           "MKV",
		"mov,mp4,m4a,3gp,3g2,mj2": "MP4",
		"avi":                     "AVI",
		"mpegts":                  "TS",
		"webm":                    "WEBM",
	}
	if mapped, ok := containerMap[container]; ok {
		container = mapped
	} else {
		container = strings.ToUpper(strings.Split(container, ",")[0])
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"duration":       duration,
		"fileSize":       fileSize,
		"bitRate":        containerBitRate,
		"container":      container,
		"videoStreams":   videoStreams,
		"audioStreams":   audioStreams,
		"subtitleTracks": subtitleTracks,
	})
}

// serveTranscodedVideo transcodes video on-the-fly using FFmpeg
func (s *Server) serveTranscodedVideo(w http.ResponseWriter, r *http.Request, filePath string) {
	// Check for seek position (in seconds)
	startTime := r.URL.Query().Get("t")

	// Build FFmpeg arguments
	args := []string{}

	// Add seek position before input for fast initial seek
	if startTime != "" {
		args = append(args, "-ss", startTime)
	}

	args = append(args,
		"-i", filePath,
		"-c:v", "libx264",      // Re-encode video to ensure proper sync after seek
		"-preset", "ultrafast", // Fast encoding
		"-crf", "23",           // Quality level
		"-c:a", "aac",          // Transcode audio to AAC
		"-b:a", "192k",         // Audio bitrate
		"-ac", "2",             // Stereo audio
		"-movflags", "frag_keyframe+empty_moov+faststart",
		"-f", "mp4", // Output format
		"-",         // Output to stdout
	)

	cmd := exec.Command("ffmpeg", args...)

	// Get stdout pipe
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		http.Error(w, "Failed to create pipe", http.StatusInternalServerError)
		return
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		http.Error(w, "Failed to start transcoding", http.StatusInternalServerError)
		return
	}

	// Set headers for streaming
	w.Header().Set("Content-Type", "video/mp4")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("Cache-Control", "no-cache")

	// Stream the output
	buf := make([]byte, 32*1024) // 32KB buffer
	for {
		n, err := stdout.Read(buf)
		if n > 0 {
			if _, writeErr := w.Write(buf[:n]); writeErr != nil {
				cmd.Process.Kill()
				break
			}
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
		if err != nil {
			break
		}
	}

	cmd.Wait()
}
