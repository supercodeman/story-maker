package agent

import (
	"context"
	"encoding/json"
	"testing"

	"story-maker/server/internal/model"
)

func TestComicComposeExecutor_EmptySegments(t *testing.T) {
	executor := NewComicComposeExecutor()
	input := comicComposeInput{ComicDramaID: 1, OutputDir: t.TempDir(), Segments: []composeSegment{}}
	inputJSON, _ := json.Marshal(input)
	ec := &ExecContext{Task: &model.AITask{Prompt: string(inputJSON)}}
	_, err := executor.Execute(context.Background(), ec)
	if err == nil {
		t.Fatal("expected error for empty segments, got nil")
	}
}

func TestComicComposeExecutor_MissingOutputDir(t *testing.T) {
	executor := NewComicComposeExecutor()
	input := comicComposeInput{ComicDramaID: 1, Segments: []composeSegment{{MediaPath: "/tmp/test.mp4", Duration: 3.0, MediaType: "video"}}}
	inputJSON, _ := json.Marshal(input)
	ec := &ExecContext{Task: &model.AITask{Prompt: string(inputJSON)}}
	_, err := executor.Execute(context.Background(), ec)
	if err == nil {
		t.Fatal("expected error for missing output_dir, got nil")
	}
}

func TestComicComposeExecutor_InvalidJSON(t *testing.T) {
	executor := NewComicComposeExecutor()
	ec := &ExecContext{Task: &model.AITask{Prompt: "not valid json"}}
	_, err := executor.Execute(context.Background(), ec)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestComicComposeExecutor_UnsupportedMediaType(t *testing.T) {
	executor := NewComicComposeExecutor()
	executor.ffmpegPath = "/nonexistent/ffmpeg"
	tmpDir := t.TempDir()
	input := comicComposeInput{
		ComicDramaID: 1,
		OutputDir:    tmpDir,
		Segments:     []composeSegment{{MediaPath: "/tmp/fake.xyz", Duration: 3.0, MediaType: "unknown_type"}},
	}
	inputJSON, _ := json.Marshal(input)
	ec := &ExecContext{Task: &model.AITask{Prompt: string(inputJSON)}}
	_, err := executor.Execute(context.Background(), ec)
	if err == nil {
		t.Fatal("expected error for unsupported media type, got nil")
	}
}
