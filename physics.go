package main

import (
	"log"

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

<<<<<<< HEAD

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
=======
	objectPos := object.Position()                  // this returns a Vector3 (by value)
	distanceVec := objectPos.Sub(particle.Position) // Vector3.Sub() returns *Vector3
	distance := distanceVec.Length()
	influenceRadius := float32(3.0)
	if distance > influenceRadius {
		return
	}

	// Strength of force decreases with distance (e.g., inverse-square)
	strength := 1.0 / (1.0 + distance*distance)

	// Force = wind's momentum scaled by proximity
	force := particle.Velocity.Clone().MultiplyScalar(particle.Mass * strength)

	// Apply to object's velocity
	acceleration := force.DivideScalar(objectMass)
	objectVelocity.Add(acceleration)

	// Optional: particle dies after pushing
	// particle.Alive = false
>>>>>>> parent of c0edab8 (prototype)
}
