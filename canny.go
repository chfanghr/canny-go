package main

import (
	"errors"
	"github.com/deckarep/golang-set"
	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/gonum/stat/combin"
	"image"
	"math"
)

type direction int

const (
	HORIZONTAL direction = iota
	VERTICAL
)

var SOBEL_X = []float64{1, 0, -1, 2, 0, -2, 1, 0, -1}
var SOBEL_Y = []float64{1, 2, 1, 0, 0, 0, -1, -2, -1}

func CannyEdgeDetect(pixels [][]GrayPixel, blur bool, minRatio, maxRatio float64) [][]GrayPixel {
	if blur {
		pixels = gaussianBlur(pixels, 5)
	}
	pixels, angles := sobel(pixels)
	pixels = nonMaximumSuppression(pixels, angles)
	max := maxPixelValue(pixels)
	high := maxRatio * float64(max)
	low := minRatio * float64(max)
	strong, weak := doublethreshold(pixels, high, low)
	edgeTracking(pixels, strong, weak)

	return pixels
}

func edgeTracking(pixels [][]GrayPixel, strong, weak mapset.Set) {

	weakIter := weak.Iterator()
	for weakPixel := range weakIter.C {
		weakPoint := weakPixel.(image.Point)

		neighbours := getAdjacentPixels(pixels, weakPoint.X, weakPoint.Y)

		if strong.Intersect(neighbours).Cardinality() > 0 {
			strong.Add(weakPoint)
		}

		x := weakPoint.X
		y := weakPoint.Y
		pixels[y][x].y = uint8(0)
	}
}

func getAdjacentPixels(pixels [][]GrayPixel, x, y int) mapset.Set {
	result := mapset.NewSet()
	height := len(pixels)
	width := len(pixels[0])
	minX := int(math.Max(float64(0), float64(x-1)))
	minY := int(math.Max(float64(0), float64(y-1)))
	maxX := int(math.Min(float64(width), float64(x+1)))
	maxY := int(math.Min(float64(height), float64(y+1)))

	for i := minY; i < maxY; i++ {
		for j := minX; j < maxX; j++ {
			if (i != y) && (j != x) {
				result.Add(image.Point{j, i})
			}
		}
	}

	return result
}

func doublethreshold(pixels [][]GrayPixel, high, low float64) (mapset.Set, mapset.Set) {
	strong := mapset.NewSet()
	weak := mapset.NewSet()

	for y := 0; y < len(pixels); y++ {
		for x := 0; x < len(pixels[0]); x++ {
			pixVal := float64(pixels[y][x].y)
			if pixVal > high {
				strong.Add(image.Point{x, y})
			} else if (high > pixVal) && (pixVal > low) {
				weak.Add(image.Point{x, y})
			} else {
				pixels[y][x].y = uint8(0)
			}
		}
	}

	return strong, weak
}

func nonMaximumSuppression(pixels [][]GrayPixel, directions [][]float64) [][]GrayPixel {

	if (len(pixels) != len(directions)) || (len(pixels[0]) != len(directions[0])) {
		panic(errors.New("dimensions of pixel and direction array must match"))
	}
	var result [][]GrayPixel

	for y := 0; y < len(pixels); y++ {
		var resultRow []GrayPixel
		for x := 0; x < len(pixels[0]); x++ {
			r := pixels[y][x]
			p, q := getPixelInGradientDirection(pixels, directions, x, y)
			if (p.y > r.y) || (q.y > r.y) {
				resultRow = append(resultRow, GrayPixel{uint8(0), uint8(255)})
			} else {
				resultRow = append(resultRow, r)
			}
		}
		result = append(result, resultRow)
	}

	return result
}

