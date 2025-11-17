package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/spf13/cobra"
	"github.com/wolveix/openxbl-go"
)

func newSyncCMD() *cobra.Command {
	syncClipsCMD := newSyncClipsCMD()
	syncScreenshotsCMD := newSyncScreenshotsCMD()

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync your latest DVR clips and screenshots",
		Run: func(command *cobra.Command, args []string) {
			syncClipsCMD.Run(command, args)
			syncScreenshotsCMD.Run(command, args)
		},
	}

	cmd.AddCommand(syncClipsCMD)
	cmd.AddCommand(syncScreenshotsCMD)

	return cmd
}

func newSyncClipsCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clips",
		Short: "Sync your latest DVR clips",
		Run: func(_ *cobra.Command, _ []string) {
			if cfg.APIKey == "" {
				log.Fatal().Msg("API key is required, set it with `xdvr config set apiKey your-api-key`")
			}

			client := openxbl.NewClient(cfg.APIKey, timeout)
			ctx := context.Background()
			httpClient := &http.Client{Timeout: timeout}

			var continuationToken string

			for {
				log.Info().Msgf("Finding DVR clips")

				clips, newContinuationToken, err := client.GetDVRClips(ctx, continuationToken)
				if err != nil {
					if err.Error() == "find clips" {
						log.Info().Msgf("No new clips to download")
						return
					}

					log.Fatal().Err(err).Msg("Failed to retrieve clips")
				}

				continuationToken = newContinuationToken

				for _, clip := range clips {
					downloadLink := clip.GetDownloadLink()
					if downloadLink == "" {
						continue
					}

					if err = processDVR(ctx, client, httpClient, clip.DVRCapture); err != nil {
						log.Error().Err(err).Msgf("Failed to process clip: %s", downloadLink)
					}
				}

				if continuationToken == "" {
					break
				}
			}
		},
	}

	return cmd
}

func newSyncScreenshotsCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "screenshots",
		Short: "Sync your latest DVR screenshots",
		Run: func(_ *cobra.Command, _ []string) {
			if cfg.APIKey == "" {
				log.Fatal().Msg("API key is required, set it with `xdvr config set apiKey your-api-key`")
			}

			if cfg.AutoDelete {
				log.Warn().Msg("Auto delete enabled, but screenshots can't be automatically deleted")
			}

			client := openxbl.NewClient(cfg.APIKey, timeout)
			ctx := context.Background()
			httpClient := &http.Client{Timeout: timeout}

			var continuationToken string

			for {
				log.Info().Msgf("Finding DVR screenshots")

				screenshots, newContinuationToken, err := client.GetDVRScreenshots(ctx, continuationToken)
				if err != nil {
					if err.Error() == "find screenshots" {
						log.Info().Msgf("No new screenshots to download")
						return
					}

					log.Fatal().Err(err).Msg("Failed to retrieve screenshots")
				}

				continuationToken = newContinuationToken

				for _, screenshot := range screenshots {
					downloadLink := screenshot.GetDownloadLink()
					if downloadLink == "" {
						continue
					}

					if err = processDVR(ctx, client, httpClient, screenshot.DVRCapture); err != nil {
						log.Error().Err(err).Msgf("Failed to process screenshot: %s", downloadLink)
					}
				}

				if continuationToken == "" {
					break
				}
			}
		},
	}

	return cmd
}

func processDVR(ctx context.Context, client *openxbl.Client, httpClient *http.Client, capture openxbl.DVRCapture) error {
	downloadURL := capture.GetDownloadLink()
	if downloadURL == "" {
		return nil
	}

	gameTitle := sanitizePath(capture.TitleName)
	gamePath := filepath.Join(cfg.SavePath, strings.ToLower(string(capture.Type))+"s", gameTitle)
	contentPath := filepath.Join(gamePath, fmt.Sprintf("%s - %s", gameTitle, capture.UploadDate.Format("2006-01-02 15_04_05")))

	if capture.Type == openxbl.DVRCaptureTypeClip {
		contentPath += ".mp4"
	} else {
		contentPath += ".png"
	}

	// Create dir for title.
	if err := os.MkdirAll(gamePath, os.ModePerm); err != nil {
		return fmt.Errorf("create save directory for %s: %w", capture.TitleName, err)
	}

	if _, err := os.Stat(contentPath); err == nil {
		log.Info().Msgf("Skipping %s (already downloaded)", contentPath)
		return nil
	}

	log.Info().Msgf("Downloading %s", contentPath)

	// Download the file.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download file: %s", downloadURL)
	}

	// Read the response body.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	// Write the response body to disk.
	if err = os.WriteFile(contentPath, body, 0o644); err != nil {
		return fmt.Errorf("write file to disk: %w", err)
	}

	if cfg.AutoDelete && capture.Type == openxbl.DVRCaptureTypeClip {
		log.Info().Msg("Deleting clip from XBL")

		if err = client.DeleteDVRClip(ctx, capture.ID); err != nil {
			return fmt.Errorf("delete clip: %w", err)
		}
	}

	return nil
}

func sanitizePath(path string) string {
	var b strings.Builder
	for _, r := range path {
		switch {
		case r == ' ' || r == '-' || r == '_':
			b.WriteRune(r)
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
		}
	}
	return b.String()
}
