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

func processSceneFile(rng *rand.Rand, scenePath string, numRenderJobs int) {
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
	renderer.Render(numRenderJobs, rng, &scene)
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
		defer func() {
			if err := f.Close(); err != nil {
				panic(err)
			}
		}()

		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

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
		extension := filepath.Ext(inputPath)
		switch extension {
		case ".json":
			processSceneFile(rng, inputPath, *numRenderJobs)
		default:
			panic("Unknown extension: " + extension)
		}
	}
}
