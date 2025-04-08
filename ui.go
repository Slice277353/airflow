package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/g3n/engine/app"
	"github.com/g3n/engine/math32"

	"github.com/g3n/engine/camera"
	"github.com/g3n/engine/core"
	"github.com/g3n/engine/gui"
	"github.com/g3n/engine/window"
)

func openFileDialog(string, error) {
	// Placeholder implementation for file dialog
	// Replace this with actual file dialog logic for your platform
	return "/path/to/selected/file.obj", nil
}

func initializeUI(scene *core.Node, windSources []WindSource, ml *ModelLoader, cam camera.ICamera) {
	// Toggle wind button
	windEnabled := false
	btn := gui.NewButton("Wind OFF")
	btn.SetPosition(100, 40)
	btn.SetSize(80, 40)
	btn.Subscribe(gui.OnClick, func(name string, ev interface{}) {
		windEnabled = !windEnabled
		if windEnabled {
			btn.Label.SetText("Wind ON")
		} else {
			btn.Label.SetText("Wind OFF")
		}
	})
	scene.Add(btn)

	// Import model button
	importBtn := gui.NewButton("Import Model")
	importBtn.SetSize(120, 40)
	importBtn.SetPosition(100, 100)
	importBtn.Subscribe(gui.OnClick, func(name string, ev interface{}) {
		filePath, err := openFileDialog()
		if err != nil || filePath == "" {
			log.Println("No file selected or error:", err)
			return
		}

		log.Println("Selected file:", filePath)

		// Remove old model
		if mesh != nil {
			scene.Remove(mesh)
			mesh = nil
		}
		ml.models = nil

		// Load new model
		if err := ml.LoadModel(filePath); err != nil {
			log.Println("Error loading model:", err)
			return
		}

		if len(ml.models) > 0 {
			mesh = ml.models[0]
			scene.Add(mesh)
			mesh.SetPosition(0, 1, 0)
			log.Printf("New model loaded and added to scene at position: %v", mesh.Position())
		} else {
			log.Println("No models were loaded.")
			mesh = nil
		}
	})
	scene.Add(importBtn)

	// Add wind source button
	addWindBtn := gui.NewButton("Add Wind Source")
	addWindBtn.SetSize(120, 40)
	addWindBtn.SetPosition(100, 160)
	waitingForWindPlacement := false
	addWindBtn.Subscribe(gui.OnClick, func(name string, ev interface{}) {
		waitingForWindPlacement = true
		log.Println("Click on the scene to place the wind source")
	})
	scene.Add(addWindBtn)

	// Handle mouse click for wind source placement
	app.App().Subscribe(window.OnMouseDown, func(evname string, ev interface{}) {
		if !waitingForWindPlacement {
			return
		}

		mev := ev.(*window.MouseEvent)
		if mev.Button != window.MouseButtonLeft {
			return
		}

		// Get the mouse position in normalized device coordinates
		w, h := app.App().GetSize()
		x := float32(mev.Xpos)/float32(w)*2 - 1
		y := -(float32(mev.Ypos)/float32(h)*2 - 1)

		// Get the projection and view matrices
		projMatrix := &math32.Matrix4{}
		viewMatrix := &math32.Matrix4{}
		cam.ProjMatrix(projMatrix)
		cam.ViewMatrix(viewMatrix)

		// Compute the combined view-projection matrix
		viewProjMatrix := &math32.Matrix4{}
		viewProjMatrix.MultiplyMatrices(projMatrix, viewMatrix)

		// Compute the inverse of the view-projection matrix
		invViewProjMatrix := &math32.Matrix4{}
		err := invViewProjMatrix.GetInverse(viewProjMatrix)
		if err != nil {
			log.Println("Failed to invert view-projection matrix")
			return
		}

		// Define near and far points in NDC
		nearNDC := math32.NewVector4(x, y, 0, 1)
		farNDC := math32.NewVector4(x, y, 1, 1)

		// Transform to world coordinates
		nearWorld := nearNDC.ApplyMatrix4(invViewProjMatrix)
		farWorld := farNDC.ApplyMatrix4(invViewProjMatrix)

		// Perspective divide
		near := math32.NewVector3(nearWorld.X/nearWorld.W, nearWorld.Y/nearWorld.W, nearWorld.Z/nearWorld.W)
		far := math32.NewVector3(farWorld.X/farWorld.W, farWorld.Y/farWorld.W, farWorld.Z/farWorld.W)

		// Compute ray direction
		direction := far.Sub(near).Normalize()

		// Compute intersection with the ground plane (y=0)
		origin := cam.(*camera.Camera).GetNode().Position()
		t := -origin.Y / direction.Y
		if t < 0 {
			log.Println("No intersection with ground plane")
			return
		}

		// Compute intersection point
		intersectPoint := origin.Add(direction.MultiplyScalar(t))

		// Add wind source at the intersection point
		windSources = addWindSource(windSources, scene, *intersectPoint)
		log.Printf("Wind source added at position: %v", intersectPoint)
		waitingForWindPlacement = false
	})

	// Numeric inputs for global parameters
	massInput := createNumericInput(mass, 100, 220, func(value float32) {
		mass = value
	})
	scene.Add(massInput)

	dragInput := createNumericInput(dragCoefficient, 100, 280, func(value float32) {
		dragCoefficient = value
	})
	scene.Add(dragInput)

	// Numeric inputs for wind source speeds
	for i, wind := range windSources {
		windSpeedInput := createNumericInput(wind.Speed, 100, 340+float32(i*60), func(value float32) {
			windSources[i].Speed = value
		})
		scene.Add(windSpeedInput)
	}
}

func createNumericInput(defaultValue float32, x, y float32, onChange func(value float32)) *gui.Edit {
	textInput := gui.NewEdit(100, fmt.Sprintf("%.2f", defaultValue))
	textInput.SetPosition(x, y)

	textInput.Subscribe(gui.OnChange, func(name string, ev interface{}) {
		text := textInput.Text()
		filteredText := filterNumericInput(text)
		if text != filteredText {
			textInput.SetText(filteredText)
		}
	})

	textInput.Subscribe(gui.OnKeyDown, func(name string, ev interface{}) {
		kev := ev.(*window.KeyEvent)
		if kev.Key == window.KeyEnter {
			text := textInput.Text()
			if value, err := strconv.ParseFloat(text, 32); err == nil && value > 0 {
				onChange(float32(value))
			} else {
				textInput.SetText(fmt.Sprintf("%.2f", defaultValue))
			}
		}
	})

	return textInput
}

func filterNumericInput(input string) string {
	var builder strings.Builder
	dotCount := 0

	for i, char := range input {
		if char >= '0' && char <= '9' {
			builder.WriteRune(char)
		} else if char == '.' && dotCount == 0 {
			builder.WriteRune(char)
			dotCount++
		} else if char == '-' && i == 0 {
			builder.WriteRune(char)
		}
	}

	return builder.String()
}
