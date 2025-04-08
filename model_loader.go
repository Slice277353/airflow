package main

import (
	"fmt"
	"path/filepath"

	"github.com/g3n/engine/core"
	"github.com/g3n/engine/loader/obj"
)

type ModelLoader struct {
	scene  *core.Node
	models []*core.Node
}

func (ml *ModelLoader) LoadModel(fpath string) error {
	ext := filepath.Ext(fpath)
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
	default:
		return fmt.Errorf("unsupported model format: %s", ext)
	}
	return nil
}

func (ml *ModelLoader) GetLoadedModel() *core.Node {
	if len(ml.models) > 0 {
		return ml.models[0]
	}
	return nil
}
