package utils

import (
	"gopkg.in/yaml.v3"
)

type YamlRawMessage struct {
	node *yaml.Node
}

func (m *YamlRawMessage) UnmarshalYAML(node *yaml.Node) error {
	m.node = node
	return nil
}

func (m YamlRawMessage) Unmarshal(receiver interface{}) error {
	return m.node.Decode(receiver)
}
