package whisperlive

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func createSRTFile(segments []Segment, outputPath string) error {
	if strings.TrimSpace(outputPath) == "" {
		return fmt.Errorf("output path is required")
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil && filepath.Dir(outputPath) != "." {
		return fmt.Errorf("creating output directory: %w", err)
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create srt file: %w", err)
	}
	defer f.Close()

	for i, seg := range segments {
		start := formatSRTTimestamp(seg.Start)
		end := formatSRTTimestamp(seg.End)
		text := strings.TrimSpace(seg.Text)
		if _, err := fmt.Fprintf(f, "%d\n%s --> %s\n%s\n\n", i+1, start, end, text); err != nil {
			return fmt.Errorf("write srt entry: %w", err)
		}
	}

	return nil
}

func formatSRTTimestamp(seconds float64) string {
	if seconds < 0 {
		seconds = 0
	}
	hours := int(seconds) / 3600
	minutes := (int(seconds) % 3600) / 60
	secs := int(seconds) % 60
	millis := int((seconds - float64(int(seconds))) * 1000)
	millis = min(max(millis, 0), 999)
	return fmt.Sprintf("%02d:%02d:%02d,%03d", hours, minutes, secs, millis)
}
