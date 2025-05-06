package main

import (
	"fmt"
	"log"
	"strconv"

	"github.com/g3n/engine/math32"

	localcam "github.com/g3n/demos/hellog3n/camera"
	"github.com/g3n/engine/camera"
	"github.com/g3n/engine/gui"
	"github.com/g3n/engine/window"
)

func initializeUI(panel *gui.Panel, windSources *[]WindSource, ml *ModelLoader, cam camera.ICamera) {
	// Toggle wind button
	btn := gui.NewButton("Wind OFF")
	btn.SetPosition(10, 10)
	btn.SetSize(80, 30)
	btn.Subscribe(gui.OnClick, func(name string, ev interface{}) {
		windEnabled = !windEnabled
		if windEnabled {
			btn.Label.SetText("Wind ON")
			// Initialize the fluid simulation when wind is turned on
			initializeFluidSimulation(scene, *windSources)
		} else {
			btn.Label.SetText("Wind OFF")
			// Clean up everything when turned off
			for _, p := range windParticles {
				if p != nil && p.Mesh != nil {
					scene.Remove(p.Mesh)
				}
			}
			windParticles = nil
			clearFluidParticles(scene)
		}
	})
	panel.Add(btn)

	// Import model button
	importBtn := gui.NewButton("Import Model")
	importBtn.SetSize(120, 30)
	importBtn.SetPosition(10, 50)
	importBtn.Subscribe(gui.OnClick, func(name string, ev interface{}) {
		filePath, err := openModelFileDialog()
		if err != nil || filePath == "" {
			log.Println("No file selected or error:", err)
			return
		}

		if mesh != nil {
			scene.Remove(mesh)
			mesh = nil
		}
		ml.models = nil

		if err := ml.LoadModel(filePath); err != nil {
			log.Println("Error loading model:", err)
			return
		}

		if len(ml.models) > 0 {
			mesh = ml.models[0]
			scene.Add(mesh)
			mesh.SetPosition(0, 1, 0)
		}
	})
	panel.Add(importBtn)

	// Add wind source button with mouse placement
	addWindBtn := gui.NewButton("Add Wind Source")
	addWindBtn.SetSize(120, 30)
	addWindBtn.SetPosition(10, 90)
	addWindBtn.Subscribe(gui.OnClick, func(name string, ev interface{}) {
		// Create a function to handle mouse click for placement
		mouseHandler := func(evname string, ev interface{}) {
			mev := ev.(*window.MouseEvent)

			// Create a ray from the camera through the mouse position
			width, height := window.Get().GetSize()
			x := 2.0*float32(mev.Xpos)/float32(width) - 1.0
			y := -2.0*float32(mev.Ypos)/float32(height) + 1.0

			ray := localcam.NewRayFromMouse(cam, x, y)

			// Calculate intersection with y=1 plane (ground plane)
			groundNormal := &math32.Vector3{X: 0, Y: 1, Z: 0}
			groundPoint := &math32.Vector3{X: 0, Y: 1, Z: 0}

			// Get ray vectors
			rayOrigin := ray.Origin()
			rayDir := ray.Direction()

			// Calculate intersection using plane equation
			denom := rayDir.Dot(groundNormal)
			if math32.Abs(denom) > 1e-6 {
				p0l0 := groundPoint.Clone().Sub(&rayOrigin)
				t := p0l0.Dot(groundNormal) / denom
				if t >= 0 {
					// Calculate intersection point
					intersectPoint := rayOrigin.Clone()
					directionScaled := rayDir.Clone().MultiplyScalar(t)
					intersectPoint.Add(directionScaled)

					// Add the wind source at the intersection point
					*windSources = addWindSource(*windSources, scene, *intersectPoint)
					updateWindControls(panel, windSources)
				}
			}

			// Remove the mouse handler after placement
			window.Get().UnsubscribeID(window.OnMouseDown, "wind_source_placement")
		}

		// Subscribe to mouse click with an ID for later removal
		window.Get().SubscribeID(window.OnMouseDown, "wind_source_placement", mouseHandler)
	})
	panel.Add(addWindBtn)

	updateWindControls(panel, windSources)
}

