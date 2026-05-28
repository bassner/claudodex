package codex

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

const MaxSSEBlockBytes = 64 << 20

func ReadSSE(r io.Reader, handle func(SSEEvent) error) error {
	reader := bufio.NewReader(r)
	var block bytes.Buffer
	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			trimmed := strings.TrimRight(string(line), "\r\n")
			if trimmed == "" {
				if err := emitBlock(block.Bytes(), handle); err != nil {
					return err
				}
				block.Reset()
			} else {
				if block.Len()+len(line) > MaxSSEBlockBytes {
					return fmt.Errorf("SSE block exceeded %d bytes", MaxSSEBlockBytes)
				}
				block.Write(line)
			}
		}
		if err != nil {
			if err == io.EOF {
				if block.Len() > 0 {
					return emitBlock(block.Bytes(), handle)
				}
				return nil
			}
			return err
		}
	}
}

func emitBlock(block []byte, handle func(SSEEvent) error) error {
	if len(bytes.TrimSpace(block)) == 0 {
		return nil
	}
	event := "message"
	var dataLines []string
	for _, raw := range bytes.Split(block, []byte{'\n'}) {
		line := strings.TrimRight(string(raw), "\r")
		switch {
		case strings.HasPrefix(line, "event:"):
			event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		case strings.HasPrefix(line, "data:"):
			dataLines = append(dataLines, strings.TrimLeft(strings.TrimPrefix(line, "data:"), " "))
		case strings.HasPrefix(line, ":") || strings.HasPrefix(line, "id:") || strings.HasPrefix(line, "retry:"):
			continue
		case len(dataLines) > 0:
			dataLines = append(dataLines, line)
		}
	}
	if len(dataLines) == 0 {
		return nil
	}
	data := strings.Join(dataLines, "\n")
	if strings.TrimSpace(data) == "[DONE]" {
		return nil
	}
	raw := json.RawMessage(data)
	return handle(SSEEvent{Event: event, Data: raw})
}
