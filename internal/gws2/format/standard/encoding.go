package standard

type encoding interface {
	extension() string
	decode(data []byte, doc *document) error
	encode(doc *document) ([]byte, error)
}

var encodings = []encoding{yamlEncoding{"yaml"}, yamlEncoding{"yml"}, jsonEncoding{}, tomlEncoding{}}
