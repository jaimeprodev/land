package cert_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"

	"github.com/lightningnetwork/lnd/cert"
	"github.com/stretchr/testify/require"
)

const (
	testTLSCertDuration = 42 * time.Hour
)

var (
	extraIPs     = []string{"1.1.1.1", "123.123.123.1", "199.189.12.12"}
	extraDomains = []string{"home", "and", "away"}
)

// TestIsOutdatedCert checks that we'll consider the TLS certificate outdated
// if the ip addresses or dns names don't match.
func TestIsOutdatedCert(t *testing.T) {
	tempDir := t.TempDir()

	certPath := tempDir + "/tls.cert"
	keyPath := tempDir + "/tls.key"

	// Generate TLS files with two extra IPs and domains.
	certBytes, keyBytes, err := cert.GenCertPair(
		"lnd autogenerated cert", extraIPs[:2], extraDomains[:2],
		false, testTLSCertDuration,
	)
	if err != nil {
		t.Fatal(err)
	}
	err = cert.WriteCertPair(certPath, keyPath, certBytes, keyBytes)
	require.NoError(t, err)

	// We'll attempt to check up-to-date status for all variants of 1-3
	// number of IPs and domains.
	for numIPs := 1; numIPs <= len(extraIPs); numIPs++ {
		for numDomains := 1; numDomains <= len(extraDomains); numDomains++ {
			certBytes, err := ioutil.ReadFile(certPath)
			require.NoError(t, err)

			keyBytes, err := ioutil.ReadFile(keyPath)
			require.NoError(t, err)

			_, parsedCert, err := cert.LoadCertFromBytes(
				certBytes, keyBytes,
			)
			if err != nil {
				t.Fatal(err)
			}

			// Using the test case's number of IPs and domains, get
			// the outdated status of the certificate we created
			// above.
			outdated, err := cert.IsOutdated(
				parsedCert, extraIPs[:numIPs],
				extraDomains[:numDomains], false,
			)
			if err != nil {
				t.Fatal(err)
			}

			// We expect it to be considered outdated if the IPs or
			// domains don't match exactly what we created.
			expected := numIPs != 2 || numDomains != 2
			if outdated != expected {
				t.Fatalf("expected certificate to be "+
					"outdated=%v, got=%v", expected,
					outdated)
			}
		}
	}
}

// TestIsOutdatedPermutation tests that the order of listed IPs or DNS names,
// nor dulicates in the lists, matter for whether we consider the certificate
// outdated.
func TestIsOutdatedPermutation(t *testing.T) {
	tempDir := t.TempDir()

	certPath := tempDir + "/tls.cert"
	keyPath := tempDir + "/tls.key"

	// Generate TLS files from the IPs and domains.
	certBytes, keyBytes, err := cert.GenCertPair(
		"lnd autogenerated cert", extraIPs[:], extraDomains[:],
		false, testTLSCertDuration,
	)
	if err != nil {
		t.Fatal(err)
	}
	err = cert.WriteCertPair(certPath, keyPath, certBytes, keyBytes)
	require.NoError(t, err)

	certBytes, err = ioutil.ReadFile(certPath)
	require.NoError(t, err)

	keyBytes, err = ioutil.ReadFile(keyPath)
	require.NoError(t, err)

	_, parsedCert, err := cert.LoadCertFromBytes(certBytes, keyBytes)
	require.NoError(t, err)

	// If we have duplicate IPs or DNS names listed, that shouldn't matter.
	dupIPs := make([]string, len(extraIPs)*2)
	for i := range dupIPs {
		dupIPs[i] = extraIPs[i/2]
	}

	dupDNS := make([]string, len(extraDomains)*2)
	for i := range dupDNS {
		dupDNS[i] = extraDomains[i/2]
	}

	outdated, err := cert.IsOutdated(parsedCert, dupIPs, dupDNS, false)
	if err != nil {
		t.Fatal(err)
	}

	if outdated {
		t.Fatalf("did not expect duplicate IPs or DNS names be " +
			"considered outdated")
	}

	// Similarly, the order of the lists shouldn't matter.
	revIPs := make([]string, len(extraIPs))
	for i := range revIPs {
		revIPs[i] = extraIPs[len(extraIPs)-1-i]
	}

	revDNS := make([]string, len(extraDomains))
	for i := range revDNS {
		revDNS[i] = extraDomains[len(extraDomains)-1-i]
	}

	outdated, err = cert.IsOutdated(parsedCert, revIPs, revDNS, false)
	if err != nil {
		t.Fatal(err)
	}

	if outdated {
		t.Fatalf("did not expect reversed IPs or DNS names be " +
			"considered outdated")
	}
}

