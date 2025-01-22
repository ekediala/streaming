package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"
)

const url = "https://download.samplelib.com/mp4/sample-5s.mp4"

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	video, err := downloadBytes(ctx, &http.Client{})
	if err != nil {
		log.Fatal(err)
	}

	streamer := NewVideoStreamer(map[string][]byte{url: video})

	mux := http.NewServeMux()
	mux.Handle("/video", VideoStreamHandler(streamer, url, len(video)))

	server := http.Server{
		Addr:    ":8888",
		Handler: mux,
	}

	go func() {
		err = server.ListenAndServe()
		if err != nil {
			log.Fatal(err)
		}
	}()

	slog.Info("streamer", "message", "server started on port 8888")

	<-ctx.Done()
	ctx, stop := context.WithTimeout(context.Background(), time.Second*30)
	defer stop()
	err = server.Shutdown(ctx)
	if err != nil {
		log.Fatal(err)
	}
}

func downloadBytes(ctx context.Context, client *http.Client) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("downloading bytes: %w", err)
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("downloading bytes: %w", err)
	}

	defer res.Body.Close()

	var buf bytes.Buffer
	_, err = io.Copy(&buf, res.Body)
	if err != nil {
		return nil, fmt.Errorf("downloading bytes: %w", err)
	}

	return buf.Bytes(), err
}
