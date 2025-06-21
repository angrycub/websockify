//go:build gui

package viewer

import (
	"image"
	"log"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
)

type FramebufferViewer struct {
	app         fyne.App
	window      fyne.Window
	image       *canvas.Image
	mutex       sync.RWMutex
	updateChan  chan image.Image
	closeChan   chan bool
	initialized bool
	running     bool
}

func NewFramebufferViewer(title string, width, height int) (*FramebufferViewer, error) {
	viewer := &FramebufferViewer{
		updateChan: make(chan image.Image, 10),
		closeChan:  make(chan bool, 1),
	}

	// Initialize Fyne app
	viewer.app = app.New()
	viewer.window = viewer.app.NewWindow(title)
	viewer.window.Resize(fyne.NewSize(float32(width), float32(height)))

	// Create initial blank image
	blankImg := image.NewRGBA(image.Rect(0, 0, width, height))
	viewer.image = canvas.NewImageFromImage(blankImg)
	viewer.image.FillMode = canvas.ImageFillOriginal

	// Set up the window content
	content := container.NewVBox(viewer.image)
	viewer.window.SetContent(content)

	viewer.initialized = true
	return viewer, nil
}

func (v *FramebufferViewer) Start() {
	if !v.initialized {
		log.Println("Warning: FramebufferViewer not initialized")
		return
	}

	v.mutex.Lock()
	if v.running {
		v.mutex.Unlock()
		return
	}
	v.running = true
	v.mutex.Unlock()

	// Start the update goroutine
	go v.updateLoop()

	// Show the window and start the GUI loop (this blocks)
	go func() {
		v.window.ShowAndRun()
		v.closeChan <- true
	}()
}

func (v *FramebufferViewer) UpdateFramebuffer(img image.Image) {
	if !v.initialized || !v.running {
		return
	}

	select {
	case v.updateChan <- img:
		// Image queued for update
	default:
		// Channel full, skip this frame
	}
}

func (v *FramebufferViewer) updateLoop() {
	ticker := time.NewTicker(16 * time.Millisecond) // ~60 FPS
	defer ticker.Stop()

	for {
		select {
		case img := <-v.updateChan:
			v.image.Image = img
			canvas.Refresh(v.image)

		case <-ticker.C:
			// Periodic refresh even if no new frames

		case <-v.closeChan:
			v.mutex.Lock()
			v.running = false
			v.mutex.Unlock()
			return
		}
	}
}

func (v *FramebufferViewer) IsRunning() bool {
	v.mutex.RLock()
	defer v.mutex.RUnlock()
	return v.running
}

func (v *FramebufferViewer) Initialize(title string, width, height int) {
	// When running with RunWithVNCClient, the window is already initialized
	// This method can be used to update the title and size if needed
	if v.window != nil {
		v.window.SetTitle(title)
		v.window.Resize(fyne.NewSize(float32(width), float32(height)))
	}
}

func (v *FramebufferViewer) Show() {
	if !v.initialized {
		return
	}
	
	v.running = true
	if v.window != nil {
		v.window.Show()
	}
}

func (v *FramebufferViewer) ShowAndRun() {
	if !v.initialized {
		return
	}
	
	v.running = true
	if v.window != nil {
		v.window.ShowAndRun()
	}
}

func (v *FramebufferViewer) Close() {
	if !v.initialized {
		return
	}

	v.mutex.Lock()
	if !v.running {
		v.mutex.Unlock()
		return
	}
	v.mutex.Unlock()

	select {
	case v.closeChan <- true:
	default:
	}

	if v.window != nil {
		v.window.Close()
	}
}

func RunWithVNCClient(title string, width, height int, vncClientFunc func(*FramebufferViewer)) {
	// Create Fyne app on main thread
	a := app.New()
	w := a.NewWindow(title)
	w.Resize(fyne.NewSize(float32(width), float32(height)))

	img := canvas.NewImageFromResource(nil)
	img.FillMode = canvas.ImageFillOriginal
	img.ScaleMode = canvas.ImageScalePixels

	content := container.NewBorder(nil, nil, nil, nil, img)
	w.SetContent(content)

	viewer := &FramebufferViewer{
		app:         a,
		window:      w,
		image:       img,
		updateChan:  make(chan image.Image, 10),
		closeChan:   make(chan bool, 1),
		initialized: true,
		running:     true,
	}
	
	// Start VNC client in goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("VNC client panic: %v", r)
			}
		}()
		vncClientFunc(viewer)
	}()
	
	// Start update handler
	go viewer.handleUpdates()
	
	// Run GUI on main thread
	w.ShowAndRun()
}

func (v *FramebufferViewer) handleUpdates() {
	ticker := time.NewTicker(16 * time.Millisecond) // ~60 FPS
	defer ticker.Stop()

	for {
		select {
		case img := <-v.updateChan:
			v.image.Image = img
			canvas.Refresh(v.image)

		case <-ticker.C:
			// Periodic refresh even if no new frames

		case <-v.closeChan:
			v.mutex.Lock()
			v.running = false
			v.mutex.Unlock()
			return
		}
	}
}