// TestTLSDisableAutofill checks that setting the --tlsdisableautofill flag
// does not add interface ip addresses or hostnames to the cert.
func TestTLSDisableAutofill(t *testing.T) {
	tempDir := t.TempDir()

	certPath := tempDir + "/tls.cert"
	keyPath := tempDir + "/tls.key"

	// Generate TLS files with two extra IPs and domains and no interface IPs.
	certBytes, keyBytes, err := cert.GenCertPair(
		"lnd autogenerated cert", extraIPs[:2], extraDomains[:2],
		true, testTLSCertDuration,
	)
	require.NoError(
		t, err,
		"unable to generate tls certificate pair",
	)
	err = cert.WriteCertPair(certPath, keyPath, certBytes, keyBytes)
	require.NoError(t, err)

	// Read certs from disk.
	certBytes, err = ioutil.ReadFile(certPath)
	require.NoError(t, err)

	keyBytes, err = ioutil.ReadFile(keyPath)
	require.NoError(t, err)

	// Load the certificate.
	_, parsedCert, err := cert.LoadCertFromBytes(
		certBytes, keyBytes,
	)
	require.NoError(
		t, err,
		"unable to load tls certificate pair",
	)

	// Check if the TLS cert is outdated while still preventing
	// interface IPs from being used. Should not be outdated.
	shouldNotBeOutdated, err := cert.IsOutdated(
		parsedCert, extraIPs[:2],
		extraDomains[:2], true,
	)
	require.NoError(t, err)

	require.Equal(
		t, false, shouldNotBeOutdated,
		"TLS Certificate was marked as outdated when it should not be",
	)

	// Check if the TLS cert is outdated while allowing for
	// interface IPs to be used. Should report as outdated.
	shouldBeOutdated, err := cert.IsOutdated(
		parsedCert, extraIPs[:2],
		extraDomains[:2], false,
	)
	require.NoError(t, err)

	require.Equal(
		t, true, shouldBeOutdated,
		"TLS Certificate was not marked as outdated when it should be",
	)
}

// TestTLSConfig tests to ensure we can generate a TLS Config from
// a tls cert and tls key.
func TestTLSConfig(t *testing.T) {
	tempDir := t.TempDir()
	certPath := filepath.Join(tempDir, "/tls.cert")
	keyPath := filepath.Join(tempDir, "/tls.key")

	// Generate TLS files with an extra IP and domain.
	certBytes, keyBytes, err := cert.GenCertPair(
		"lnd autogenerated cert", []string{extraIPs[0]},
		[]string{extraDomains[0]}, false, testTLSCertDuration,
	)
	require.NoError(t, err)

	err = cert.WriteCertPair(certPath, keyPath, certBytes, keyBytes)
	require.NoError(t, err)

	certBytes, err = ioutil.ReadFile(certPath)
	require.NoError(t, err)

	keyBytes, err = ioutil.ReadFile(keyPath)
	require.NoError(t, err)

	// Load the certificate.
	certData, parsedCert, err := cert.LoadCertFromBytes(
		certBytes, keyBytes,
	)
	require.NoError(t, err)

	// Check to make sure the IP and domain are in the cert.
	var foundIp bool
	require.Contains(t, parsedCert.DNSNames, extraDomains[0])
	for _, ip := range parsedCert.IPAddresses {
		if ip.String() == extraIPs[0] {
			foundIp = true
			break
		}
	}
	require.Equal(t, true, foundIp, "Did not find required ip inside of "+
		"TLS Certificate.")

	// Create TLS Config.
	tlsCfg := cert.TLSConfFromCert(certData)

	require.Equal(t, 1, len(tlsCfg.Certificates))
}
