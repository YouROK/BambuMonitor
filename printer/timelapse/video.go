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

func (t *Timelapse) AssemblePreview(folderName string) error {
	savePath := t.core.GetConfig().Timelapse.SavePath
	fullPath := filepath.Join(savePath, folderName)
	inputFile := filepath.Join(fullPath, "timelapse.mp4")
	outputFile := filepath.Join(fullPath, "preview.mp4")

	// Используем уникальный ключ блокировки для превью, чтобы не мешать сборке основного видео
	lockKey := folderName + "_preview"
	if _, loading := t.assembling.LoadOrStore(lockKey, true); loading {
		return fmt.Errorf("сборка этого превью уже запущена")
	}
	defer t.assembling.Delete(lockKey)

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return fmt.Errorf("папка не найдена")
	}

	// Проверяем, что исходный timelapse.mp4 на месте
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		return fmt.Errorf("исходное видео timelapse.mp4 не найдено")
	}

	// Команда сборки оптимизированного превью из оригинального видео
	cmd := exec.Command("ffmpeg",
		"-y",
		"-i", inputFile,
		"-vf", "select='not(mod(n,5))',setpts=0.2*PTS,scale=320:-2",
		"-c:v", "libx264",
		"-pix_fmt", "yuv420p",
		"-preset", "ultrafast",
		"-crf", "32",
		outputFile,
	)

	log.Printf("[Timelapse] Старт сборки превью: %s", folderName)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("FFmpeg preview error: %v\noutput:\n%s", err, string(output))
	}

	log.Printf("[Timelapse] Сборка превью завершена: %s", outputFile)
	return nil
}
