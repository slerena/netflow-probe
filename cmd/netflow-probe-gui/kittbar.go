//go:build (darwin || windows) && cgo

package main

import (
	"image"
	"image/color"
	"math"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

// kittBar is the green sweeping spinner used across the Pandora desktop tools
// (ported from pandora-mini's kittbar.go, same visual: a green light that
// sweeps side to side). It keeps Start/Stop and Show/Hide.
type kittBar struct {
	widget.BaseWidget
	mu      sync.Mutex
	running bool
	stop    chan struct{}
	pos     float64 // 0..1 sweep position
	dir     float64 // +1 right, -1 left
}

func newKittBar() *kittBar {
	k := &kittBar{dir: 1, pos: 0.5}
	k.ExtendBaseWidget(k)
	return k
}

func (k *kittBar) Start() {
	k.mu.Lock()
	if k.running {
		k.mu.Unlock()
		return
	}
	k.running = true
	k.stop = make(chan struct{})
	stop := k.stop
	k.mu.Unlock()
	go func() {
		tk := time.NewTicker(33 * time.Millisecond)
		defer tk.Stop()
		for {
			select {
			case <-stop:
				return
			case <-tk.C:
				k.mu.Lock()
				k.pos += k.dir * 0.028
				if k.pos >= 1 {
					k.pos, k.dir = 1, -1
				} else if k.pos <= 0 {
					k.pos, k.dir = 0, 1
				}
				k.mu.Unlock()
				fyne.Do(k.Refresh)
			}
		}
	}()
}

func (k *kittBar) Stop() {
	k.mu.Lock()
	if !k.running {
		k.mu.Unlock()
		return
	}
	k.running = false
	close(k.stop)
	k.mu.Unlock()
}

func (k *kittBar) CreateRenderer() fyne.WidgetRenderer {
	return &rasterWidget{raster: canvas.NewRaster(k.render), min: fyne.NewSize(120, 6)}
}

func (k *kittBar) render(pw, ph int) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, pw, ph))
	if pw < 2 || ph < 2 {
		return img
	}
	k.mu.Lock()
	pos := k.pos
	k.mu.Unlock()

	head := pos * float64(pw)
	sigma := math.Max(float64(pw)*0.10, 6)
	bh := math.Min(float64(ph)*0.55, 6)
	if bh < 2 {
		bh = float64(ph)
	}
	y0 := (float64(ph) - bh) / 2
	green := color.NRGBA{R: 0x3d, G: 0xff, B: 0x63, A: 0xff}
	for x := 0; x < pw; x++ {
		d := (float64(x) - head) / sigma
		a := math.Exp(-d * d)
		if a < 0.03 {
			continue
		}
		col := color.NRGBA{R: green.R, G: green.G, B: green.B, A: uint8(a * 255)}
		for y := int(y0); y < int(y0+bh) && y < ph; y++ {
			if y >= 0 {
				img.SetNRGBA(x, y, col)
			}
		}
	}
	return img
}

type rasterWidget struct {
	raster *canvas.Raster
	min    fyne.Size
}

func (r *rasterWidget) Layout(s fyne.Size)           { r.raster.Resize(s) }
func (r *rasterWidget) Refresh()                     { r.raster.Refresh() }
func (r *rasterWidget) Destroy()                     {}
func (r *rasterWidget) MinSize() fyne.Size           { return r.min }
func (r *rasterWidget) Objects() []fyne.CanvasObject { return []fyne.CanvasObject{r.raster} }
