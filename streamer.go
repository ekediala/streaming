package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

const (
	DEFAULT_CHUNK = 1024*512 - 1 // Default 500kB chunk
)

type Streamer interface {
	Seek(fileName string, start, end int) (reader io.Reader, n int, error error)
}

type VideoStreamer struct {
	store map[string][]byte
}

func (vs *VideoStreamer) Seek(fileName string, start, end int) (io.Reader, int, error) {
	if record, ok := vs.store[fileName]; ok {
		if start < 0 || start >= len(record) {
			return nil, 0, errors.New("out of range")
		}

		if end < start || end >= len(record) {
			end = len(record) - 1
		}

		r := bytes.NewReader(record[start : end+1])

		return r, r.Len(), nil

	}

	return nil, 0, errors.New("data not found")
}

func NewVideoStreamer(store map[string][]byte) *VideoStreamer {
	return &VideoStreamer{store}
}

func VideoStreamHandler(streamer *VideoStreamer, videoKey string, totalSize int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var start, end int

		rangeHeader := r.Header.Get("Range")
		if rangeHeader == "" {
			start = 0
			end = DEFAULT_CHUNK
		} else {
			// Parse the Range header: "bytes=start-end"
			rangeParts := strings.TrimPrefix(rangeHeader, "bytes=")
			rangeValues := strings.Split(rangeParts, "-")
			var err error

			// Get start byte
			start, err = strconv.Atoi(rangeValues[0])
			if err != nil {
				http.Error(w, "Invalid start byte", http.StatusBadRequest)
				return
			}

			// Get end byte or set to default i
			if len(rangeValues) > 1 && rangeValues[1] != "" {
				end, err = strconv.Atoi(rangeValues[1])
				if err != nil {
					http.Error(w, "Invalid end byte", http.StatusBadRequest)
					return
				}
			} else {
				end = start + DEFAULT_CHUNK
			}
		}

		// Ensure end is within the total video size
		if end >= totalSize {
			end = totalSize - 1
		}

		data, n, err := streamer.Seek(videoKey, start, end)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Set headers and serve the video chunk
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, totalSize))
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", n))
		w.Header().Set("Content-Type", "video/mp4")
		w.WriteHeader(http.StatusPartialContent)

		_, err = io.Copy(w, data)
		if err != nil {
			http.Error(w, "Error streaming video", http.StatusInternalServerError)
		}
	}
}
