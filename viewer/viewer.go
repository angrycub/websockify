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

	return viewer, nil
}

func (v *FramebufferViewer) Initialize(title string, width, height int) {
	// When running with RunWithVNCClient, the window is already initialized
	// This method can be used to update the title and size if needed
	if v.window != nil {
		fyne.Do(func() {
			v.window.SetTitle(title)
			v.window.Resize(fyne.NewSize(float32(width), float32(height)))
		})
	}
	log.Printf("GUI viewer updated: %dx%d", width, height)
}

func (v *FramebufferViewer) UpdateFramebuffer(img image.Image) {
	if !v.initialized || !v.running {
		return
	}
	
	select {
	case v.updateChan <- img:
		// Update queued
	default:
		// Channel full, skip this update
	}
}

func (v *FramebufferViewer) Show() {
	if !v.initialized {
		return
	}
	
	v.running = true
	// Window is already shown by RunWithVNCClient, just start updates
	// Start update goroutine
	go v.handleUpdates()
}

func (v *FramebufferViewer) handleUpdates() {
	ticker := time.NewTicker(50 * time.Millisecond) // 20 FPS max
	defer ticker.Stop()
	
	var lastImage image.Image
	
	for {
		select {
		case img := <-v.updateChan:
			lastImage = img
		case <-ticker.C:
			if lastImage != nil && v.image != nil {
				// Capture the image to update
				img := lastImage
				// Use fyne.Do to safely update UI from goroutine
				fyne.Do(func() {
					v.mutex.Lock()
					if v.image != nil {
						v.image.Image = img
						v.image.Refresh()
					}
					v.mutex.Unlock()
				})
				lastImage = nil
			}
		case <-v.closeChan:
			return
		}
	}
}

func (v *FramebufferViewer) ShowAndRun() {
	if !v.initialized {
		return
	}
	
	v.running = true
	go v.handleUpdates()
	v.window.ShowAndRun()
}

func (v *FramebufferViewer) Close() {
	if !v.initialized {
		return
	}
	
	v.running = false
	
	select {
	case v.closeChan <- true:
	default:
	}
	
	v.window.Close()
}

func (v *FramebufferViewer) SetTitle(title string) {
	if !v.initialized {
		return
	}
	
	v.window.SetTitle(title)
}

// RunWithVNCClient runs the GUI with a VNC client function
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