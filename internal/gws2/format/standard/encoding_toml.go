package standard

import (
	"bytes"

	"github.com/pelletier/go-toml/v2"
)

type tomlEncoding struct{}

func (tomlEncoding) extension() string {
	return "toml"
}

func (tomlEncoding) decode(data []byte, doc *document) error {
	decoder := toml.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	return decoder.Decode(doc)
}

func (tomlEncoding) encode(doc *document) ([]byte, error) {
	var buf bytes.Buffer
	encoder := toml.NewEncoder(&buf)
	encoder.SetIndentTables(true)
	if err := encoder.Encode(doc); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
