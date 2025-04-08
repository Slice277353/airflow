package main

import (
	"github.com/g3n/engine/core"
	"github.com/g3n/engine/math32"
)

var velocity = math32.NewVector3(0, 0, 0)
var dragCoefficient float32 = 0.47

const airDensity = 1.225
const area = 1.0

var mass float32 = 1.0

const gravity = -9.8

func updatePhysics(particle *WindParticle, object *core.Node, objectVelocity *math32.Vector3, objectMass float32) {
	if particle == nil || object == nil || !particle.Alive {
		return
	}

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
}
