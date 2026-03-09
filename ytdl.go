package ytdl

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kkdai/youtube/v2"
)

type DownloadOptions struct {
	URL      string
	OutDir   string
	Filename string
	Client   *youtube.Client
}

func Download(ctx context.Context, opt DownloadOptions) (string, error) {

	opt.URL = NormalizeURL(opt.URL)

	if strings.TrimSpace(opt.URL) == "" {
		return "", fmt.Errorf("missing url")
	}

	outDir := strings.TrimSpace(opt.OutDir)
	if outDir == "" {
		outDir = "."
	}

	client := opt.Client
	if client == nil {
		client = &youtube.Client{}
	}

	video, err := client.GetVideoContext(ctx, opt.URL)
	if err != nil {
		return "", fmt.Errorf("get video: %w", err)
	}

	format := pickBestMuxed(video.Formats)
	if format == nil {
		return "", fmt.Errorf("no suitable muxed format found")
	}

	name := strings.TrimSpace(opt.Filename)
	if name == "" {
		name = slug(video.Title)
	}
	if name == "" {
		name = "video"
	}

	ext := extFromMime(format.MimeType)
	final := filepath.Join(outDir, name+ext)
	part := final + ".part"

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", fmt.Errorf("create output directory: %w", err)
	}

	stream, _, err := client.GetStreamContext(ctx, video, format)
	if err != nil {
		return "", fmt.Errorf("get stream: %w", err)
	}
	defer stream.Close()

	out, err := os.Create(part)
	if err != nil {
		return "", fmt.Errorf("create output file: %w", err)
	}

	ok := false
	defer func() {
		_ = out.Close()
		if !ok {
			_ = os.Remove(part)
		}
	}()

	if _, err := io.Copy(out, stream); err != nil {
		return "", fmt.Errorf("download stream: %w", err)
	}

	if err := out.Close(); err != nil {
		return "", fmt.Errorf("close output file: %w", err)
	}

	if err := os.Rename(part, final); err != nil {
		return "", fmt.Errorf("rename output file: %w", err)
	}

	ok = true

	abs, err := filepath.Abs(final)
	if err != nil {
		return final, nil
	}

	return abs, nil
}

func pickBestMuxed(formats youtube.FormatList) *youtube.Format {
	var best *youtube.Format

	for i := range formats {
		f := &formats[i]

		if f.AudioChannels == 0 {
			continue
		}

		mime := strings.ToLower(f.MimeType)
		if !strings.Contains(mime, "video/") {
			continue
		}

		if best == nil {
			best = f
			continue
		}

		bestMP4 := strings.Contains(strings.ToLower(best.MimeType), "mp4")
		thisMP4 := strings.Contains(mime, "mp4")

		switch {
		case thisMP4 && !bestMP4:
			best = f
		case thisMP4 == bestMP4 && f.Height > best.Height:
			best = f
		case thisMP4 == bestMP4 && f.Height == best.Height && f.Bitrate > best.Bitrate:
			best = f
		}
	}

	return best
}

func extFromMime(mime string) string {
	mime = strings.ToLower(mime)

	switch {
	case strings.Contains(mime, "mp4"):
		return ".mp4"
	case strings.Contains(mime, "webm"):
		return ".webm"
	case strings.Contains(mime, "3gpp"):
		return ".3gp"
	default:
		return ".bin"
	}
}

func slug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "&", " and ")
	s = strings.ReplaceAll(s, "+", " plus ")

	reBad := regexp.MustCompile(`[^a-z0-9]+`)
	s = reBad.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")

	reDash := regexp.MustCompile(`-+`)
	s = reDash.ReplaceAllString(s, "-")

	return s
}