func sobel(pixels [][]GrayPixel) ([][]GrayPixel, [][]float64) {
	var result [][]GrayPixel
	var directions [][]float64

	sobel_X := *mat.NewDense(3, 3, SOBEL_X)
	sobel_Y := *mat.NewDense(3, 3, SOBEL_Y)

	for y := 0; y < len(pixels); y++ {
		var resultRow []GrayPixel
		var angleRow []float64
		for x := 0; x < len(pixels[y]); x++ {
			var angle float64

			imagePane := getSorroundingPixelMatrix(pixels, y, x, 3)

			sobelRes_X := convolve(imagePane, sobel_X)
			sobelRes_Y := convolve(imagePane, sobel_Y)

			combinedRes := uint8(math.Sqrt(math.Pow(sobelRes_X, 2) + math.Pow(sobelRes_Y, 2)))
			resultRow = append(resultRow, GrayPixel{combinedRes, uint8(255)})

			if (sobelRes_X == float64(0)) || (sobelRes_Y == float64(0)) {
				angle = float64(0)
			} else {
				angle = math.Atan(sobelRes_Y / sobelRes_X)
			}
			angle = angle * (180 / math.Pi)
			angleRow = append(angleRow, angle)
		}
		result = append(result, resultRow)
		directions = append(directions, angleRow)
	}

	return result, directions
}

func gaussianBlur(pixels [][]GrayPixel, kernelSize uint) [][]GrayPixel {
	if kernelSize%2 == 0 {
		panic(errors.New("size of kernel must be odd"))
	}
	var result [][]GrayPixel
	kernel := getPascalTriangleRow(kernelSize - 1)
	kernel = normalizeVec(kernel)

	for y := 0; y < len(pixels); y++ {
		var resultRow []GrayPixel
		for x := 0; x < len(pixels[y]); x++ {
			vecVert := getPixelVector(pixels, y, x, kernel.Len(), VERTICAL)
			vecHor := getPixelVector(pixels, y, x, kernel.Len(), HORIZONTAL)
			verticalSum := innerProduct(vecVert, kernel)
			horizontalSum := innerProduct(vecHor, kernel)
			combinedRes := uint8(math.Sqrt(verticalSum*verticalSum + horizontalSum*horizontalSum))
			resultRow = append(resultRow, GrayPixel{combinedRes, 255})
		}
		result = append(result, resultRow)
	}

	return result
}

func getPixelInGradientDirection(pixels [][]GrayPixel, directions [][]float64, x, y int) (p, q GrayPixel) {
	var pY, pX, qY, qX int
	height := len(pixels)
	width := len(pixels[0])
	dirVal := directions[y][x]

	if (dirVal >= float64(-90)) && (dirVal < float64(-67.5)) {

		pY, pX = y-1, x
		qY, qX = y+1, x
	} else if (dirVal >= float64(-67.5)) && (dirVal < float64(-22.5)) {

		pY, pX = y-1, x+1
		qY, qX = y+1, x-1
	} else if (dirVal >= float64(-22.5)) && (dirVal < float64(22.5)) {

		pY, pX = y, x+1
		qY, qX = y, x-1
	} else if (dirVal >= float64(22.5)) && (dirVal < float64(67.5)) {

		pY, pX = y+1, x+1
		qY, qX = y-1, x-1
	} else if (dirVal >= float64(67.5)) && (dirVal <= float64(90)) {

		pY, pX = y+1, x
		qY, qX = y-1, x
	} else {
		panic(errors.New("invalid value for direction, out of range [-90, 90]"))
	}

	if (pY < 0) || (pY >= height) {
		pY = y
	}
	if (pX < 0) || (pX >= width) {
		pX = x
	}
	if (qY < 0) || (qY >= height) {
		qY = y
	}
	if (qX < 0) || (qX >= width) {
		qX = x
	}

	p = pixels[pY][pX]
	q = pixels[qY][qX]
	return p, q
}

