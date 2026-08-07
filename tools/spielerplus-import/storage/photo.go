package storage

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg" // register the JPEG decoder for image.DecodeConfig
	_ "image/png"  // register the PNG decoder for image.DecodeConfig
	"net/http"
)

// maxDecodePixels caps a photo's total pixel count before it would be fully
// decoded elsewhere, matching the backend's own decompression-bomb guard
// (backend/internal/auth/service.go's maxDecodePixels) - this importer never
// actually decodes/resizes the image (SpielerPlus already serves a
// pre-sized 200x200 rendition), but validates it the same way before
// uploading, since the source bytes come from a third party.
const maxDecodePixels = 50_000_000

// ErrUnsupportedPhotoFormat is returned by ValidatePhoto for anything other
// than a JPEG or PNG - the same two formats the backend's own photo upload
// endpoint accepts.
var ErrUnsupportedPhotoFormat = errors.New("storage: only JPEG and PNG images are supported")

// ErrPhotoTooLarge is returned by ValidatePhoto for an image whose declared
// dimensions exceed maxDecodePixels.
var ErrPhotoTooLarge = errors.New("storage: image dimensions exceed the allowed maximum")

// ValidatePhoto sniffs data's real content type (JPEG/PNG only, matching
// the backend's own upload validation) and checks its declared pixel count
// stays within maxDecodePixels, without fully decoding it. Returns the
// detected content type for Store.Put.
func ValidatePhoto(data []byte) (contentType string, err error) {
	contentType = http.DetectContentType(data)
	if contentType != "image/jpeg" && contentType != "image/png" {
		return "", fmt.Errorf("%w: detected %q", ErrUnsupportedPhotoFormat, contentType)
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("storage: decode image config: %w", err)
	}
	if cfg.Width*cfg.Height > maxDecodePixels {
		return "", fmt.Errorf("%w (%dx%d)", ErrPhotoTooLarge, cfg.Width, cfg.Height)
	}
	return contentType, nil
}
