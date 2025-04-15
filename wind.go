package main

import (
	"github.com/g3n/engine/core"
	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/graphic"
	"github.com/g3n/engine/material"
	"github.com/g3n/engine/math32"
)

type WindSource struct {
	Position  math32.Vector3
	Direction math32.Vector3
	Speed     float32
	Node      *graphic.Mesh
}

type WindParticle struct {
	Mesh     *graphic.Mesh
	Velocity math32.Vector3
	Position *math32.Vector3
	Lifespan float32
	Elapsed  float32
}

func initializeWindSources(scene *core.Node) []WindSource {
	windSources := []WindSource{
		{Position: *math32.NewVector3(0, 1, 0), Direction: *math32.NewVector3(1, 0, 0).Normalize(), Speed: 5.0},
	}

	for i := range windSources {
		sphereGeom := geometry.NewSphere(0.2, 16, 16)
		sphereMat := material.NewStandard(math32.NewColor("Red"))
		sphereMesh := graphic.NewMesh(sphereGeom, sphereMat)
		sphereMesh.SetPositionVec(&windSources[i].Position)
		windSources[i].Node = sphereMesh
		scene.Add(sphereMesh)
	}

	return windSources
}

func createWindParticle(position, direction math32.Vector3) *WindParticle {
	particleGeom := geometry.NewSphere(0.1, 8, 8)
	particleMat := material.NewStandard(math32.NewColor("Cyan"))
	particleMesh := graphic.NewMesh(particleGeom, particleMat)
	particleMesh.SetPositionVec(&position)
	scene.Add(particleMesh)

	return &WindParticle{
		Mesh:     particleMesh,
		Velocity: *direction.Clone().MultiplyScalar(5.0),
		Position: position.Clone(),
		Mass:     1.0,
		Lifespan: 5.0,
		Elapsed:  0,
		Alive:    true,
	}
}

func updateWindParticles(deltaTime float32, scene *core.Node, model *core.Node) {
	var activeParticles []*WindParticle
	for _, particle := range windParticles {
		particle.Elapsed += deltaTime
		if particle.Elapsed >= particle.Lifespan {
			scene.Remove(particle.Mesh)
			continue
		}
		activeParticles = append(activeParticles, particle)
	}
	windParticles = activeParticles
}