func getSorroundingPixelMatrix(pixels [][]GrayPixel, posY, posX int, length int) mat.Dense {
	if length%2 == 0 {
		panic(errors.New("length must be odd number"))
	}

	var values []float64
	var currentPixel GrayPixel
	padding := (length / 2)

	minX := posX - padding
	minY := posY - padding
	maxX := posX + padding
	maxY := posY + padding
	height := len(pixels)
	width := len(pixels[0])

	var curY, curX int
	for y := minY; y <= maxY; y++ {
		if y < 0 {
			curY = posY + abs(y)
		} else if y >= height {
			overlap := y - height + 1
			curY = posY - overlap
		} else {
			curY = y
		}
		for x := minX; x <= maxX; x++ {
			if x < 0 {
				curX = posX + abs(x)
			} else if x >= width {
				overlap := x - width + 1
				curX = posX - overlap
			} else {
				curX = x
			}

			currentPixel = pixels[curY][curX]
			values = append(values, float64(currentPixel.y))
		}
	}

	return *mat.NewDense(length, length, values)
}

func getPixelVector(pixels [][]GrayPixel, posY, posX int, length int, dir direction) mat.VecDense {
	if length%2 == 0 {
		panic(errors.New("length must be odd number"))
	}

	var values []float64
	var currentPixel GrayPixel
	padding := (length / 2)

	switch dir {
	case HORIZONTAL:
		minX := posX - padding
		maxX := posX + padding
		for i := minX; i <= maxX; i++ {
			rowLength := len(pixels[posY])
			if i < 0 {
				currentPixel = pixels[posY][posX+abs(i)]
			} else if i >= rowLength {
				overlap := i - rowLength + 1
				currentPixel = pixels[posY][posX-overlap]
			} else {
				currentPixel = pixels[posY][i]
			}
			values = append(values, float64(currentPixel.y))

		}
	case VERTICAL:
		minY := posY - padding
		maxY := posY + padding
		for i := minY; i <= maxY; i++ {
			columnLength := len(pixels)
			if i < 0 {
				currentPixel = pixels[posY+abs(i)][posX]
			} else if i >= columnLength {
				overlap := i - columnLength + 1
				currentPixel = pixels[posY-overlap][posX]
			} else {
				currentPixel = pixels[i][posX]
			}
			values = append(values, float64(currentPixel.y))
		}
	}

	return *mat.NewVecDense(len(values), values)
}

func innerProduct(pixels, kernel mat.VecDense) float64 {
	if pixels.Len() != kernel.Len() {
		panic(errors.New("length of given vectors must be equal"))
	}

	var result float64 = 0
	for i := 0; i < pixels.Len(); i++ {
		result += pixels.At(i, 0) * kernel.At(i, 0)
	}

	return result
}

func convolve(m1, m2 mat.Dense) float64 {
	row_1, col_1 := m1.Dims()
	row_2, col_2 := m2.Dims()
	if row_1 != row_2 || col_1 != col_2 {
		panic(errors.New("invalid matrix dimensions for convolution operation"))
	}

	var result float64 = 0
	rows, cols := m1.Dims()

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			result += m1.At(y, x) * m2.At(y, x)
		}
	}

	return result
}

func getPascalTriangleRow(index uint) mat.VecDense {
	size := int(index + 1)
	values := make([]float64, size)

	for i := 0; i < size; i++ {
		values[i] = float64(combin.Binomial(int(index), i))
	}

	result := mat.NewVecDense(size, values)
	return *result
}

func normalizeVec(v mat.VecDense) mat.VecDense {

	var sum float64 = 0
	for i := 0; i < v.Len(); i++ {
		sum += v.At(i, 0)
	}

	var result mat.VecDense
	result.ScaleVec(1/sum, v.SliceVec(0, v.Len()))
	return result
}

func maxPixelValue(pixels [][]GrayPixel) uint8 {
	var max uint8 = 0
	for y := 0; y < len(pixels); y++ {
		for x := 0; x < len(pixels[0]); x++ {
			pixVal := pixels[y][x].y
			if pixVal > max {
				max = pixVal
			}
		}
	}

	return max
}

func abs(x int) int {
	if x < 0 {
		return (-x)
	} else {
		return x
	}
}
