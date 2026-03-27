package utils

import (
	"encoding/binary"
	"testing"
)

// Minimal valid WebP file header for testing.
func makeMinimalWebP() []byte {
	// RIFF header + WEBP + minimal VP8 chunk
	data := make([]byte, 20)
	copy(data[0:4], "RIFF")
	binary.LittleEndian.PutUint32(data[4:8], 12) // file size - 8
	copy(data[8:12], "WEBP")
	copy(data[12:16], "VP8 ")
	binary.LittleEndian.PutUint32(data[16:20], 0) // chunk size
	return data
}

func TestAddStickerExif_AddsExifChunk(t *testing.T) {
	webp := makeMinimalWebP()
	result, err := AddStickerExif(webp, "TestPack", "TestAuthor")
	if err != nil {
		t.Fatalf("AddStickerExif failed: %v", err)
	}

	// Result should be larger than input (exif chunk added)
	if len(result) <= len(webp) {
		t.Errorf("Result (%d bytes) should be larger than input (%d bytes)", len(result), len(webp))
	}

	// Result should start with RIFF
	if string(result[0:4]) != "RIFF" {
		t.Errorf("Result should start with RIFF, got %q", string(result[0:4]))
	}

	// Should contain WEBP
	if string(result[8:12]) != "WEBP" {
		t.Errorf("Result should contain WEBP at offset 8, got %q", string(result[8:12]))
	}

	// Should contain EXIF chunk somewhere
	found := false
	pos := 12
	for pos+8 <= len(result) {
		chunkID := string(result[pos : pos+4])
		chunkSize := binary.LittleEndian.Uint32(result[pos+4 : pos+8])
		if chunkID == "EXIF" {
			found = true
			break
		}
		totalSize := 8 + int(chunkSize)
		if totalSize%2 != 0 {
			totalSize++
		}
		pos += totalSize
	}
	if !found {
		t.Error("Result should contain an EXIF chunk")
	}

	// RIFF size should be correct
	riffSize := binary.LittleEndian.Uint32(result[4:8])
	if int(riffSize) != len(result)-8 {
		t.Errorf("RIFF size = %d, want %d", riffSize, len(result)-8)
	}
}

func TestAddStickerExif_NoDuplicateExif(t *testing.T) {
	webp := makeMinimalWebP()

	// Add exif twice
	result1, _ := AddStickerExif(webp, "Pack1", "Author1")
	result2, _ := AddStickerExif(result1, "Pack2", "Author2")

	// Count EXIF chunks
	count := 0
	pos := 12
	for pos+8 <= len(result2) {
		chunkID := string(result2[pos : pos+4])
		chunkSize := binary.LittleEndian.Uint32(result2[pos+4 : pos+8])
		if chunkID == "EXIF" {
			count++
		}
		totalSize := 8 + int(chunkSize)
		if totalSize%2 != 0 {
			totalSize++
		}
		pos += totalSize
	}
	if count != 1 {
		t.Errorf("Should have exactly 1 EXIF chunk, found %d", count)
	}
}

func TestAddStickerExif_TooSmallInput(t *testing.T) {
	tiny := []byte{0x01, 0x02, 0x03}
	result, err := AddStickerExif(tiny, "Pack", "Author")
	if err != nil {
		t.Fatalf("Should not error on small input: %v", err)
	}
	// Should return input unchanged
	if len(result) != len(tiny) {
		t.Errorf("Small input should be returned unchanged, got %d bytes", len(result))
	}
}

func TestRemoveChunk(t *testing.T) {
	webp := makeMinimalWebP()
	// Add an EXIF chunk manually
	exifData := []byte{0x01, 0x02, 0x03, 0x04}
	chunk := buildRIFFChunk("EXIF", exifData)
	withExif := append(webp, chunk...)
	binary.LittleEndian.PutUint32(withExif[4:8], uint32(len(withExif)-8))

	// Remove it
	result := removeChunk(withExif, "EXIF")

	// Check that EXIF is gone
	pos := 12
	for pos+8 <= len(result) {
		chunkID := string(result[pos : pos+4])
		if chunkID == "EXIF" {
			t.Error("EXIF chunk should have been removed")
		}
		chunkSize := binary.LittleEndian.Uint32(result[pos+4 : pos+8])
		totalSize := 8 + int(chunkSize)
		if totalSize%2 != 0 {
			totalSize++
		}
		pos += totalSize
	}
}

func TestBuildRIFFChunk(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03}
	chunk := buildRIFFChunk("TEST", data)

	// FourCC
	if string(chunk[0:4]) != "TEST" {
		t.Errorf("FourCC = %q, want TEST", string(chunk[0:4]))
	}

	// Size should be original data length (not padded)
	size := binary.LittleEndian.Uint32(chunk[4:8])
	if size != 3 {
		t.Errorf("Chunk size = %d, want 3", size)
	}

	// Total length should be 8 + padded to even (3 -> 4)
	if len(chunk) != 12 {
		t.Errorf("Total chunk length = %d, want 12 (8 header + 4 padded)", len(chunk))
	}
}
