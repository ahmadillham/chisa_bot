package main

import (
	"chisa_bot/internal/services"
	"chisa_bot/pkg/utils"
	"fmt"
	"os"
)

func main() {
	f := services.NewFFmpegService()
	data, err := os.ReadFile("dummy.jpg")
	if err != nil {
		panic(err)
	}

	webp, err := f.ImageToWebP(data)
	if err != nil {
		fmt.Println("ERROR:", err)
		os.Exit(1)
	}
	
	webp, err = utils.AddStickerExif(webp, "Pack", "Author")
	if err != nil {
		fmt.Println("EXIF ERROR:", err)
		os.Exit(1)
	}

	fmt.Printf("Success! WebP + EXIF size: %d bytes\n", len(webp))
}
