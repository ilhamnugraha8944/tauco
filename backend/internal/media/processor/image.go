package processor

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"sort"
	"sync"

	"github.com/gen2brain/webp"
	mediaapp "github.com/ilhamnugraha8944/tauco/backend/internal/media/application"
	mediadomain "github.com/ilhamnugraha8944/tauco/backend/internal/media/domain"
	xdraw "golang.org/x/image/draw"
)

const (
	// Keep each CPU-bound image step below 10 seconds on the acceptance host,
	// leaving 15 seconds of the one-shot worker budget for S3 and PostgreSQL.
	MaxPixels      = 12_500_000
	MaxSide        = 6_000
	WebPQuality    = 82
	variantWorkers = 2
)

type Image struct{}

func (Image) Normalize(source []byte) (mediaapp.NormalizedImage, error) {
	if len(source) == 0 || len(source) > mediaapp.MaxUploadBytes {
		return mediaapp.NormalizedImage{}, errors.New("media exceeds the 10 MiB source limit")
	}
	format, err := detectFormat(source)
	if err != nil {
		return mediaapp.NormalizedImage{}, err
	}
	config, err := decodeConfig(format, source)
	if err != nil {
		return mediaapp.NormalizedImage{}, errors.New("media header is corrupt")
	}
	if err := validateDimensions(config.Width, config.Height); err != nil {
		return mediaapp.NormalizedImage{}, err
	}
	decoded, err := decode(format, source)
	if err != nil {
		return mediaapp.NormalizedImage{}, errors.New("media data is corrupt or truncated")
	}
	if format == "jpeg" {
		decoded = applyOrientation(decoded, jpegOrientation(source))
	}
	var output bytes.Buffer
	if err := (&png.Encoder{CompressionLevel: png.BestSpeed}).Encode(&output, decoded); err != nil {
		return mediaapp.NormalizedImage{}, fmt.Errorf("normalize media: %w", err)
	}
	digest := sha256.Sum256(output.Bytes())
	bounds := decoded.Bounds()
	return mediaapp.NormalizedImage{
		Data: output.Bytes(), MIME: "image/png",
		Width: bounds.Dx(), Height: bounds.Dy(), SHA256: hex.EncodeToString(digest[:]),
	}, nil
}

func (Image) Variants(normalized []byte) ([]mediaapp.GeneratedVariant, []int, error) {
	if len(normalized) == 0 {
		return nil, nil, errors.New("normalized media is empty")
	}
	source, err := png.Decode(bytes.NewReader(normalized))
	if err != nil {
		return nil, nil, errors.New("normalized media must be a valid PNG")
	}
	bounds := source.Bounds()
	if err := validateDimensions(bounds.Dx(), bounds.Dy()); err != nil {
		return nil, nil, err
	}

	skipped := make([]int, 0, len(mediadomain.VariantWidths))
	targets := make([]int, 0, len(mediadomain.VariantWidths))
	for _, width := range mediadomain.VariantWidths {
		if width > bounds.Dx() {
			skipped = append(skipped, width)
		} else {
			targets = append(targets, width)
		}
	}
	if len(targets) == 0 {
		// Even a source narrower than the smallest preset needs one public WebP.
		targets = append(targets, bounds.Dx())
	}
	type result struct {
		variant mediaapp.GeneratedVariant
		err     error
	}
	results := make(chan result, len(targets))
	semaphore := make(chan struct{}, variantWorkers)
	var wait sync.WaitGroup
	for _, width := range targets {
		wait.Add(1)
		go func() {
			defer wait.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			height := max(1, bounds.Dy()*width/bounds.Dx())
			destination := image.NewNRGBA(image.Rect(0, 0, width, height))
			if width == bounds.Dx() {
				draw.Draw(destination, destination.Bounds(), source, bounds.Min, draw.Src)
			} else {
				xdraw.CatmullRom.Scale(destination, destination.Bounds(), source, bounds, draw.Src, nil)
			}
			var output bytes.Buffer
			if err := webp.Encode(&output, destination, webp.Options{Quality: WebPQuality, Method: 4}); err != nil {
				results <- result{err: fmt.Errorf("encode width %d: %w", width, err)}
				return
			}
			digest := sha256.Sum256(output.Bytes())
			results <- result{variant: mediaapp.GeneratedVariant{
				Data: output.Bytes(), Width: width, Height: height,
				SHA256: hex.EncodeToString(digest[:]),
			}}
		}()
	}
	wait.Wait()
	close(results)
	variants := make([]mediaapp.GeneratedVariant, 0, len(targets))
	for item := range results {
		if item.err != nil {
			return nil, skipped, item.err
		}
		variants = append(variants, item.variant)
	}
	sort.Slice(variants, func(left, right int) bool { return variants[left].Width < variants[right].Width })
	return variants, skipped, nil
}

