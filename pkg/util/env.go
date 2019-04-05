package util

import (
	"os"
	"strings"
)

func EnvToMap() map[string]string {
	res := make(map[string]string)

	for _, envVar := range os.Environ() {
		parts := strings.SplitN(envVar, "=", 2)
		res[parts[0]] = parts[1]
	}

	return res
}
