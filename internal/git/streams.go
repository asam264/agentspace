package git

import (
	"io"
	"os"
)

// stdout/stderr are the streams used by RunPassthrough. They default to the
// process streams but are exported as package vars so they could be redirected
// in tests if needed.
var (
	stdout io.Writer = os.Stdout
	stderr io.Writer = os.Stderr
)
