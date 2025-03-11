package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/g3n/engine/core"
	"github.com/g3n/engine/loader/collada"
	"github.com/g3n/engine/loader/gltf"
	"github.com/g3n/engine/loader/obj"
)

// ModelLoader handles loading of 3D models
type ModelLoader struct {
	scene  *core.Node
	models []*core.Node
}

func openFileDialog() (string, error) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("powershell", "-Command", "Add-Type -AssemblyName System.Windows.Forms; "+
			"$dlg = New-Object System.Windows.Forms.OpenFileDialog; "+
			"$dlg.Filter = '3D Models (*.obj;*.gltf;*.dae)|*.obj;*.gltf;*.dae'; "+
			"$dlg.ShowDialog() | Out-Null; "+
			"Write-Output $dlg.FileName")
	case "darwin":
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
			// Simplified GLTF support: Add a placeholder node for the scene
			log.Println("Loading GLTF model (simplified implementation)")
			placeholder := core.NewNode()
			ml.scene.Add(placeholder)
			ml.models = append(ml.models, placeholder)
			// TODO: Full GLTF support requires processing g.Nodes, g.Meshes, etc.
		} else {
			log.Println("GLTF Scene undefined, check the file.")
			return fmt.Errorf("no scene defined in GLTF file")
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
