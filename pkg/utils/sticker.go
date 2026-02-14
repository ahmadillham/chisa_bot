package utils

import (
	"bytes"
	"encoding/binary"
)

// AddStickerExif writes WhatsApp-compatible Exif metadata into a WebP file
// so that the sticker appears with a pack name and author.
func AddStickerExif(webpData []byte, packName, author string) ([]byte, error) {
	// Build the Exif payload following the WhatsApp sticker Exif spec.
	// WhatsApp reads EXIF tags 0x0501 (pack name) and 0x0502 (author) from the "EXIF" IFD.

	exifPayload := buildExifPayload(packName, author)
	riffChunk := buildRIFFChunk("EXIF", exifPayload)

	// Append the EXIF chunk before the end of the RIFF container.
	// WebP files are RIFF containers: "RIFF" + size + "WEBP" + chunks
	if len(webpData) < 12 {
		return webpData, nil
	}

	// Remove any existing EXIF chunk to avoid duplication.
	cleaned := removeChunk(webpData, "EXIF")

	// Insert the new EXIF chunk at the end of the RIFF data.
	result := make([]byte, 0, len(cleaned)+len(riffChunk))
	result = append(result, cleaned...)
	result = append(result, riffChunk...)

	// Update the RIFF header size (bytes 4-7, little-endian, = file size - 8).
	newSize := uint32(len(result) - 8)
	binary.LittleEndian.PutUint32(result[4:8], newSize)

	return result, nil
}

func buildExifPayload(packName, author string) []byte {
	var buf bytes.Buffer

	// Exif header
	buf.Write([]byte{0x49, 0x49}) // Little-endian ("II")
	buf.Write([]byte{0x2A, 0x00}) // TIFF magic
	buf.Write([]byte{0x08, 0x00, 0x00, 0x00}) // Offset to first IFD

	// IFD with 2 entries
	ifdEntryCount := uint16(2)
	binary.Write(&buf, binary.LittleEndian, ifdEntryCount)

	packNameBytes := []byte(packName)
	authorBytes := []byte(author)

	// Calculate offsets: IFD header(2) + 2 entries(24) + next IFD ptr(4) = 30
	// Data starts at offset 8 (IFD start) + 30 = 38
	dataOffset := uint32(8 + 2 + 12*2 + 4)

	// Entry 1: Tag 0x0501 = sticker pack name (ASCII)
	binary.Write(&buf, binary.LittleEndian, uint16(0x0501))
	binary.Write(&buf, binary.LittleEndian, uint16(2)) // ASCII type
	binary.Write(&buf, binary.LittleEndian, uint32(len(packNameBytes)+1))
	if len(packNameBytes)+1 <= 4 {
		padded := make([]byte, 4)
		copy(padded, packNameBytes)
		buf.Write(padded)
	} else {
		binary.Write(&buf, binary.LittleEndian, dataOffset)
	}

	authorDataOffset := dataOffset
	if len(packNameBytes)+1 > 4 {
		authorDataOffset = dataOffset + uint32(len(packNameBytes)+1)
		// Align to even
		if authorDataOffset%2 != 0 {
			authorDataOffset++
		}
	}

	// Entry 2: Tag 0x0502 = sticker author (ASCII)
	binary.Write(&buf, binary.LittleEndian, uint16(0x0502))
	binary.Write(&buf, binary.LittleEndian, uint16(2)) // ASCII type
	binary.Write(&buf, binary.LittleEndian, uint32(len(authorBytes)+1))
	if len(authorBytes)+1 <= 4 {
		padded := make([]byte, 4)
		copy(padded, authorBytes)
		buf.Write(padded)
	} else {
		binary.Write(&buf, binary.LittleEndian, authorDataOffset)
	}

	// Next IFD offset = 0 (no more IFDs)
	binary.Write(&buf, binary.LittleEndian, uint32(0))

	// Write pack name data (if > 4 bytes)
	if len(packNameBytes)+1 > 4 {
		buf.Write(packNameBytes)
		buf.WriteByte(0) // null terminator
		if (len(packNameBytes)+1)%2 != 0 {
			buf.WriteByte(0) // padding
		}
	}

	// Write author data (if > 4 bytes)
	if len(authorBytes)+1 > 4 {
		buf.Write(authorBytes)
		buf.WriteByte(0)
	}

	return buf.Bytes()
}

func buildRIFFChunk(fourCC string, data []byte) []byte {
	// Pad data to even length
	padded := data
	if len(data)%2 != 0 {
		padded = make([]byte, len(data)+1)
		copy(padded, data)
	}

	chunk := make([]byte, 8+len(padded))
	copy(chunk[0:4], fourCC)
	binary.LittleEndian.PutUint32(chunk[4:8], uint32(len(data)))
	copy(chunk[8:], padded)
	return chunk
}

func removeChunk(webpData []byte, fourCC string) []byte {
	if len(webpData) < 12 {
		return webpData
	}
	target := []byte(fourCC)
	pos := 12 // Skip "RIFF" + size + "WEBP"
	var result []byte
	result = append(result, webpData[:12]...)

	for pos+8 <= len(webpData) {
		chunkID := webpData[pos : pos+4]
		chunkSize := binary.LittleEndian.Uint32(webpData[pos+4 : pos+8])
		totalChunkSize := 8 + int(chunkSize)
		if totalChunkSize%2 != 0 {
			totalChunkSize++ // padding byte
		}
		if !bytes.Equal(chunkID, target) {
			end := pos + totalChunkSize
			if end > len(webpData) {
				end = len(webpData)
			}
			result = append(result, webpData[pos:end]...)
		}
		pos += totalChunkSize
	}
	return result
}
