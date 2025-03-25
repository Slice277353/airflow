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
	Radius    float32
	Speed     float32
	Direction math32.Vector3
	Node      *graphic.Mesh // 25.03
}

type WindParticle struct {
	Mesh     *graphic.Mesh
	Velocity math32.Vector3
	Lifespan float32
	Elapsed  float32
}

var windParticles []*WindParticle

func initializeWindSources(scene *core.Node) []WindSource {
	windSources := []WindSource{
		{Position: *math32.NewVector3(2, 1, 0), Radius: 3.0, Speed: 10.0, Direction: *math32.NewVector3(-1, 0, 0).Normalize()},
		{Position: *math32.NewVector3(-2, 1, 0), Radius: 2.0, Speed: 5.0, Direction: *math32.NewVector3(1, 0, 0).Normalize()},
	}

	for i := range windSources {
		sphereGeom := geometry.NewSphere(0.2, 16, 16)
		sphereMat := material.NewStandard(math32.NewColor("Red"))
		sphereMesh := graphic.NewMesh(sphereGeom, sphereMat)
		sphereMesh.SetPositionVec(&windSources[i].Position)
		windSources[i].Node = sphereMesh // Store the mesh in the WindSource struct
		scene.Add(sphereMesh)
	} // a few changes in here as well

	return windSources
}

func addWindSource(windSource []WindSource, scene *core.Node, position math32.Vector3) []WindSource {
	newWind := WindSource{
		Position:  position,
		Radius:    2.0,
		Speed:     5.0,
		Direction: *math32.NewVector3(1, 0, 0).Normalize(),
	}

	sphereGeom := geometry.NewSphere(0.2, 16, 16)
	sphereMat := material.NewStandard(math32.NewColor("Red"))
	sphereMesh := graphic.NewMesh(sphereGeom, sphereMat)
	sphereMesh.SetPositionVec(&newWind.Position)
	newWind.Node = sphereMesh
	scene.Add(sphereMesh)

	return append(windSource, newWind)
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
