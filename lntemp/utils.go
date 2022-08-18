package lntemp

import (
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/lightningnetwork/lnd/lntest"
)

const (
	// NeutrinoBackendName is the name of the neutrino backend.
	NeutrinoBackendName = "neutrino"

	// TODO(yy): delete.
	DefaultTimeout = lntest.DefaultTimeout

	// noFeeLimitMsat is used to specify we will put no requirements on fee
	// charged when choosing a route path.
	noFeeLimitMsat = math.MaxInt64

	// defaultPaymentTimeout specifies the default timeout in seconds when
	// sending a payment.
	defaultPaymentTimeout = 60
)

// CopyFile copies the file src to dest.
func CopyFile(dest, src string) error {
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer s.Close()

	d, err := os.Create(dest)
	if err != nil {
		return err
	}

	if _, err := io.Copy(d, s); err != nil {
		d.Close()
		return err
	}

	return d.Close()
}

// errNumNotMatched is a helper method to return a nicely formatted error.
func errNumNotMatched(name string, subject string,
	want, got, total, old int) error {

	return fmt.Errorf("%s: assert %s failed: want %d, got: %d, total: "+
		"%d, previously had: %d", name, subject, want, got, total, old)
}

// parseDerivationPath parses a path in the form of m/x'/y'/z'/a/b into a slice
// of [x, y, z, a, b], meaning that the apostrophe is ignored and 2^31 is _not_
// added to the numbers.
func ParseDerivationPath(path string) ([]uint32, error) {
	path = strings.TrimSpace(path)
	if len(path) == 0 {
		return nil, fmt.Errorf("path cannot be empty")
	}
	if !strings.HasPrefix(path, "m/") {
		return nil, fmt.Errorf("path must start with m/")
	}

	// Just the root key, no path was provided. This is valid but not useful
	// in most cases.
	rest := strings.ReplaceAll(path, "m/", "")
	if rest == "" {
		return []uint32{}, nil
	}

	parts := strings.Split(rest, "/")
	indices := make([]uint32, len(parts))
	for i := 0; i < len(parts); i++ {
		part := parts[i]
		if strings.Contains(parts[i], "'") {
			part = strings.TrimRight(parts[i], "'")
		}
		parsed, err := strconv.ParseInt(part, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("could not parse part \"%s\": "+
				"%v", part, err)
		}
		indices[i] = uint32(parsed)
	}

	return indices, nil
}
