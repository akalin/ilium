package ilium

import "encoding/json"
import "io/ioutil"

func ParseConfig(inputPath string) (
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
