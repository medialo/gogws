package standard

import (
	"bytes"
	"errors"
	"io"

	"gopkg.in/yaml.v3"
)

type yamlEncoding struct {
	ext string
}

func (e yamlEncoding) extension() string {
	return e.ext
}

func (yamlEncoding) decode(data []byte, doc *document) error {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(doc); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}

func (yamlEncoding) encode(doc *document) ([]byte, error) {
	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(doc); err != nil {
		return nil, err
	}
	if err := encoder.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
