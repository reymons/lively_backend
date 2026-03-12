package codec

const (
	ERequiredField = "required field"
)

type Errors = map[string]string

type Validator interface {
	// Validates an object and returns a problems' map
	// If len(map) > 0, the object is invalid
	Valid() Errors
}
