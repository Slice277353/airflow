package main

import (
	"fmt"
	"github.com/g3n/engine/app"
	"log"
	"strconv"
	"strings"

	"github.com/g3n/engine/core"
	"github.com/g3n/engine/gui"
	"github.com/g3n/engine/window"
)

func initializeUI(scene *core.Node, windSources []WindSource, ml *ModelLoader) {
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

	emptyBtn := gui.NewButton("Import an object")
	emptyBtn.SetSize(120, 40)
	scene.Add(emptyBtn)

	updateButtonLayout := func(w, h int) {
		const minWidth, minHeight = 400, 200
		if w < minWidth || h < minHeight {
			emptyBtn.SetVisible(false)
			return
		}
		emptyBtn.SetVisible(true)

		btnWidth := float32(w) * 0.15
		btnHeight := float32(h) * 0.05
		btnX := float32(w) - btnWidth - float32(w)*0.05
		btnY := float32(h) * 0.1

		emptyBtn.SetSize(btnWidth, btnHeight)
		emptyBtn.SetPosition(btnX, btnY)
	}

	app.App().Subscribe(window.OnWindowSize, func(evname string, ev interface{}) {
		w, h := app.App().GetSize()
		updateButtonLayout(w, h)
	})

	w, h := app.App().GetSize()
	updateButtonLayout(w, h)

	emptyBtn.Subscribe(gui.OnClick, func(name string, ev interface{}) {
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
			// Set position directly (remove centering logic for now)
			mesh.SetPosition(0, 1, 0)
			log.Printf("New mesh loaded and added to scene at position: %v", mesh.Position())
		} else {
			log.Println("No models were loaded.")
			mesh = nil
		}
	})

	// Use global mass and dragCoefficient from physics.go
	massInput := createNumericInput(mass, 100, 100, func(value float32) {
		mass = value
	})
	scene.Add(massInput)

	dragInput := createNumericInput(dragCoefficient, 100, 150, func(value float32) {
		dragCoefficient = value
	})
	scene.Add(dragInput)

	for i, wind := range windSources {
		windSpeedInput := createNumericInput(wind.Speed, 100, 200+float32(i*50), func(value float32) {
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
