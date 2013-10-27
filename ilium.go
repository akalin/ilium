package main

import "encoding/json"
import "flag"
import "fmt"
import "io/ioutil"
import "math/rand"
import "time"
import "os"
import "path/filepath"
import "runtime"
import "runtime/pprof"

func processSceneFile(scenePath string, numRenderJobs int) {
	configBytes, err := ioutil.ReadFile(scenePath)
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

func main() {
	numRenderJobs := flag.Int(
		"j", runtime.NumCPU(), "how many render jobs to spawn")

	profilePath := flag.String(
		"p", "", "if non-empty, path to write the cpu profile to")

	flag.Parse()

	if len(*profilePath) > 0 {
		f, err := os.Create(*profilePath)
		if err != nil {
			panic(err)
		}

		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	runtime.GOMAXPROCS(*numRenderJobs)

	if flag.NArg() < 1 {
		fmt.Fprintf(
			os.Stderr, "%s [options] [scene.json]\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(-1)
	}

	inputPath := flag.Arg(0)
	extension := filepath.Ext(inputPath)
	switch extension {
	case ".json":
		processSceneFile(inputPath, *numRenderJobs)
	default:
		panic("Unknown extension: " + extension)
	}
}
