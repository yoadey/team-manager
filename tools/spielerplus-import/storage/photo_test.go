package storage

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
	"testing"
)

func tinyJPEG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatalf("encode jpeg: %v", err)
	}
	return buf.Bytes()
}

func tinyPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func TestValidatePhoto_JPEG(t *testing.T) {
	ct, err := ValidatePhoto(tinyJPEG(t))
	if err != nil {
		t.Fatalf("ValidatePhoto() error = %v", err)
	}
	if ct != "image/jpeg" {
		t.Errorf("contentType = %q, want image/jpeg", ct)
	}
}

func TestValidatePhoto_PNG(t *testing.T) {
	ct, err := ValidatePhoto(tinyPNG(t))
	if err != nil {
		t.Fatalf("ValidatePhoto() error = %v", err)
	}
	if ct != "image/png" {
		t.Errorf("contentType = %q, want image/png", ct)
	}
}

func TestValidatePhoto_UnsupportedFormat(t *testing.T) {
	_, err := ValidatePhoto([]byte("<svg></svg>"))
	if err == nil {
		t.Fatal("expected an error for a non-JPEG/PNG payload")
	}
}

func TestValidatePhoto_Empty(t *testing.T) {
	if _, err := ValidatePhoto(nil); err == nil {
		t.Fatal("expected an error for empty data")
	}
}
