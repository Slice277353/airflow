package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/g3n/engine/app"
	"github.com/g3n/engine/camera"
	"github.com/g3n/engine/core"
	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/graphic"
	"github.com/g3n/engine/gui"
	"github.com/g3n/engine/light"
	"github.com/g3n/engine/loader/collada"
	"github.com/g3n/engine/loader/gltf"
	"github.com/g3n/engine/loader/obj"
	"github.com/g3n/engine/material"
	"github.com/g3n/engine/math32"
	"github.com/g3n/engine/renderer"
	"github.com/g3n/engine/util/helper"
	"github.com/g3n/engine/window"
)

// ModelLoader handles loading of 3D models
type ModelLoader struct {
	scene  *core.Node
	models []*core.Node
}

// Function to open a file dialog and return the selected file path
func openFileDialog() (string, error) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("powershell", "-Command", "Add-Type -AssemblyName System.Windows.Forms; "+
			"$dlg = New-Object System.Windows.Forms.OpenFileDialog; "+
			"$dlg.Filter = '3D Models (*.obj;*.gltf;*.dae)|*.obj;*.gltf;*.dae'; "+
			"$dlg.ShowDialog() | Out-Null; "+
			"Write-Output $dlg.FileName")
	case "darwin": // macOS
		cmd = exec.Command("osascript", "-e",
			`set filePath to POSIX path of (choose file with prompt "Select a 3D model" of type {"obj", "gltf", "dae"})`,
			"-e", `do shell script "echo " & quoted form of filePath`)
	case "linux":
		cmd = exec.Command("zenity", "--file-selection", "--title=Select a 3D model", "--file-filter=*.obj *.gltf *.dae")
	default:
		return "", fmt.Errorf("unsupported platform")
	}

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

func (ml *ModelLoader) LoadModel(fpath string) error {
	dir, file := filepath.Split(fpath)
	ext := filepath.Ext(file)

	switch ext {
	case ".obj":
		dec, err := obj.Decode(fpath, "")
		if err != nil {
			return err
		}
		grp, err := dec.NewGroup()
		if err != nil {
			return err
		}
		ml.scene.Add(grp)
		ml.models = append(ml.models, grp)

	case ".gltf":
		data, err := os.ReadFile(fpath)
		if err != nil {
			return err
		}
		g, err := gltf.ParseJSON(string(data))
		if err != nil {
			return err
		}
		if g.Scene != nil {
			log.Println("glTF model loaded, but node processing is required.")
			// Note: Minimal glTF support; add scene processing if needed
		} else {
			log.Println("glTF Scene undefined, check the file.")
		}

	case ".dae":
		dec, err := collada.Decode(fpath)
		if err != nil && err != io.EOF {
			return err
		}
		dec.SetDirImages(dir)
		s, err := dec.NewScene()
		if err != nil {
			return err
		}
		ml.scene.Add(s)
		ml.models = append(ml.models, s.GetNode())
	default:
		return fmt.Errorf("unknown model format: %s", ext)
	}
	return nil
}

// WindSource represents a wind generator in the scene
type WindSource struct {
	Position  math32.Vector3
	Radius    float32
	Speed     float32
	Direction math32.Vector3
}

// WindParticle stores a particle's position and velocity
type WindParticle struct {
	Mesh     *graphic.Mesh
	Velocity math32.Vector3
	Lifespan float32
	Elapsed  float32
}

var scene *core.Node
var windParticles []*WindParticle

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

func createWindParticle(position, direction math32.Vector3) *WindParticle {
	particleGeom := geometry.NewSphere(0.05, 8, 8)
	particleMat := material.NewStandard(math32.NewColor("White"))
	particleMesh := graphic.NewMesh(particleGeom, particleMat)
	particleMesh.SetPositionVec(&position)
	scene.Add(particleMesh)

	return &WindParticle{
		Mesh:     particleMesh,
		Velocity: *direction.Clone().MultiplyScalar(0.5),
		Lifespan: 2.0,
		Elapsed:  0,
	}
}

func updateWindParticles(deltaTime float32) {
	var newParticles []*WindParticle

	for _, particle := range windParticles {
		particle.Elapsed += deltaTime
		if particle.Elapsed >= particle.Lifespan {
			scene.Remove(particle.Mesh)
			continue
		}

		pos := particle.Mesh.Position()
		pos.Add(&particle.Velocity)
		particle.Mesh.SetPositionVec(&pos)
		newParticles = append(newParticles, particle)
	}

	windParticles = newParticles
}

