package main

import "encoding/json"
import "fmt"
import "io/ioutil"
import "math/rand"
import "time"
import "os"
import "runtime"

func main() {
	numRenderJobs := runtime.NumCPU()
	runtime.GOMAXPROCS(numRenderJobs)

	seed := time.Now().UTC().UnixNano()
	rng := rand.New(rand.NewSource(seed))

	nArgs := len(os.Args)
	for i := 1; i < nArgs; i++ {
		inputPath := os.Args[i]
		fmt.Printf(
			"Processing %s (%d/%d)...\n",
			inputPath, i, nArgs-1)
		configBytes, err := ioutil.ReadFile(inputPath)
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
		renderer.Render(numRenderJobs, rng, &scene)
	}
}
