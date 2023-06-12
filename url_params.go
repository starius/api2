package api2

import (
	"fmt"
	"strings"
)

type paramMapType struct{}

type classifier struct {
	maskPartsArray [][]string
	paramsArray    [][]bool
	paramsNum      []int
}

func splitUrl(url string) []string {
	parts := strings.Split(url, "/")
	for len(parts) > 0 && parts[0] == "" {
		parts = parts[1:]
	}
	for len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	return parts
}

func newPathClassifier(masks []string) *classifier {
	maskPartsArray := make([][]string, 0, len(masks))
	paramsArray := make([][]bool, 0, len(masks))
	paramsNum := make([]int, 0, len(masks))
	for _, mask := range masks {
		parts := splitUrl(mask)
		params := make([]bool, len(parts))
		count := 0
		for i, part := range parts {
			if strings.HasPrefix(part, ":") {
				parts[i] = strings.TrimPrefix(part, ":")
				params[i] = true
				count++
			}
		}
		maskPartsArray = append(maskPartsArray, parts)
		paramsArray = append(paramsArray, params)
		paramsNum = append(paramsNum, count)
	}
	return &classifier{
		maskPartsArray: maskPartsArray,
		paramsArray:    paramsArray,
		paramsNum:      paramsNum,
	}
}

func match(pathParts, maskParts []string, params []bool, count int) (bool, map[string]string) {
	if len(pathParts) != len(maskParts) {
		return false, nil
	}
	// Check if all static parts match.
	for i := 0; i < len(pathParts); i++ {
		if !params[i] && pathParts[i] != maskParts[i] {
			return false, nil
		}
	}
	// Fill values of the parameters.
	param2value := make(map[string]string, count)
	for i := 0; i < len(pathParts); i++ {
		if !params[i] {
			continue
		}
		key := maskParts[i]
		value := pathParts[i]
		param2value[key] = value
	}
	return true, param2value
}

// Classify returns index of matching mask (-1 if not found) and parameters map.
func (c *classifier) Classify(path string) (index int, param2value map[string]string) {
	pathParts := splitUrl(path)
	for i, maskParts := range c.maskPartsArray {
		ok, param2value := match(pathParts, maskParts, c.paramsArray[i], c.paramsNum[i])
		if ok {
			return i, param2value
		}
	}
	return -1, nil
}

func findUrlKeys(mask string) []string {
	parts := strings.Split(mask, "/")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.HasPrefix(part, ":") {
			result = append(result, strings.TrimPrefix(part, ":"))
		}
	}
	return result
}

func cutUrlParams(mask string) string {
	before, _, _ := strings.Cut(mask, "/:")
	if before == mask {
		return mask
	}
	if !strings.HasSuffix(before, "/") {
		before += "/"
	}
	return before
}

func buildUrl(mask string, param2value map[string]string) (string, error) {
	urlParts := strings.Split(mask, "/")
	replaced := make(map[string]struct{}, len(param2value))
	for i, part := range urlParts {
		if !strings.HasPrefix(part, ":") {
			continue
		}
		part = strings.TrimPrefix(part, ":")
		value, has := param2value[part]
		if !has {
			return "", fmt.Errorf("unknown parameter: %s", part)
		}
		urlParts[i] = value
		replaced[part] = struct{}{}
	}
	if len(replaced) != len(param2value) {
		return "", fmt.Errorf("not all parameters were built into URL: want %d, got %d", len(param2value), len(replaced))
	}
	return strings.Join(urlParts, "/"), nil
}
