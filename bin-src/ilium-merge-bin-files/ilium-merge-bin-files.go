package main

import "github.com/akalin/ilium/ilium"
import "flag"
import "fmt"
import "os"
import "runtime/pprof"

func main() {
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

	if flag.NArg() < 1 || len(*outputPath) == 0 {
		fmt.Fprintf(
			os.Stderr, "%s -o output.ext [image.bin...]\n",
			os.Args[0])
		flag.PrintDefaults()
		os.Exit(-1)
	}

	firstImage, err := ilium.ReadImageFromBin(flag.Arg(0))
	if err != nil {
		panic(err)
	}
	for i := 1; i < flag.NArg(); i++ {
		image, err := ilium.ReadImageFromBin(flag.Arg(i))
		if err != nil {
			panic(err)
		}
		firstImage.Merge(image)
	}
	if err = firstImage.WriteToFile(*outputPath); err != nil {
		panic(err)
	}
}
