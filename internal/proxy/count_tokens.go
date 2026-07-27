package proxy

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"strings"

	"github.com/bassner/claudodex/internal/codex"
)

const (
	imagePatchSize            = 32
	lowDetailImageTokens      = 256 // GPT-5.6 low detail receives a 512x512 image.
	unknownImageTokenEstimate = 1844
)

func (s *Server) handleCountTokens(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxMessagesBody+1))
	if err != nil {
		writeAnthropicError(w, http.StatusBadRequest, "invalid_request_error", "failed to read request body")
		return
	}
	if len(body) > maxMessagesBody {
		writeAnthropicError(w, http.StatusRequestEntityTooLarge, "invalid_request_error", "request body is too large")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"input_tokens": estimateTokenCountFromBytes(body, false)})
}

func estimateTokenCountFromBytes(data []byte, truncated bool) int {
	normalized, imageTokens := normalizeTokenEstimateInput(data)
	tokens := (len(normalized) + 2) / 3 // deliberately conservative JSON chars/token fallback.
	tokens += imageTokens
	if truncated {
		tokens += 1_000_000
	}
	if tokens < 1 {
		return 1
	}
	return tokens
}

func estimateCodexInputItems(items []codex.InputItem) int {
	if len(items) == 0 {
		return 0
	}
	data, err := json.Marshal(items)
	if err != nil {
		return 0
	}
	return estimateTokenCountFromBytes(data, false)
}

func normalizeTokenEstimateInput(data []byte) ([]byte, int) {
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return data, 0
	}
	imageTokens := scrubImagePayloads(value)
	normalized, err := json.Marshal(value)
	if err != nil {
		return data, imageTokens
	}
	return normalized, imageTokens
}

func scrubImagePayloads(value any) int {
	switch v := value.(type) {
	case map[string]any:
		imageTokens := 0
		typ, _ := v["type"].(string)
		isImage := isImageType(typ)
		detail, _ := v["detail"].(string)
		estimatedImage := false

		if imageURL, ok := v["image_url"]; ok {
			imageTokens += imageURLTokenEstimate(imageURL, detail)
			v["image_url"] = ""
			isImage = true
			estimatedImage = true
		}
		if source, ok := v["source"].(map[string]any); ok {
			sourceType, _ := source["type"].(string)
			mediaType, _ := source["media_type"].(string)
			isImage = isImage || strings.HasPrefix(strings.ToLower(mediaType), "image/")
			if isImage && (sourceType == "base64" || sourceType == "url") {
				switch sourceType {
				case "base64":
					encoded, _ := source["data"].(string)
					imageTokens += encodedImageTokenEstimate(encoded, detail)
					estimatedImage = true
				case "url":
					imageTokens += imageURLTokenEstimate(source["url"], detail)
					estimatedImage = true
				}
				if _, ok := source["data"]; ok {
					source["data"] = ""
				}
				if _, ok := source["url"]; ok {
					source["url"] = ""
				}
			}
		}
		if isImage && !estimatedImage {
			imageTokens += unknownImageTokenEstimate
		}
		for _, child := range v {
			imageTokens += scrubImagePayloads(child)
		}
		return imageTokens
	case []any:
		imageTokens := 0
		for _, child := range v {
			imageTokens += scrubImagePayloads(child)
		}
		return imageTokens
	default:
		return 0
	}
}

func isImageType(value string) bool {
	value = strings.ToLower(value)
	return value == "image" || value == "input_image"
}

func imageURLTokenEstimate(value any, detail string) int {
	switch imageURL := value.(type) {
	case string:
		const base64Marker = ";base64,"
		marker := strings.Index(strings.ToLower(imageURL), base64Marker)
		if marker < 0 {
			return unknownImageTokenEstimate
		}
		return encodedImageTokenEstimate(imageURL[marker+len(base64Marker):], detail)
	case map[string]any:
		nestedDetail, _ := imageURL["detail"].(string)
		if nestedDetail == "" {
			nestedDetail = detail
		}
		return imageURLTokenEstimate(imageURL["url"], nestedDetail)
	default:
		return unknownImageTokenEstimate
	}
}

func encodedImageTokenEstimate(encoded, detail string) int {
	if strings.EqualFold(strings.TrimSpace(detail), "low") {
		return lowDetailImageTokens
	}
	width, height, ok := base64ImageDimensions(encoded)
	if !ok {
		return unknownImageTokenEstimate
	}
	return imagePatchTokenCount(width, height)
}

func imagePatchTokenCount(width, height int) int {
	if width <= 0 || height <= 0 {
		return unknownImageTokenEstimate
	}
	patchesWide := (int64(width) + imagePatchSize - 1) / imagePatchSize
	patchesHigh := (int64(height) + imagePatchSize - 1) / imagePatchSize
	const maxInt = int64(^uint(0) >> 1)
	if patchesHigh > 0 && patchesWide > maxInt/patchesHigh {
		return int(maxInt)
	}
	return int(patchesWide * patchesHigh)
}

func base64ImageDimensions(encoded string) (int, int, bool) {
	encoded = strings.TrimSpace(encoded)
	if encoded == "" {
		return 0, 0, false
	}

	decoder := base64.NewDecoder(base64.StdEncoding, strings.NewReader(encoded))
	header := make([]byte, 30)
	headerBytes, _ := io.ReadFull(decoder, header)
	header = header[:headerBytes]
	if width, height, ok := webPDimensions(header); ok {
		return width, height, true
	}

	decoder = base64.NewDecoder(base64.StdEncoding, strings.NewReader(encoded))
	config, _, err := image.DecodeConfig(decoder)
	if err != nil || config.Width <= 0 || config.Height <= 0 {
		return 0, 0, false
	}
	return config.Width, config.Height, true
}

func webPDimensions(header []byte) (int, int, bool) {
	if len(header) < 30 ||
		!bytes.Equal(header[0:4], []byte("RIFF")) ||
		!bytes.Equal(header[8:12], []byte("WEBP")) {
		return 0, 0, false
	}
	switch string(header[12:16]) {
	case "VP8 ":
		if !bytes.Equal(header[23:26], []byte{0x9d, 0x01, 0x2a}) {
			return 0, 0, false
		}
		width := int(uint16(header[26])|uint16(header[27])<<8) & 0x3fff
		height := int(uint16(header[28])|uint16(header[29])<<8) & 0x3fff
		return width, height, width > 0 && height > 0
	case "VP8L":
		if header[20] != 0x2f {
			return 0, 0, false
		}
		bits := uint32(header[21]) |
			uint32(header[22])<<8 |
			uint32(header[23])<<16 |
			uint32(header[24])<<24
		return int(bits&0x3fff) + 1, int((bits>>14)&0x3fff) + 1, true
	case "VP8X":
		width := int(header[24]) | int(header[25])<<8 | int(header[26])<<16
		height := int(header[27]) | int(header[28])<<8 | int(header[29])<<16
		return width + 1, height + 1, true
	default:
		return 0, 0, false
	}
}