func main() {
	a := app.App()
	scene = core.NewNode()
	var mesh *core.Node
	ml := &ModelLoader{scene: scene} // Empty ModelLoader at start
	gui.Manager().Set(scene)

	// Physics variables
	velocity := math32.NewVector3(0, 0, 0)
	var dragCoefficient float32 = 0.47
	const airDensity = 1.225
	const area = 1.0
	var mass float32 = 1.0
	const gravity = -9.8

	type SimulationData struct {
		Time            float32
		Acceleration    math32.Vector3
		WindPower       float32
		AngularMomentum math32.Vector3
		DampingEffect   float32
	}
	var simulationData []SimulationData

	// Camera setup
	cam := camera.New(1)
	cam.SetPosition(0, 5, 10)
	cam.LookAt(&math32.Vector3{X: 0, Y: 0, Z: 0}, &math32.Vector3{X: 0, Y: 1, Z: 0})
	scene.Add(cam)
	camera.NewOrbitControl(cam)

	// Window resize handling
	onResize := func(evname string, ev interface{}) {
		width, height := a.GetSize()
		a.Gls().Viewport(0, 0, int32(width), int32(height))
		cam.SetAspect(float32(width) / float32(height))
	}
	a.Subscribe(window.OnWindowSize, onResize)
	onResize("", nil)

	// Load model
	//ml := &ModelLoader{scene: scene}
	//filePath := "./cube.obj" // Adjust this path as needed
	//if _, err := os.Stat(filePath); os.IsNotExist(err) {
	//	log.Fatal("File not found: ", filePath)
	//}
	//if err := ml.LoadModel(filePath); err != nil {
	//	log.Fatal("Error loading model: ", err)
	//}
	//mesh := ml.models[0] // Use the first loaded model
	//mesh.SetPosition(0, 1, 0)

	// Create surface
	surfaceGeom := geometry.NewPlane(20, 20)
	surfaceMat := material.NewStandard(math32.NewColor("Green"))
	surfaceMesh := graphic.NewMesh(surfaceGeom, surfaceMat)
	surfaceMesh.SetRotationX(-math32.Pi / 2)
	scene.Add(surfaceMesh)

	// Wind sources
	windSources := []WindSource{
		{Position: *math32.NewVector3(2, 1, 0), Radius: 3.0, Speed: 10.0, Direction: *math32.NewVector3(-1, 0, 0).Normalize()},
		{Position: *math32.NewVector3(-2, 1, 0), Radius: 2.0, Speed: 5.0, Direction: *math32.NewVector3(1, 0, 0).Normalize()},
	}

	for _, wind := range windSources {
		sphereGeom := geometry.NewSphere(0.2, 16, 16)
		sphereMat := material.NewStandard(math32.NewColor("Red"))
		sphereMesh := graphic.NewMesh(sphereGeom, sphereMat)
		sphereMesh.SetPositionVec(&wind.Position)
		scene.Add(sphereMesh)
	}

	// UI: Wind Toggle Button
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

	// новая кнопка, ни за что не отвечает
	emptyBtn := gui.NewButton("Import an object")
	emptyBtn.SetSize(120, 40)
	scene.Add(emptyBtn)

	updateButtonLayout := func(w, h int) {
		const minWidth, minHeight = 400, 200 // Минимальные размеры окна

		if w < minWidth || h < minHeight {
			emptyBtn.SetVisible(false) // Скрываем кнопку, если окно слишком маленькое
			return
		}
		emptyBtn.SetVisible(true)

		// Адаптивные размеры и позиция
		btnWidth := float32(w) * 0.15                   // 15% от ширины окна
		btnHeight := float32(h) * 0.05                  // 5% от высоты окна
		btnX := float32(w) - btnWidth - float32(w)*0.05 // 5% отступ справа
		btnY := float32(h) * 0.1                        // 10% от верха

		emptyBtn.SetSize(btnWidth, btnHeight)
		emptyBtn.SetPosition(btnX, btnY)
	}

	a.Subscribe(window.OnWindowSize, func(evname string, ev interface{}) {
		w, h := a.GetSize() // Получаем текущие размеры окна
		updateButtonLayout(w, h)
	})

	w, h := a.GetSize()
	updateButtonLayout(w, h)

	emptyBtn.Subscribe(gui.OnClick, func(name string, ev interface{}) {
		filePath, err := openFileDialog()
		if err != nil || filePath == "" {
			log.Println("No file selected or error:", err)
			return
		}

		log.Println("Selected file:", filePath)

		// Remove old models before adding new ones
		for _, m := range ml.models {
			scene.Remove(m)
		}
		ml.models = nil

		if err := ml.LoadModel(filePath); err != nil {
			log.Println("Error loading model:", err)
			return
		}

		// Assign the first model to mesh
		if len(ml.models) > 0 {
			mesh = ml.models[0]
			mesh.SetPosition(0, 1, 0)
		} else {
			log.Println("No models were loaded.")
		}
	})

	massInput := createNumericInput(mass, 320, 100, func(value float32) {
		mass = value
	})
	massInput.SetPosition(100, 100)
	scene.Add(massInput)

	dragInput := createNumericInput(dragCoefficient, 320, 150, func(value float32) {
		dragCoefficient = value
	})
	dragInput.SetPosition(100, 150)
	scene.Add(dragInput)

	for i, wind := range windSources {
		windSpeedInput := createNumericInput(wind.Speed, 320, 200+float32(i*50), func(value float32) {
			windSources[i].Speed = value
		})
		windSpeedInput.SetPosition(100, 200+float32(i*50))
		scene.Add(windSpeedInput)
	}

	// Lights and helpers
	scene.Add(light.NewAmbient(&math32.Color{R: 1.0, G: 1.0, B: 1.0}, 0.8))
	pointLight := light.NewPoint(&math32.Color{R: 1, G: 1, B: 1}, 5.0)
	pointLight.SetPosition(1, 0, 2)
	scene.Add(pointLight)
	scene.Add(helper.NewAxes(1.0))

	a.Gls().ClearColor(0.5, 0.5, 0.5, 1.0)

	// Application loop
	a.Run(func(renderer *renderer.Renderer, deltaTime time.Duration) {
		a.Gls().Clear(gls.DEPTH_BUFFER_BIT | gls.STENCIL_BUFFER_BIT | gls.COLOR_BUFFER_BIT)
		renderer.Render(scene, cam)

		if mesh == nil {
			return
		}

		torusPos := mesh.Position()

		if windEnabled {
			totalForce := math32.NewVector3(0, 0, 0)
			angularMomentum := math32.NewVector3(0, 0, 0)
			windPower := float32(0)
			dampingEffect := float32(0.01)

			for i := range windSources {
				wind := &windSources[i]
				distanceVec := torusPos.Clone().Sub(&wind.Position)
				distance := distanceVec.Length()

				if distance <= wind.Radius {
					windVelocity := wind.Direction.Clone().MultiplyScalar(wind.Speed)
					dragMagnitude := 0.5 * airDensity * wind.Speed * wind.Speed * dragCoefficient * area
					dragForce := windVelocity.Clone().Normalize().MultiplyScalar(dragMagnitude)
					totalForce.Add(dragForce)

					windPower += dragMagnitude * wind.Speed
					angularMomentum.Add(dragForce.Cross(&torusPos))

					windParticles = append(windParticles, createWindParticle(wind.Position, wind.Direction))
				}
			}

			gravityForce := math32.NewVector3(0, gravity*mass, 0)
			totalForce.Add(gravityForce)

			velocity.MultiplyScalar(1 - dampingEffect)
			dt := float32(deltaTime.Seconds())
			acceleration := totalForce.DivideScalar(mass)
			velocity.Add(acceleration.MultiplyScalar(dt))

			if velocity.Length() > 10 {
				velocity.Normalize().MultiplyScalar(10)
			}

			displacement := velocity.Clone().MultiplyScalar(dt)
			mesh.SetPositionVec(torusPos.Add(displacement))

			if mesh.Position().Y < 1 {
				pos := mesh.Position()
				pos.SetY(1)
				mesh.SetPositionVec(&pos)
				velocity.SetY(0)
			}

			fmt.Printf("Position: %v, Velocity: %v, Total Force: %v\n", mesh.Position(), velocity, totalForce)

			simulationData = append(simulationData, SimulationData{
				Time:            float32(time.Now().UnixNano()) / 1e9,
				Acceleration:    *acceleration,
				WindPower:       windPower,
				AngularMomentum: *angularMomentum,
				DampingEffect:   dampingEffect,
			})
		}

		updateWindParticles(float32(deltaTime.Seconds()))
	})

	// Serialize data
	filename := fmt.Sprintf("simulation_data_%d.json", time.Now().UnixNano())
	file, err := os.Create(filename)
	if err != nil {
		log.Fatal("Error creating simulation data file: ", err)
	}
	defer file.Close()
	json.NewEncoder(file).Encode(simulationData)
}
