package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime/pprof"
)

type GrayPixel struct {
	y uint8
	a uint8
}

func main() {

	blurFlagPtr := flag.Bool("blur", true, "perform gaussian blur before edge detection (optional, default: true)")
	inputFileArgPtr := flag.String("input", "", "path to input file (required)")
	outputFileArgPtr := flag.String("output", "out.jpg", "path to output file (optional, default: out.jpg")
	minThresholdArgPtr := flag.Float64("min", float64(0.2), "ratio of lower threshold (optional, default: 0.2")
	maxThresholdArgPtr := flag.Float64("max", float64(0.6), "ratio of upper threshold (optional, default: 0.6")
	profileFlag := flag.Bool("profile", false, "do cpu/mem profile on the main logic")

	flag.Parse()

	if *inputFileArgPtr == "" {
		fmt.Println("No path to input file specified, nothing to do.")
		return
	}

	if !isValidRatioValue(*minThresholdArgPtr) || !isValidRatioValue(*maxThresholdArgPtr) {
		fmt.Println("Invalid value for threshold ratio given, exiting.")
		return
	}

	image.RegisterFormat("jpeg", "jpeg", jpeg.Decode, jpeg.DecodeConfig)
	image.RegisterFormat("png", "png", png.Decode, png.DecodeConfig)

	pixels := openImage(*inputFileArgPtr)
	if *profileFlag {
		cpuf, err := os.Create("cpu_profile")
		if err != nil {
			log.Fatal(err)
		}
		_ = pprof.StartCPUProfile(cpuf)
	}

	pixels = CannyEdgeDetect(pixels, *blurFlagPtr, *minThresholdArgPtr, *maxThresholdArgPtr)

	if *profileFlag {
		pprof.StopCPUProfile()

		memf, err := os.Create("mem_profile")
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
		}

		if err := pprof.WriteHeapProfile(memf); err != nil {
			log.Fatal("could not write memory profile: ", err)
		}
		_ = memf.Close()
	}

	writeImage(pixels, *outputFileArgPtr)
}

func openImage(path string) [][]GrayPixel {
	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	pixels, err := getPixelArray(file)
	if err != nil {
		log.Fatal(err)
	}

	return pixels
}

func writeImage(pixels [][]GrayPixel, path string) {

	grayImg := getImageFromArray(pixels)
	outFile, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}

	ext := filepath.Ext(path)
	if ext == "png" {
		err = png.Encode(outFile, grayImg)
	} else {
		opts := jpeg.Options{95}
		err = jpeg.Encode(outFile, grayImg, &opts)
	}
	if err != nil {
		log.Fatal(err)
	}
}

func getPixelArray(file io.Reader) ([][]GrayPixel, error) {
	var pixelArr [][]GrayPixel

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}
	height := img.Bounds().Max.Y
	width := img.Bounds().Max.X

	for y := 0; y < height; y++ {
		var row []GrayPixel
		for x := 0; x < width; x++ {
			pixel := img.At(x, y)
			grayPixel := rgbaToGrayPixel(pixel)
			row = append(row, grayPixel)
		}
		pixelArr = append(pixelArr, row)
	}

	return pixelArr, nil
}

func getImageFromArray(pixels [][]GrayPixel) *image.Gray {

	bounds := image.Rect(0, 0, len(pixels[0]), len(pixels))
	img := image.NewGray(bounds)

	for y := 0; y < len(pixels); y++ {
		for x := 0; x < len(pixels[y]); x++ {
			img.SetGray(x, y, color.Gray{pixels[y][x].y})
		}
	}

	return img
}

func isValidRatioValue(x float64) bool {
	if (x >= float64(0)) && (x <= float64(1)) {
		return true
	}
	return false
}

func rgbaToGrayPixel(pixel color.Color) GrayPixel {
	_, _, _, a := pixel.RGBA()
	gray := color.GrayModel.Convert(pixel).(color.Gray).Y

	return GrayPixel{gray, uint8(a >> 8)}
}
