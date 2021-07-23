/* Copyright (c) Edgeless Systems GmbH

   This program is free software; you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation; version 2 of the License.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with this program; if not, write to the Free Software
   Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA  02110-1335  USA */

package db

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"net"
	"time"

	"github.com/edgelesssys/edb/edb/util"
)

func splitHostPort(address, defaultPort string) (host, port string) {
	if address == "" {
		return "0.0.0.0", defaultPort
	}
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return address, defaultPort
	}
	if host == "" {
		host = "0.0.0.0"
	}
	if port == "" {
		port = defaultPort
	}
	return
}

func toPEM(cert []byte, key crypto.PrivateKey) (pemCert, pemKey []byte, err error) {
	pemCert = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert})
	if len(pemCert) <= 0 {
		return nil, nil, errors.New("pem.EncodeToMemory failed")
	}

	keyBytes, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, nil, err
	}

	pemKey = pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyBytes})
	if len(pemKey) <= 0 {
		return nil, nil, errors.New("pem.EncodeToMemory failed")
	}

	return
}

func createCertificate(commonName string) ([]byte, crypto.PrivateKey, error) {
	serialNumber, err := util.GenerateCertificateSerialNumber()
	if err != nil {
		return nil, nil, err
	}
	template := &x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               pkix.Name{Organization: []string{"EDB root"}, CommonName: commonName},
		NotAfter:              time.Now().AddDate(10, 0, 0),
		DNSNames:              []string{commonName},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	cert, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	if err != nil {
		return nil, nil, err
	}
	return cert, priv, nil
}
