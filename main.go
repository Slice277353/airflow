package main

import (
	"time"

	"github.com/g3n/engine/app"
	"github.com/g3n/engine/camera"
	"github.com/g3n/engine/core"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/gui"
	"github.com/g3n/engine/light"
	"github.com/g3n/engine/math32"
	"github.com/g3n/engine/renderer"
	"github.com/g3n/engine/util/helper"
	"github.com/g3n/engine/window"
)

var scene *core.Node
var windEnabled bool
var windSources []WindSource
var windParticles []*WindParticle

func main() {
	// Initialize the app and scene
	a := app.App()
	scene = core.NewNode()
	gui.Manager().Set(scene)
	windEnabled = true

	// Initialize the model loader
	ml := &ModelLoader{scene: scene}

	// Setup the camera
	cam := camera.New(1)
	cam.SetPosition(0, 2, 5)
	cam.LookAt(&math32.Vector3{X: 0, Y: 1, Z: 0}, &math32.Vector3{X: 0, Y: 1, Z: 0})
	scene.Add(cam)
	camera.NewOrbitControl(cam)

	// Handle window resizing
	onResize := func(evname string, ev interface{}) {
		width, height := a.GetSize()
		a.Gls().Viewport(0, 0, int32(width), int32(height))
		cam.SetAspect(float32(width) / float32(height))
	}
	a.Subscribe(window.OnWindowSize, onResize)
	onResize("", nil)

	// Add lights and helpers
	scene.Add(light.NewAmbient(&math32.Color{1, 1, 1}, 0.8))
	pointLight := light.NewPoint(&math32.Color{1, 1, 1}, 5.0)
	pointLight.SetPosition(1, 2, 2)
	scene.Add(pointLight)
	scene.Add(helper.NewAxes(1.0))

	// Initialize wind sources
	windSources = initializeWindSources(scene)

	// Initialize the UI
	initializeUI(scene, windSources, ml, cam)

	// Application loop
	lastParticleTime := time.Now()
	a.Run(func(renderer *renderer.Renderer, deltaTime time.Duration) {
		a.Gls().Clear(gls.DEPTH_BUFFER_BIT | gls.STENCIL_BUFFER_BIT | gls.COLOR_BUFFER_BIT)
		renderer.Render(scene, cam)

		// Generate wind particles
		if windEnabled && time.Since(lastParticleTime).Milliseconds() >= 100 {
			for _, wind := range windSources {
				windParticles = append(windParticles, createWindParticle(wind.Position, wind.Direction))
			}
			lastParticleTime = time.Now()
		}

		// Update wind particles
		importedModel := ml.GetLoadedModel()
		for _, particle := range windParticles {
			updatePhysics(particle, importedModel, float32(deltaTime.Seconds()))
		}
		updateWindParticles(float32(deltaTime.Seconds()), scene, importedModel)
	})
}
