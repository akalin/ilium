package main

import "github.com/akalin/ilium/ilium"
import "encoding/json"
import "flag"
import "fmt"
import "io/ioutil"
import "math/rand"
import "time"
import "os"
import "runtime"
import "runtime/pprof"

func main() {
	numRenderJobs := flag.Int(
		"j", runtime.NumCPU(), "how many render jobs to spawn")

	profilePath := flag.String(
		"p", "", "if non-empty, path to write the cpu profile to")

	outputDir := flag.String(
		"d", "", "if non-empty, directory to prepend to relative "+
			"output paths")

	outputExt := flag.String(
		"x", "", "if non-empty, the extension to append to "+
			"output paths (but before the real extension)")

	// This flag, combined with -j=1, can be used to get
	// repeatable renders.
	seed := flag.Int64(
		"s", time.Now().UTC().UnixNano(),
		"the seed to use for the random number generator")

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

	rng := rand.New(rand.NewSource(*seed))

	if flag.NArg() < 1 {
		fmt.Fprintf(
			os.Stderr, "%s [options] [scene.json...]\n", os.Args[0])
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
		scene := ilium.MakeScene(sceneConfig)
		rendererConfig := config["renderer"].(map[string]interface{})
		renderer := ilium.MakeRenderer(rendererConfig)
		renderer.Render(
			*numRenderJobs, rng, &scene, *outputDir, *outputExt)
	}
}
