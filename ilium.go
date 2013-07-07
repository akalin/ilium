package main

import "encoding/json"
import "io/ioutil"
import "math/rand"
import "time"
import "os"
import "runtime"

func main() {
	numRenderJobs := runtime.NumCPU()
	runtime.GOMAXPROCS(numRenderJobs)

	configBytes, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	var config map[string]interface{}
	err = json.Unmarshal(configBytes, &config)
	if err != nil {
		panic(err)
	}
	sceneConfig := config["scene"].(map[string]interface{})
	scene := MakeScene(sceneConfig)
	rendererConfig := config["renderer"].(map[string]interface{})
	renderer := MakeRenderer(rendererConfig)
	seed := time.Now().UTC().UnixNano()
	rand := rand.New(rand.NewSource(seed))
	renderer.Render(numRenderJobs, rand, &scene)
}
