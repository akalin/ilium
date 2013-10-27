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

func processBinFiles(binPaths []string, outputPath string) {
	firstImage, err := ReadImageFromBin(binPaths[0])
	if err != nil {
		panic(err)
	}
	for i, binPath := range binPaths {
		fmt.Printf(
			"Processing %s (%d/%d)...\n",
			binPath, i+1, len(binPaths))
		image, err := ReadImageFromBin(binPath)
		if err != nil {
			panic(err)
		}
		firstImage.Merge(image)
	}
	if err = firstImage.WriteToFile(outputPath); err != nil {
		panic(err)
	}
}

func onArgError() {
	fmt.Fprintf(
		os.Stderr, "%s [options] [scene.json...]\n", os.Args[0])
	fmt.Fprintf(
		os.Stderr, "%s -o output.ext [image.bin...]\n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(-1)
}

func main() {
	numRenderJobs := flag.Int(
		"j", runtime.NumCPU(), "how many render jobs to spawn")

	profilePath := flag.String(
		"p", "", "if non-empty, path to write the cpu profile to")

	outputPath := flag.String(
		"o", "", "if non-empty, path to write the output to "+
			"(only applies when processing a .bin file)")

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
		onArgError()
	}

	firstInputPath := flag.Arg(0)
	extension := filepath.Ext(firstInputPath)

	switch extension {
	case ".json":
		for i := 0; i < flag.NArg(); i++ {
			inputPath := flag.Arg(i)
			fmt.Printf(
				"Processing %s (%d/%d)...\n",
				inputPath, i+1, flag.NArg())
			processSceneFile(rng, inputPath, *numRenderJobs)
		}
	case ".bin":
		if len(*outputPath) == 0 {
			onArgError()
		}
		processBinFiles(flag.Args(), *outputPath)
	default:
		panic("Unknown extension: " + extension)
	}
}
