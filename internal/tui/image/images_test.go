package image

import (
	stdimage "image"
	"image/color"
	"os"
	"testing"
)

func TestValidateFileSize(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/test.txt"
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	// File is 5 bytes, limit 10 -> not too large
	if tooLarge, err := ValidateFileSize(path, 10); err != nil || tooLarge {
		t.Fatalf("expected not too large, got %v %v", tooLarge, err)
	}

	// File is 5 bytes, limit 3 -> too large
	if tooLarge, err := ValidateFileSize(path, 3); err != nil || !tooLarge {
		t.Fatalf("expected too large, got %v %v", tooLarge, err)
	}

	// Missing file
	if _, err := ValidateFileSize(dir+"/missing.txt", 10); err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestToString(t *testing.T) {
	img := stdimage.NewRGBA(stdimage.Rect(0, 0, 4, 4))
	img.Set(0, 0, color.RGBA{255, 0, 0, 255})
	img.Set(1, 0, color.RGBA{0, 255, 0, 255})
	img.Set(2, 0, color.RGBA{0, 0, 255, 255})
	img.Set(3, 0, color.RGBA{255, 255, 255, 255})

	out := ToString(4, img)
	if out == "" {
		t.Fatal("ToString should return non-empty string")
	}
}