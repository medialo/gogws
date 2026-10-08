package standard

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

type jsonEncoding struct{}

func (jsonEncoding) extension() string {
	return "json"
}

func (jsonEncoding) decode(data []byte, doc *document) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(doc); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}

func (jsonEncoding) encode(doc *document) ([]byte, error) {
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}
