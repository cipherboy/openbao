package listenerutil

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/caddyserver/certmagic"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/go-secure-stdlib/reloadutil"
	"github.com/mitchellh/cli"
	"go.uber.org/zap"

	"github.com/openbao/openbao/internalshared/configutil"
)

type Reloadable interface {
	Reload() error
}

type CertGetter interface {
	GetCertificate(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error)
}

type ReloadableCertGetter interface {
	Reloadable
	CertGetter
}

type ACMECertGetter struct {
	Listener *configutil.Listener

	Magic *certmagic.Config
	ACME  *certmagic.ACMEIssuer
}

func NewCertificateGetter(l *configutil.Listener, ui cli.Ui) (ReloadableCertGetter, error) {
	// Assume a validated listener here.
	if l.TLSCertFile != "" {
		cg := reloadutil.NewCertificateGetter(l.TLSCertFile, l.TLSKeyFile, "")
		if err := cg.Reload(); err != nil {
			// We try the key without a passphrase first and if we get an incorrect
			// passphrase response, try again after prompting for a passphrase
			if errwrap.Contains(err, x509.IncorrectPasswordError.Error()) {
				var passphrase string
				passphrase, err = ui.AskSecret(fmt.Sprintf("Enter passphrase for %s:", l.TLSKeyFile))
				if err == nil {
					cg = reloadutil.NewCertificateGetter(l.TLSCertFile, l.TLSKeyFile, passphrase)
					if err = cg.Reload(); err == nil {
						return cg, nil
					}

					return nil, fmt.Errorf("error loading TLS cert with password: %w", err)
				}
			}

			return nil, fmt.Errorf("error loading TLS cert: %w", err)
		}

		return cg, nil
	}

	acg := &ACMECertGetter{
		Listener: l,
		Magic:    certmagic.NewDefault(),
	}

	acg.Magic.OnDemand = new(certmagic.OnDemandConfig)
	acg.Magic.Logger = zap.NewNop()

	if l.TLSACMECachePath != "" {
		acg.Magic.Storage = &certmagic.FileStorage{
			Path: l.TLSACMECachePath,
		}
	}

	if l.TLSACMEKeyType != "" {
		switch certmagic.KeyType(l.TLSACMEKeyType) {
		case certmagic.ED25519, certmagic.P256, certmagic.P384, certmagic.RSA2048, certmagic.RSA4096, certmagic.RSA8192:
		default:
			return nil, fmt.Errorf("unknown value for tls_acme_key_type (`%v`); allowed values are `%v`, `%v`, `%v`, `%v`, `%v`, `%v`", l.TLSACMEKeyType, certmagic.ED25519, certmagic.P256, certmagic.P384, certmagic.RSA2048, certmagic.RSA4096, certmagic.RSA8192)
		}

		acg.Magic.KeySource = certmagic.StandardKeyGenerator{
			KeyType: certmagic.KeyType(l.TLSACMEKeyType),
		}
	}

	template := certmagic.ACMEIssuer{
		CA:     l.TLSACMECADirectory,
		TestCA: l.TLSACMECADirectory,
		Email:  l.TLSACMEEmail,
		Agreed: true,
		Logger: zap.NewNop(),
	}

	if l.TLSACMECARoot != "" {
		caPool := x509.NewCertPool()

		data, err := ioutil.ReadFile(l.TLSACMECARoot)
		if err != nil {
			return nil, fmt.Errorf("failed to read ACME CA file: %w", err)
		}

		if !caPool.AppendCertsFromPEM(data) {
			return nil, fmt.Errorf("failed to parse ACME CA certificate")
		}

		template.TrustedRoots = caPool
	}

	if l.TLSACMEDNSConfig != nil {
		template.DisableHTTPChallenge = true
		template.DisableTLSALPNChallenge = true

		template.DNS01Solver = &certmagic.DNS01Solver{
			DNSProvider:        l.TLSACMEDNSConfig.GetProvider(),
			TTL:                l.TLSACMEDNSConfig.TTL,
			PropagationDelay:   l.TLSACMEDNSConfig.PropagationDelay,
			PropagationTimeout: l.TLSACMEDNSConfig.PropagationTimeout,
			Resolvers:          l.TLSACMEDNSConfig.Resolvers,
			OverrideDomain:     l.TLSACMEDNSConfig.OverrideDomain,
		}
	}

	acg.ACME = certmagic.NewACMEIssuer(acg.Magic, template)

	acg.Magic.Issuers = []certmagic.Issuer{acg.ACME}

	return acg, nil
}

func (c *ACMECertGetter) ALPNProtos() []string {
	if c.Listener.TLSACMEDNSConfig != nil {
		return nil
	}

	return []string{"acme-tls/1"}
}

func (c *ACMECertGetter) HandleHTTPChallenge(w http.ResponseWriter, r *http.Request) {
	c.ACME.HandleHTTPChallenge(w, r)
}

func (c *ACMECertGetter) Reload() error { return nil }

func (c *ACMECertGetter) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	// ACMECertGetter follows the strategy by Caddy of on-demand TLS
	// configuration: when a connection comes in for a domain, attempt to
	// get a valid certificate from the ACME responder, just-in-time.
	return c.Magic.GetCertificate(hello)
}
