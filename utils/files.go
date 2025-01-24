package utils

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os/exec"
	"path/filepath"
)

func GetAbsPathRelativeTo(path string, reference string) string {
	if filepath.IsAbs(path) {
		return path
	} else {
		return filepath.Join(reference, path)
	}
}

func ValidateEnvironment(dependencies ...string) {
	missingDependencies := zerolog.Arr()
	hasMissingDependencies := false
	for _, dependency := range dependencies {
		_, err := exec.LookPath(dependency)
		if err != nil {
			hasMissingDependencies = true
			missingDependencies.Str(dependency)
		}
	}

	if hasMissingDependencies {
		log.Fatal().Array("dependencies", missingDependencies).Msg("Missing dependencies")
		Exit(1)
	}
}
