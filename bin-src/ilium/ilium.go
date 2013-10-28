package main

import "github.com/akalin/ilium/ilium"
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
import "strings"

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
	scene := ilium.MakeScene(sceneConfig)
	rendererConfig := config["renderer"].(map[string]interface{})
	renderer := ilium.MakeRenderer(rendererConfig)
	seed := time.Now().UTC().UnixNano()
	rand := rand.New(rand.NewSource(seed))
	renderer.Render(numRenderJobs, rand, &scene)
}

func processBinFile(binPaths []string, outputPath string) {
	firstImage, err := ilium.ReadImageFromBin(binPaths[0])
	if err != nil {
		panic(err)
	}
	for _, binPath := range binPaths {
		image, err := ilium.ReadImageFromBin(binPath)
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
		os.Stderr, "%s [options] [scene.json]\n", os.Args[0])
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

		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	runtime.GOMAXPROCS(*numRenderJobs)

	if flag.NArg() < 1 {
		onArgError()
	}

	inputPath := flag.Arg(0)
	extension := filepath.Ext(inputPath)
	switch {
	case extension == ".json":
		processSceneFile(inputPath, *numRenderJobs)
	case strings.Contains(inputPath, ".bin"):
		if len(*outputPath) == 0 {
			onArgError()
		}
		processBinFile(flag.Args(), *outputPath)
	default:
		panic("Unknown extension: " + extension)
	}
}
