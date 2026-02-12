package timelapse

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

func (t *Timelapse) AssembleVideo(folderName string) error {
	savePath := t.core.GetConfig().Timelapse.SavePath
	fullPath := filepath.Join(savePath, folderName)
	outputFile := filepath.Join(fullPath, "timelapse.mp4")

	if _, loading := t.assembling.LoadOrStore(folderName, true); loading {
		return fmt.Errorf("сборка этого видео уже запущена")
	}
	defer t.assembling.Delete(folderName)

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return fmt.Errorf("папка не найдена")
	}

	fps := t.core.GetConfig().Timelapse.Fps
	if fps <= 0 {
		fps = 20
	}

	cmd := exec.Command("ffmpeg",
		"-y",
		"-framerate", fmt.Sprintf("%d", fps),
		"-pattern_type", "glob",
		"-i", filepath.Join(fullPath, "layer_*.jpg"),
		"-c:v", "libx264",
		"-pix_fmt", "yuv420p",
		"-profile:v", "high",
		"-level", "4.1",
		"-crf", "23",
		outputFile,
	)

	log.Printf("[Timelapse] Старт сборки: %s (FPS: %d)", folderName, fps)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("FFmpeg error: %v\noutput:\n%s", err, string(output))
	}

	log.Printf("[Timelapse] Сборка завершена: %s", outputFile)
	return nil
}
