package main
import (
	"fmt"
	"os/exec"
)
func main() {
	sizes := []int{72, 60, 54, 48, 40}
	for _, size := range sizes {
		filter := fmt.Sprintf("color=white:s=512x512:d=1,drawtext=fontfile='/usr/share/fonts/TTF/DejaVuSans-Bold.ttf':text='TERUS TERANG':fontcolor=white:fontsize=%d:bordercolor=black:borderw=%d:x=(w-text_w)/2:y=(h-text_h)/2", size, size/12)
		cmd := exec.Command("ffmpeg", "-f", "lavfi", "-i", filter, "-vframes", "1", "-y", fmt.Sprintf("test_size_%d.jpg", size))
		cmd.Run()
	}
}
