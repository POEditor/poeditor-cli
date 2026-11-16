package raw

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// readable rewrites JSON with every string spelled out: the API (PHP's
// json_encode) sends "café" and "a\/b", which a person should read as
// "café" and "a/b". Only what JSON requires stays escapes (quotes,
// backslashes, control characters). Key order and number text are kept
// exactly as sent; whitespace is dropped.
func readable(raw json.RawMessage) (json.RawMessage, error) {
	var b bytes.Buffer
	if err := writeReadable(&b, raw); err != nil {
		return nil, fmt.Errorf("the API returned invalid JSON: %w", err)
	}
	return b.Bytes(), nil
}

func writeReadable(b *bytes.Buffer, raw json.RawMessage) error {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return fmt.Errorf("empty value")
	}
	switch raw[0] {
	case '{':
		keys, values, ok := objectFields(raw)
		if !ok {
			return fmt.Errorf("invalid object")
		}
		b.WriteByte('{')
		for i, key := range keys {
			if i > 0 {
				b.WriteByte(',')
			}
			writeString(b, key)
			b.WriteByte(':')
			if err := writeReadable(b, values[key]); err != nil {
				return err
			}
		}
		b.WriteByte('}')
	case '[':
		var items []json.RawMessage
		if err := json.Unmarshal(raw, &items); err != nil {
			return err
		}
		b.WriteByte('[')
		for i, item := range items {
			if i > 0 {
				b.WriteByte(',')
			}
			if err := writeReadable(b, item); err != nil {
				return err
			}
		}
		b.WriteByte(']')
	case '"':
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return err
		}
		writeString(b, s)
	default: // number, true, false, null: as sent
		if !json.Valid(raw) {
			return fmt.Errorf("invalid value %q", raw)
		}
		b.Write(raw)
	}
	return nil
}

// writeString writes s as a JSON string without escaping HTML (<, >, &) or
// non-ASCII characters.
func writeString(b *bytes.Buffer, s string) {
	enc := json.NewEncoder(b)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(s)       // a string always encodes
	b.Truncate(b.Len() - 1) // Encode ends with a newline
}
