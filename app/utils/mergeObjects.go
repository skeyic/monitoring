package utils

type ToMergeObject interface {
	ID() int64
}

// Descend arrays
func MergeDescendObjects(sourceObjects, newObjects []ToMergeObject) (lastObjects []ToMergeObject) {
	var (
		newAnchor, sourceAnchor = 0, 0
		newLength, sourceLength = len(newObjects), len(sourceObjects)
	)

	if newLength == 0 {
		return sourceObjects
	}

	if sourceLength == 0 {
		return newObjects
	}

	for {
		if newAnchor < newLength && sourceAnchor < sourceLength {
			if sourceObjects[sourceAnchor].ID() < newObjects[newAnchor].ID() {
				lastObjects = append(lastObjects, newObjects[newAnchor])
				newAnchor++
			} else if sourceObjects[sourceAnchor].ID() == newObjects[newAnchor].ID() {
				lastObjects = append(lastObjects, newObjects[newAnchor])
				newAnchor++
				sourceAnchor++
			} else {
				lastObjects = append(lastObjects, sourceObjects[sourceAnchor])
				sourceAnchor++
			}
		}

		if newAnchor == newLength {
			lastObjects = append(lastObjects, sourceObjects[sourceAnchor:]...)
			break
		}

		if sourceAnchor == sourceLength {
			lastObjects = append(lastObjects, newObjects[newAnchor:]...)
			break
		}
	}

	return
}
