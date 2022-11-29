package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestYamlRawMessage(t *testing.T) {
	const sample = `
plugins:
cpu: rocks
ppp:
  x: y
  z: 1
`
	mapping := make(map[string]YamlRawMessage)
	require.NoError(t, yaml.Unmarshal([]byte(sample), &mapping))
	cpuConfig, ok := mapping["cpu"]
	require.True(t, ok)
	var stringPayload string
	require.NoError(t, cpuConfig.Unmarshal(&stringPayload))
	require.Equal(t, "rocks", stringPayload)

	type objectStruct struct {
		X string
		Z int
	}
	var objectPayload objectStruct
	objectConfig, ok := mapping["ppp"]
	require.True(t, ok)
	require.NoError(t, objectConfig.Unmarshal(&objectPayload))
	require.Equal(t, objectStruct{
		X: "y",
		Z: 1,
	}, objectPayload)
}
