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

	if flag.NArg() < 1 {
		fmt.Fprintf(
			os.Stderr, "%s [options] [scene.json]\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(-1)
	}

	configBytes, err := ioutil.ReadFile(flag.Arg(0))
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
	renderer.Render(*numRenderJobs, rand, &scene)
}
