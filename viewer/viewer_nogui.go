//go:build !gui

package viewer

import (
	"image"
	"log"
)

// FramebufferViewer provides a no-op implementation when GUI is disabled
type FramebufferViewer struct {
	initialized bool
	running     bool
}

func NewFramebufferViewer(title string, width, height int) (*FramebufferViewer, error) {
	log.Printf("GUI viewer disabled (built without 'gui' tag). Title: %s, Size: %dx%d", title, width, height)
	return &FramebufferViewer{
		initialized: true,
	}, nil
}

func (v *FramebufferViewer) Start() {
	if !v.initialized {
		log.Println("Warning: FramebufferViewer not initialized")
		return
	}
	
	v.running = true
	log.Println("GUI viewer started (no-op mode)")
}

func (v *FramebufferViewer) UpdateFramebuffer(img image.Image) {
	// No-op when GUI is disabled
}

func (v *FramebufferViewer) IsRunning() bool {
	return v.running
}

func (v *FramebufferViewer) Initialize(title string, width, height int) {
	log.Printf("GUI viewer initialize (no-op). Title: %s, Size: %dx%d", title, width, height)
}

func (v *FramebufferViewer) Show() {
	v.running = true
	log.Println("GUI viewer show (no-op)")
}

func (v *FramebufferViewer) ShowAndRun() {
	v.running = true
	log.Println("GUI viewer show and run (no-op)")
}

func (v *FramebufferViewer) Close() {
	if v.running {
		v.running = false
		log.Println("GUI viewer closed")
	}
}

func RunWithVNCClient(title string, width, height int, vncClientFunc func(*FramebufferViewer)) {
	log.Printf("GUI viewer disabled (built without 'gui' tag). Running VNC client without GUI. Title: %s, Size: %dx%d", title, width, height)
	
	viewer := &FramebufferViewer{
		initialized: true,
		running:     true,
	}
	
	// Run VNC client function directly (no GUI)
	vncClientFunc(viewer)
}