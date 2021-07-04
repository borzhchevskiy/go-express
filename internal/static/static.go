package static

import (
	"reflect"
	"strings"
)

func ProcessStatic(staticMap map[string]string, path string) (bool, string) {
	var splittedKey []string
	splittedPath := strings.Split(path, "/")
	for k, v := range staticMap {
		splittedKey = strings.Split(k, "/")
		if len(splittedKey) == len(splittedPath[:len(splittedKey)]) {
			equal := reflect.DeepEqual(splittedPath[:len(splittedKey)], splittedKey)
			if equal {
				var filepath string
				for _, v := range splittedPath[len(splittedKey):] {
					filepath += "/" + v
				}
				return true, v + string([]byte(filepath)[:len([]byte(filepath))-1])
			}
		} else {
			return false, ""
		}
	}
	return false, ""
}
