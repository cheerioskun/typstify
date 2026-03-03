package viewer

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	"io"
	"log"
	"os"
	"sync/atomic"

	_ "golang.org/x/image/webp"
	"looz.ws/typstify/utils"

	"gioui.org/op/paint"
	xdraw "golang.org/x/image/draw"
)

type Quality uint8

const (
	// scale source image using the nearest neighbor interpolator.
	Low Quality = iota
	// scale using ApproxBiLinear interpolator.
	Medium
	// scale using BiLinear interpolator. It is slow but gives high quality.
	High
	// scale using Catmull-Rom interpolator. It is very slow but gives the
	// best quality among the four.
	Highest
)

var emptyImg = paint.NewImageOp(image.NewUniform(color.Opaque))

// ImageSource wraps a local or remote image. Only jpeg, png and gif format for
// is supported. When displaying, image scaled by the specified size and cached.
type ImageSource struct {
	// location is the local file path for local image.
	location string
	src      *bytes.Buffer
	srcSize  image.Point

	// for local and network image
	isLoading atomic.Bool
	loaded    bool
	loadErr   error
	// cache the loaded image
	cache        *paint.ImageOp
	ScaleQuality Quality
	// onLoaded is a callback called when image data is loaded.
	onLoaded       func()
	onLoadedRedraw func()

	// lru cache for different sized images which is scaled during window resize.
	imgCache *utils.LruCache[*image.RGBA]
}

// ImageFromFile load an image from local filesystem or from network lazily.
// For eager image loading, use ImageFromBuf instead.
func ImageFromFile(src string) *ImageSource {
	return &ImageSource{location: src, ScaleQuality: Medium}
}

func (img *ImageSource) loadMeta(srcReader io.Reader) error {
	imgConfig, _, err := image.DecodeConfig(srcReader)
	if err != nil {
		img.loadErr = err
		return err
	}

	img.srcSize = image.Point{X: imgConfig.Width, Y: imgConfig.Height}
	return nil
}

// loads the img from network asynchronously.
func (img *ImageSource) load() {
	if img.loaded {
		return
	}

	if !img.isLoading.CompareAndSwap(false, true) {
		return
	}

	go func() {
		defer func() {
			if img.isLoading.CompareAndSwap(true, false) {
				if img.imgCache != nil {
					img.imgCache.Clear()
				}

				img.loaded = true
				if img.onLoaded != nil {
					img.onLoaded()
				}

				if img.onLoadedRedraw != nil {
					img.onLoadedRedraw()
				}

			}
		}()

		// Try to load from the file system.
		reader, err := os.Open(img.location)
		if err != nil {
			img.loadErr = err
			return
		}
		defer reader.Close()

		if img.src == nil {
			img.src = &bytes.Buffer{}
		} else {
			img.src.Reset()
		}

		_, err = img.src.ReadFrom(reader)
		if err != nil {
			img.loadErr = err
			return
		}

		img.loadMeta(bytes.NewReader(img.src.Bytes()))

	}()

}

func (img *ImageSource) ScaleByRatio(ratio float32) (*paint.ImageOp, error) {
	if ratio <= 0 {
		ratio = 1.0
	}

	width, height := img.srcSize.X, img.srcSize.Y
	size := image.Point{X: int(float32(width) * ratio), Y: int(float32(height) * ratio)}

	if size == (image.Point{}) {
		size = img.srcSize
	}

	if img.cache != nil && size == img.cache.Size() {
		return img.cache, nil
	}

	dest := img.getCachedImg(ratio)

	op := paint.NewImageOp(dest)
	img.cache = &op

	return img.cache, nil
}

// 20 discret sized images can be cached.
var steps = 20

func (img *ImageSource) getCachedImg(ratio float32) *image.RGBA {
	if img.imgCache == nil {
		// set a small cache size to balance the memory usage and performance(gain by reducing scale & memory allocation).
		img.imgCache = utils.NewLruCache[*image.RGBA](3, nil)
	}

	cacheKey := int(ratio * float32(steps))

	key := fmt.Sprintf("%d", cacheKey)
	dest := img.imgCache.Get(key)
	if dest != nil {
		//log.Println("Hit cache key: ", cacheKey, ratio)
		return dest
	}

	//log.Println("cache missed: ", cacheKey, ratio)

	// create new
	newRatio := float32(cacheKey) / float32(steps)
	size := image.Point{X: int(float32(img.srcSize.X) * newRatio), Y: int(float32(img.srcSize.Y) * newRatio)}

	dest = image.NewRGBA(image.Rectangle{Max: size})
	img.scale(dest)
	img.imgCache.Put(key, dest)

	return dest

}

// Typst compier exported PNG have the same dimensions for the same PPI, this enables us to use pooled
// objects here.
//
// The result of decoding an image format might not be an image.RGBA:
// decoding a GIF results in an image.Paletted,
// decoding a JPEG results in a ycbcr.YCbCr, and the result of decoding a PNG depends on the image data.
// Here we use PNG image exported from Typst compiler, we might not get RGBA data.
//
// As paint.NewImageOp convert other format to RGBA and reallocate memory, we try to avoid it here.
func (img *ImageSource) scale(dest draw.Image) error {
	srcImg, _, err := image.Decode(bytes.NewReader(img.src.Bytes()))
	if err != nil {
		return err
	}

	var interpolator xdraw.Interpolator
	switch img.ScaleQuality {
	case Low:
		interpolator = xdraw.NearestNeighbor
	case Medium:
		interpolator = xdraw.ApproxBiLinear
	case High:
		interpolator = xdraw.BiLinear
	case Highest:
		interpolator = xdraw.CatmullRom
	default:
		interpolator = xdraw.ApproxBiLinear
	}

	interpolator.Scale(dest, dest.Bounds(), srcImg, srcImg.Bounds(), draw.Src, nil)
	return nil
}

func (img *ImageSource) Error() error {
	return img.loadErr
}

func (img *ImageSource) Reresh() {
	img.loaded = false
	img.load()
}

// ImageOp scales the src image dynamically or scales to the expected size if size if set.
// If the passed size is the zero value of image Point, image is not scaled.
func (img *ImageSource) ImageOp(size image.Point) *paint.ImageOp {
	img.load()
	if img.isLoading.Load() {
		if img.cache != nil {
			return img.cache
		}
		return &emptyImg
	}

	if img.loadErr != nil {
		return &emptyImg
	}

	width, height := img.srcSize.X, img.srcSize.Y
	ratio := min(float32(size.X)/float32(width), float32(size.Y)/float32(height))
	if ratio > 1.0 {
		// Do not scale up, do it in Gio image. Scaling of Gio Image will cause blurry output.
		ratio = 1.0
	}
	scaledImg, err := img.ScaleByRatio(ratio)
	if err != nil {
		log.Printf("scale image failed: %v", err)
		return &emptyImg
	}

	return scaledImg
}

func (img *ImageSource) Size() image.Point {
	return img.srcSize
}

func (img *ImageSource) Location() string {
	return img.location
}

func (img *ImageSource) OnLoaded(callback func()) {
	img.onLoaded = callback
}
