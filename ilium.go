package main

import "encoding/json"
import "flag"
import "fmt"
import "io/ioutil"
import "math/rand"
import "time"
import "os"
import "runtime"

func main() {
	numRenderJobs := flag.Int(
		"j", runtime.NumCPU(), "how many render jobs to spawn")

	flag.Parse()

	runtime.GOMAXPROCS(*numRenderJobs)

	seed := time.Now().UTC().UnixNano()
	rng := rand.New(rand.NewSource(seed))

	if flag.NArg() < 1 {
		fmt.Fprintf(
			os.Stderr, "%s [options] [scene.json...]\n",
			os.Args[0])
		flag.PrintDefaults()
		os.Exit(-1)
	}

	for i := 0; i < flag.NArg(); i++ {
		inputPath := flag.Arg(i)
		fmt.Printf(
			"Processing %s (%d/%d)...\n",
			inputPath, i+1, flag.NArg())
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
		renderer.Render(*numRenderJobs, rng, &scene)
	}
}
