package printer

import (
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"time"
)

type BambuCamera struct {
	core     Core
	stopChan chan struct{}
}

func NewBambuCamera(core Core) *BambuCamera {
	return &BambuCamera{
		core:     core,
		stopChan: make(chan struct{}),
	}
}

func (b *BambuCamera) Start() {
	go b.run()
}

func (b *BambuCamera) Stop() {
	close(b.stopChan)
}

func (b *BambuCamera) run() {
	username := "bblp"
	port := 6000

	// Подготовка бинарной аутентификации (80 байт)
	authData := make([]byte, 80)
	binary.LittleEndian.PutUint32(authData[0:4], 0x40)   // Magic
	binary.LittleEndian.PutUint32(authData[4:8], 0x3000) // Command
	copy(authData[16:48], username)
	copy(authData[48:80], b.core.GetConfig().Printer.Password)

	for {
		select {
		case <-b.stopChan:
			return
		default:
			log.Printf("[Camera] Connecting to %s:%d", b.core.GetConfig().Printer.Hostname, port)
			conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", b.core.GetConfig().Printer.Hostname, port), 5*time.Second)
			if err != nil {
				log.Printf("[Camera] Connection failed: %v", err)
				b.core.UpdateFrame(nil, 0)
				b.core.SetOnline(false)
				time.Sleep(5 * time.Second)
				continue
			}

			b.handleConnection(conn, authData)
			time.Sleep(2 * time.Second)
		}
	}
}

func (b *BambuCamera) handleConnection(conn net.Conn, authData []byte) {
	tlsConn := tls.Client(conn, &tls.Config{
		InsecureSkipVerify: true,
	})

	defer tlsConn.Close()

	if _, err := tlsConn.Write(authData); err != nil {
		return
	}

	frames := 0
	startTime := time.Now()

	log.Printf("[Camera] Start reading camera...")
	b.core.SetOnline(true)
	for {
		// Установка таймаута за какое время должен прочитать
		tlsConn.SetReadDeadline(time.Now().Add(5 * time.Second))
		// 1. Читаем 16-байтный заголовок Bambu
		header := make([]byte, 16)
		if _, err := io.ReadFull(tlsConn, header); err != nil {
			log.Printf("[Camera] Read header error: %v", err)
			return
		}

		// 2. Достаем размер JPEG (первые 4 байта, little endian)
		imgSize := binary.LittleEndian.Uint32(header[0:4]) & 0x00FFFFFF
		if imgSize == 0 || imgSize > 2*1024*1024 { // Лимит 2МБ для безопасности
			return
		}

		// 3. Читаем тело JPEG
		imgBuf := make([]byte, imgSize)
		if _, err := io.ReadFull(tlsConn, imgBuf); err != nil {
			return
		}

		// 4. Считаем FPS
		frames++
		now := time.Now()
		diff := now.Sub(startTime).Seconds()
		var currentFps float64
		if diff >= 1 {
			currentFps = float64(frames) / diff
			frames = 0
			startTime = now
		}

		// 5. Отдаем кадр в App
		b.core.UpdateFrame(imgBuf, currentFps)

		// Искусственная задержка из конфига
		if b.core.GetConfig().Printer.EncodeWait > 0 {
			time.Sleep(time.Millisecond * time.Duration(b.core.GetConfig().Printer.EncodeWait))
		}
	}
}
