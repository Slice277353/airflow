package camera

import (
	"github.com/g3n/engine/camera"
	"github.com/g3n/engine/math32"
)

func NewRayFromMouse(cam camera.ICamera, x, y float32) *math32.Ray {
	// Create ray with initial vectors
	origin := math32.NewVector3(0, 2, 5) // Default camera position
	direction := math32.NewVector3(x, y, -1)

	// Create ray
	ray := math32.NewRay(origin, direction)

	// Calculate world-space direction
	direction.Normalize()

	return ray
}