func updateWindControls(panel *gui.Panel, windSources *[]WindSource) {
	// Remove existing controls by getting rid of all children after index 3
	children := panel.Children()
	for i := len(children) - 1; i >= 4; i-- {
		if guiChild, ok := children[i].(gui.IPanel); ok {
			panel.Remove(guiChild)
		}
	}
	y := float32(130)

	// Add controls for each wind source
	for i := range *windSources {
		// Source label
		label := gui.NewLabel(fmt.Sprintf("Wind Source %d", i+1))
		label.SetPosition(10, y)
		panel.Add(label)
		y += 25

		// Speed control
		speedLabel := gui.NewLabel("Speed:")
		speedLabel.SetPosition(20, y)
		panel.Add(speedLabel)

		speedInput := gui.NewEdit(60, fmt.Sprintf("%.1f", (*windSources)[i].Speed))
		speedInput.SetPosition(80, y)
		speedInput.Subscribe(gui.OnChange, func(name string, ev interface{}) {
			if val, err := strconv.ParseFloat(speedInput.Text(), 32); err == nil {
				(*windSources)[i].Speed = float32(val)
			}
		})
		panel.Add(speedInput)
		y += 25

		// Temperature control
		tempLabel := gui.NewLabel("Temp:")
		tempLabel.SetPosition(20, y)
		panel.Add(tempLabel)

		tempInput := gui.NewEdit(60, fmt.Sprintf("%.1f", (*windSources)[i].Temperature))
		tempInput.SetPosition(80, y)
		tempInput.Subscribe(gui.OnChange, func(name string, ev interface{}) {
			if val, err := strconv.ParseFloat(tempInput.Text(), 32); err == nil {
				(*windSources)[i].Temperature = float32(val)
			}
		})
		panel.Add(tempInput)
		y += 25

		// Direction controls
		dirLabel := gui.NewLabel("Direction:")
		dirLabel.SetPosition(20, y)
		panel.Add(dirLabel)
		y += 25

		// X direction
		xInput := gui.NewEdit(40, fmt.Sprintf("%.1f", (*windSources)[i].Direction.X))
		xInput.SetPosition(30, y)
		panel.Add(xInput)

		// Y direction
		yInput := gui.NewEdit(40, fmt.Sprintf("%.1f", (*windSources)[i].Direction.Y))
		yInput.SetPosition(80, y)
		panel.Add(yInput)

		// Z direction
		zInput := gui.NewEdit(40, fmt.Sprintf("%.1f", (*windSources)[i].Direction.Z))
		zInput.SetPosition(130, y)
		panel.Add(zInput)

		// Update direction handler
		updateDirFunc := func() {
			x, _ := strconv.ParseFloat(xInput.Text(), 32)
			y, _ := strconv.ParseFloat(yInput.Text(), 32)
			z, _ := strconv.ParseFloat(zInput.Text(), 32)
			dir := math32.NewVector3(float32(x), float32(y), float32(z))
			if dir.Length() > 0 {
				dir.Normalize()
				(*windSources)[i].Direction = *dir
			}
		}

		xInput.Subscribe(gui.OnChange, func(name string, ev interface{}) { updateDirFunc() })
		yInput.Subscribe(gui.OnChange, func(name string, ev interface{}) { updateDirFunc() })
		zInput.Subscribe(gui.OnChange, func(name string, ev interface{}) { updateDirFunc() })

		y += 40
	}
}

func createNumericInput(defaultValue float32, x, y float32, onChange func(value float32)) *gui.Edit {
	textInput := gui.NewEdit(100, fmt.Sprintf("%.2f", defaultValue))
	textInput.SetPosition(x, y)

	textInput.Subscribe(gui.OnChange, func(name string, ev interface{}) {
		text := textInput.Text()
		if value, err := strconv.ParseFloat(text, 32); err == nil {
			onChange(float32(value))
		}
	})

	return textInput
}
