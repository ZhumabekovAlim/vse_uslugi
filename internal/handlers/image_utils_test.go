package handlers

import (
	"bytes"
	"mime/multipart"
	"net/http/httptest"
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

func TestGatherStringsFromForm(t *testing.T) {
	form := &multipart.Form{
		Value: map[string][]string{
			"delete_images": []string{"[\"/a.jpg\", \"\", \"null\"]", "/b.jpg"},
		},
	}

	values, ok, err := gatherStringsFromForm(form, "delete_images")
	if err != nil {
		t.Fatalf("gatherStringsFromForm returned error: %v", err)
	}
	if !ok {
		t.Fatalf("expected ok=true when valid values present")
	}
	if len(values) != 2 {
		t.Fatalf("expected 2 values, got %d", len(values))
	}
	if values[0] != "/a.jpg" || values[1] != "/b.jpg" {
		t.Fatalf("unexpected values: %#v", values)
	}
}

func TestGatherStringsFromFormEmpty(t *testing.T) {
	form := &multipart.Form{
		Value: map[string][]string{
			"delete_images": []string{"", "null", "undefined"},
		},
	}

	values, ok, err := gatherStringsFromForm(form, "delete_images")
	if err != nil {
		t.Fatalf("gatherStringsFromForm returned error: %v", err)
	}
	if ok {
		t.Fatalf("expected ok=false when no valid values present")
	}
	if len(values) != 0 {
		t.Fatalf("expected zero values, got %d", len(values))
	}
}


func TestGatherStringsFromFormFilesUsesFilename(t *testing.T) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("delete_images[]", "old_photo.jpg")
	if err != nil {
		t.Fatalf("CreateFormFile failed: %v", err)
	}
	if _, err := part.Write([]byte("")); err != nil {
		t.Fatalf("writing to form file failed: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("closing writer failed: %v", err)
	}

	req := httptest.NewRequest("POST", "/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if err := req.ParseMultipartForm(1024); err != nil {
		t.Fatalf("ParseMultipartForm failed: %v", err)
	}

	values, ok, err := gatherStringsFromFormFiles(req.MultipartForm, "delete_images", "delete_images[]")
	if err != nil {
		t.Fatalf("gatherStringsFromFormFiles returned error: %v", err)
	}
	if !ok {
		t.Fatalf("expected ok=true when file payload provided")
	}
	if len(values) != 1 {
		t.Fatalf("expected a single filename entry, got %d", len(values))
	}
	if values[0] != "old_photo.jpg" {
		t.Fatalf("unexpected filename parsed: %v", values)
	}
}

func TestGatherStringsFromFormFilesParsesJSONPayload(t *testing.T) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("delete_images", "payload.json")
	if err != nil {
		t.Fatalf("CreateFormFile failed: %v", err)
	}
	if _, err := part.Write([]byte(`["/a.jpg","/b.jpg"]`)); err != nil {
		t.Fatalf("writing to form file failed: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("closing writer failed: %v", err)
	}

	req := httptest.NewRequest("POST", "/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if err := req.ParseMultipartForm(1024); err != nil {
		t.Fatalf("ParseMultipartForm failed: %v", err)
	}

	values, ok, err := gatherStringsFromFormFiles(req.MultipartForm, "delete_images")
	if err != nil {
		t.Fatalf("gatherStringsFromFormFiles returned error: %v", err)
	}
	if !ok {
		t.Fatalf("expected ok=true when textual file payload provided")
	}

	if len(values) != 3 {
		t.Fatalf("expected filename plus two entries, got %d (%v)", len(values), values)
	}

	if values[1] != "/a.jpg" || values[2] != "/b.jpg" {
		t.Fatalf("unexpected parsed values: %v", values)
	}
}

