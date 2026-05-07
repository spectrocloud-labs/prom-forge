package config

// OpaqueString is a string that is not meant to be logged.
type OpaqueString string

// String returns the string representation of the OpaqueString.
func (s OpaqueString) String() string {
	return "[REDACTED]"
}
