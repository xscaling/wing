package utils

import (
	"regexp"
	"strconv"

	"k8s.io/apimachinery/pkg/version"
)

type KubernetesVersion struct {
	Version       *version.Info
	Major         int
	Minor         int
	PrettyVersion string
	Parsed        bool
}

var (
	kubernetesVersionDigitsRegexp = regexp.MustCompile(`^([0-9]+)`)
)

func NewKubernetesVersion(version *version.Info) KubernetesVersion {
	majorStr := kubernetesVersionDigitsRegexp.FindString(version.Major)
	minorStr := kubernetesVersionDigitsRegexp.FindString(version.Minor)
	var (
		err           error
		parsedVersion = KubernetesVersion{
			Version: version,
			Parsed:  true,
		}
	)
	if parsedVersion.Major, err = strconv.Atoi(majorStr); err != nil {
		parsedVersion.Parsed = false
	}
	if parsedVersion.Minor, err = strconv.Atoi(minorStr); err != nil {
		parsedVersion.Parsed = false
	}
	if parsedVersion.Parsed {
		parsedVersion.PrettyVersion = version.Major + "." + version.Minor
	}
	return parsedVersion
}
