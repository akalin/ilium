package ilium

import "encoding/json"
import "errors"
import "io/ioutil"

func rawParseConfig(inputPath string) (
	config map[string]interface{}, err error) {
	configBytes, err := ioutil.ReadFile(inputPath)
	if err != nil {
		return
	}
	err = json.Unmarshal(configBytes, &config)
	if err != nil {
		config = nil
		return
	}
	return
}

func processIncludes(config interface{}, depth int) (
	newConfig interface{}, err error) {
	if depth >= 10 {
		err = errors.New("Include depth limit exceeded")
		return
	}

	switch typedConfig := config.(type) {
	case map[string]interface{}:
		if includeConfig, ok := typedConfig["_include"]; ok {
			includePath := includeConfig.(string)
			newConfig, err = parseConfigRecursively(
				includePath, depth+1)
			if err != nil {
				newConfig = nil
			}
			return
		}

		for k, v := range typedConfig {
			v, err = processIncludes(v, depth)
			if err != nil {
				return
			}
			typedConfig[k] = v
		}

	case []interface{}:
		for i, v := range typedConfig {
			v, err = processIncludes(v, depth)
			if err != nil {
				return
			}
			typedConfig[i] = v
		}
	}

	newConfig = config
	return
}

func parseConfigRecursively(inputPath string, depth int) (
	config map[string]interface{}, err error) {
	config, err = rawParseConfig(inputPath)
	if err != nil {
		return
	}
	var processedConfig interface{}
	processedConfig, err = processIncludes(config, 0)
	if err != nil {
		config = nil
		return
	}
	config = processedConfig.(map[string]interface{})
	return
}

func ParseConfig(inputPath string) (config map[string]interface{}, err error) {
	return parseConfigRecursively(inputPath, 0)
}
