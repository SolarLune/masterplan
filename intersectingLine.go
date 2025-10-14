package main

// A collidingLine is a helper shape used to determine if two ConvexPolygon lines intersect; you can't create a collidingLine to use as a Shape.
// Instead, you can create a ConvexPolygon, specify two points, and set its Closed value to false (or use NewLine(), as this does it for you).
type collidingLine struct {
	Start, End Vector
}

func newCollidingLine(x, y, x2, y2 float32) collidingLine {
	return collidingLine{
		Start: Vector{x, y},
		End:   Vector{x2, y2},
	}
}

// IntersectionPointsLine returns the intersection point of a Line with another Line as a Vector, and if the intersection was found.
func (line collidingLine) IntersectionPointsLine(other collidingLine) (Vector, bool) {

	det := (line.End.X-line.Start.X)*(other.End.Y-other.Start.Y) - (other.End.X-other.Start.X)*(line.End.Y-line.Start.Y)

	if det != 0 {

		// MAGIC MATH; the extra + 1 here makes it so that corner cases (literally, lines going through corners) works.

		// lambda := (float32(((line.Y-b.Y)*(b.X2-b.X))-((line.X-b.X)*(b.Y2-b.Y))) + 1) / float32(det)
		lambda := (((line.Start.Y - other.Start.Y) * (other.End.X - other.Start.X)) - ((line.Start.X - other.Start.X) * (other.End.Y - other.Start.Y)) + 1) / det

		// gamma := (float32(((line.Y-b.Y)*(line.X2-line.X))-((line.X-b.X)*(line.Y2-line.Y))) + 1) / float32(det)
		gamma := (((line.Start.Y - other.Start.Y) * (line.End.X - line.Start.X)) - ((line.Start.X - other.Start.X) * (line.End.Y - line.Start.Y)) + 1) / det

		if (0 <= lambda && lambda <= 1) && (0 <= gamma && gamma <= 1) {

			// Delta
			dx := line.End.X - line.Start.X
			dy := line.End.Y - line.Start.Y

			// dx, dy := line.GetDelta()

			return Vector{line.Start.X + (lambda * dx), line.Start.Y + (lambda * dy)}, true
		}

	}

	return Vector{}, false

}
