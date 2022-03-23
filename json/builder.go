package json

import (
	"encoding/json"
)

type JSONBuilder interface {
	Field(key string, value interface{}) JSONBuilder
	BuildString() (string, error)
	BuildBytes() ([]byte, error)
	BuildMap() map[string]interface{}
}

type jsonBuilder struct {
	jsonMap map[string]interface{}
}

func NewJsonBuilder() JSONBuilder {
	return jsonBuilder{
		jsonMap: make(map[string]interface{}),
	}
}

func (b jsonBuilder) Field(key string, value interface{}) JSONBuilder {
	b.jsonMap[key] = value
	return b
}

func (b jsonBuilder) BuildString() (string, error) {
	bytes, err := b.BuildBytes()
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func (b jsonBuilder) BuildBytes() ([]byte, error) {
	return json.Marshal(b.jsonMap)
}

func (b jsonBuilder) BuildMap() map[string]interface{} {
	copied := make(map[string]interface{})
	for k, v := range b.jsonMap {
		copied[k] = v
	}
	return copied
}
