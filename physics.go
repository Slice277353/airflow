package main

import (
	"github.com/g3n/engine/core"
	"github.com/g3n/engine/math32"
)

const airDensity = 1.225
const dragCoefficient = 0.47
const area = 1.0
const gravity = 9.81 // Earth's gravity in m/s^2

func updatePhysics(particle *WindParticle, object *core.Node, deltaTime float32) {
	if particle == nil || object == nil || !particle.Alive {
		return
	}

	// Apply gravity
	gravityForce := math32.NewVector3(0, gravity*particle.Mass, 0)
	particle.Velocity.Add(gravityForce.MultiplyScalar(deltaTime))

	// Apply drag
	velocityMagnitude := particle.Velocity.Length()
	dragForce := particle.Velocity.Clone().Normalize().MultiplyScalar(
		-0.5 * airDensity * dragCoefficient * area * velocityMagnitude * velocityMagnitude,
	)
	particle.Velocity.Add(dragForce.MultiplyScalar(deltaTime))

	// Update position
	particle.Position.Add(particle.Velocity.Clone().MultiplyScalar(deltaTime))
	particle.Mesh.SetPositionVec(particle.Position)
}
