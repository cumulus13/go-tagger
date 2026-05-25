// Package mimelist maps image format names to MIME types.
package mimelist

import "strings"

// ImageMIME maps a format name (from image.DecodeConfig) to a MIME type.
func ImageMIME(format string) string {
	switch strings.ToUpper(strings.TrimSpace(format)) {
	case "JPEG", "JPG":
		return "image/jpeg"
	case "PNG":
		return "image/png"
	case "GIF":
		return "image/gif"
	case "BMP":
		return "image/bmp"
	case "TIFF", "TIF":
		return "image/tiff"
	case "WEBP":
		return "image/webp"
	default:
		return "image/jpeg"
	}
}

// ExtFromMIME returns a file extension for an image MIME type.
func ExtFromMIME(mime string) string {
	switch strings.ToLower(mime) {
	case "image/jpeg":
		return "jpg"
	case "image/png":
		return "png"
	case "image/gif":
		return "gif"
	case "image/bmp":
		return "bmp"
	case "image/tiff":
		return "tiff"
	case "image/webp":
		return "webp"
	default:
		return "jpg"
	}
}