func detectFormat(source []byte) (string, error) {
	switch {
	case len(source) >= 8 && bytes.Equal(source[:8], []byte("\x89PNG\r\n\x1a\n")):
		return "png", nil
	case len(source) >= 3 && source[0] == 0xff && source[1] == 0xd8 && source[2] == 0xff:
		return "jpeg", nil
	case len(source) >= 12 && bytes.Equal(source[:4], []byte("RIFF")) && bytes.Equal(source[8:12], []byte("WEBP")):
		if animatedWebP(source) {
			return "", errors.New("animated WebP is not supported")
		}
		return "webp", nil
	default:
		return "", errors.New("only JPEG, PNG, and static WebP are supported")
	}
}

func decodeConfig(format string, source []byte) (image.Config, error) {
	reader := bytes.NewReader(source)
	switch format {
	case "jpeg":
		return jpeg.DecodeConfig(reader)
	case "png":
		return png.DecodeConfig(reader)
	case "webp":
		return webp.DecodeConfig(reader)
	default:
		return image.Config{}, errors.New("unsupported image format")
	}
}

func decode(format string, source []byte) (image.Image, error) {
	reader := bytes.NewReader(source)
	switch format {
	case "jpeg":
		return jpeg.Decode(reader)
	case "png":
		return png.Decode(reader)
	case "webp":
		return webp.Decode(reader, webp.Options{AutoRotate: true})
	default:
		return nil, errors.New("unsupported image format")
	}
}

func validateDimensions(width, height int) error {
	if width < 1 || height < 1 || width > MaxSide || height > MaxSide || int64(width)*int64(height) > MaxPixels {
		return errors.New("media dimensions exceed the 6000 side or 12.5 megapixel limit")
	}
	return nil
}

func animatedWebP(source []byte) bool {
	for offset := 12; offset+8 <= len(source); {
		name := source[offset : offset+4]
		size := int(binary.LittleEndian.Uint32(source[offset+4 : offset+8]))
		if bytes.Equal(name, []byte("ANIM")) || bytes.Equal(name, []byte("ANMF")) {
			return true
		}
		if size < 0 || offset+8+size > len(source) {
			return false
		}
		offset += 8 + size + size%2
	}
	return false
}

func jpegOrientation(source []byte) int {
	if len(source) < 4 {
		return 1
	}
	for offset := 2; offset+4 <= len(source); {
		if source[offset] != 0xff {
			return 1
		}
		marker := source[offset+1]
		offset += 2
		if marker == 0xd9 || marker == 0xda {
			return 1
		}
		if offset+2 > len(source) {
			return 1
		}
		length := int(binary.BigEndian.Uint16(source[offset : offset+2]))
		if length < 2 || offset+length > len(source) {
			return 1
		}
		segment := source[offset+2 : offset+length]
		if marker == 0xe1 && len(segment) >= 14 && bytes.Equal(segment[:6], []byte("Exif\x00\x00")) {
			return tiffOrientation(segment[6:])
		}
		offset += length
	}
	return 1
}

func tiffOrientation(tiff []byte) int {
	if len(tiff) < 8 {
		return 1
	}
	var order binary.ByteOrder
	switch string(tiff[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return 1
	}
	if order.Uint16(tiff[2:4]) != 42 {
		return 1
	}
	ifd := int(order.Uint32(tiff[4:8]))
	if ifd < 0 || ifd+2 > len(tiff) {
		return 1
	}
	entries := int(order.Uint16(tiff[ifd : ifd+2]))
	for index := 0; index < entries; index++ {
		offset := ifd + 2 + index*12
		if offset+12 > len(tiff) {
			return 1
		}
		entry := tiff[offset : offset+12]
		if order.Uint16(entry[:2]) == 0x0112 && order.Uint16(entry[2:4]) == 3 && order.Uint32(entry[4:8]) == 1 {
			orientation := int(order.Uint16(entry[8:10]))
			if orientation >= 1 && orientation <= 8 {
				return orientation
			}
		}
	}
	return 1
}

func applyOrientation(source image.Image, orientation int) image.Image {
	bounds := source.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if orientation < 2 || orientation > 8 {
		return source
	}
	destinationWidth, destinationHeight := width, height
	if orientation >= 5 {
		destinationWidth, destinationHeight = height, width
	}
	destination := image.NewNRGBA(image.Rect(0, 0, destinationWidth, destinationHeight))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			var targetX, targetY int
			switch orientation {
			case 2:
				targetX, targetY = width-1-x, y
			case 3:
				targetX, targetY = width-1-x, height-1-y
			case 4:
				targetX, targetY = x, height-1-y
			case 5:
				targetX, targetY = y, x
			case 6:
				targetX, targetY = height-1-y, x
			case 7:
				targetX, targetY = height-1-y, width-1-x
			case 8:
				targetX, targetY = y, width-1-x
			}
			destination.Set(targetX, targetY, source.At(bounds.Min.X+x, bounds.Min.Y+y))
		}
	}
	return destination
}
