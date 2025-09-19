package handlers

import (
	"mime/multipart"
	"testing"

	"naimuBack/internal/models"
)

func TestParseImagesFromValuesSkipsInvalid(t *testing.T) {
	values := []string{"[object Object]", "{not json}", "\"/static/video.mp4\"", "https://cdn.example.com/image.jpg"}

	videos, err := parseImagesFromValues[models.Video](values)
	if err != nil {
		t.Fatalf("parseImagesFromValues returned error: %v", err)
	}

	if len(videos) != 2 {
		t.Fatalf("expected 2 parsed entries, got %d", len(videos))
	}

	if videos[0].Path != "/static/video.mp4" {
		t.Errorf("expected first video path to be unquoted, got %q", videos[0].Path)
	}

	if videos[1].Path != "https://cdn.example.com/image.jpg" {
		t.Errorf("unexpected second video path: %q", videos[1].Path)
	}
}

func TestParseImagesFromValuesArrayOfStrings(t *testing.T) {
	values := []string{"[\"/a.jpg\",\"/b.jpg\"]"}

	images, err := parseImagesFromValues[models.Image](values)
	if err != nil {
		t.Fatalf("parseImagesFromValues returned error: %v", err)
	}

	if len(images) != 2 {
		t.Fatalf("expected 2 images, got %d", len(images))
	}

	if images[0].Path != "/a.jpg" || images[1].Path != "/b.jpg" {
		t.Fatalf("unexpected image paths: %#v", images)
	}
}

func TestGatherImagesFromFormInvalidValuesIgnored(t *testing.T) {
	form := &multipart.Form{
		Value: map[string][]string{
			"videos": []string{"[object Object]", ""},
		},
	}

	videos, ok, err := gatherImagesFromForm[models.Video](form, "videos")
	if err != nil {
		t.Fatalf("gatherImagesFromForm returned error: %v", err)
	}

	if ok {
		t.Fatalf("expected ok to be false when no valid payloads are found")
	}

	if len(videos) != 0 {
		t.Fatalf("expected zero videos, got %d", len(videos))
	}
}
