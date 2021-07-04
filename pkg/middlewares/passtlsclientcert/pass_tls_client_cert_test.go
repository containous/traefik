package passtlsclientcert

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/testhelpers"
)

const (
	signingCA = `Certificate:
    Data:
        Version: 3 (0x2)
        Serial Number: 2 (0x2)
        Signature Algorithm: sha1WithRSAEncryption
        Issuer: DC=org, DC=cheese, O=Cheese, O=Cheese 2, OU=Cheese Section, OU=Cheese Section 2, CN=Simple Root CA, CN=Simple Root CA 2, C=FR, C=US, L=TOULOUSE, L=LYON, ST=Root State, ST=Root State 2/emailAddress=root@signing.com/emailAddress=root2@signing.com
        Validity
            Not Before: Dec  6 11:10:09 2018 GMT
            Not After : Dec  5 11:10:09 2028 GMT
        Subject: DC=org, DC=cheese, O=Cheese, O=Cheese 2, OU=Simple Signing Section, OU=Simple Signing Section 2, CN=Simple Signing CA, CN=Simple Signing CA 2, C=FR, C=US, L=TOULOUSE, L=LYON, ST=Signing State, ST=Signing State 2/emailAddress=simple@signing.com/emailAddress=simple2@signing.com
        Subject Public Key Info:
            Public Key Algorithm: rsaEncryption
                RSA Public-Key: (2048 bit)
                Modulus:
                    00:c3:9d:9f:61:15:57:3f:78:cc:e7:5d:20:e2:3e:
                    2e:79:4a:c3:3a:0c:26:40:18:db:87:08:85:c2:f7:
                    af:87:13:1a:ff:67:8a:b7:2b:58:a7:cc:89:dd:77:
                    ff:5e:27:65:11:80:82:8f:af:a0:af:25:86:ec:a2:
                    4f:20:0e:14:15:16:12:d7:74:5a:c3:99:bd:3b:81:
                    c8:63:6f:fc:90:14:86:d2:39:ee:87:b2:ff:6d:a5:
                    69:da:ab:5a:3a:97:cd:23:37:6a:4b:ba:63:cd:a1:
                    a9:e6:79:aa:37:b8:d1:90:c9:24:b5:e8:70:fc:15:
                    ad:39:97:28:73:47:66:f6:22:79:5a:b0:03:83:8a:
                    f1:ca:ae:8b:50:1e:c8:fa:0d:9f:76:2e:00:c2:0e:
                    75:bc:47:5a:b6:d8:05:ed:5a:bc:6d:50:50:36:6b:
                    ab:ab:69:f6:9b:1b:6c:7e:a8:9f:b2:33:3a:3c:8c:
                    6d:5e:83:ce:17:82:9e:10:51:a6:39:ec:98:4e:50:
                    b7:b1:aa:8b:ac:bb:a1:60:1b:ea:31:3b:b8:0a:ea:
                    63:41:79:b5:ec:ee:19:e9:85:8e:f3:6d:93:80:da:
                    98:58:a2:40:93:a5:53:eb:1d:24:b6:66:07:ec:58:
                    10:63:e7:fa:6e:18:60:74:76:15:39:3c:f4:95:95:
                    7e:df
                Exponent: 65537 (0x10001)
        X509v3 extensions:
            X509v3 Key Usage: critical
                Certificate Sign, CRL Sign
            X509v3 Basic Constraints: critical
                CA:TRUE, pathlen:0
            X509v3 Subject Key Identifier: 
                1E:52:A2:E8:54:D5:37:EB:D5:A8:1D:E4:C2:04:1D:37:E2:F7:70:03
            X509v3 Authority Key Identifier: 
                keyid:36:70:35:AA:F0:F6:93:B2:86:5D:32:73:F9:41:5A:3F:3B:C8:BC:8B

    Signature Algorithm: sha1WithRSAEncryption
         76:f3:16:21:27:6d:a2:2e:e8:18:49:aa:54:1e:f8:3b:07:fa:
         65:50:d8:1f:a2:cf:64:6c:15:e0:0f:c8:46:b2:d7:b8:0e:cd:
         05:3b:06:fb:dd:c6:2f:01:ae:bd:69:d3:bb:55:47:a9:f6:e5:
         ba:be:4b:45:fb:2e:3c:33:e0:57:d4:3e:8e:3e:11:f2:0a:f1:
         7d:06:ab:04:2e:a5:76:20:c2:db:a4:68:5a:39:00:62:2a:1d:
         c2:12:b1:90:66:8c:36:a8:fd:83:d1:1b:da:23:a7:1d:5b:e6:
         9b:40:c4:78:25:c7:b7:6b:75:35:cf:bb:37:4a:4f:fc:7e:32:
         1f:8c:cf:12:d2:c9:c8:99:d9:4a:55:0a:1e:ac:de:b4:cb:7c:
         bf:c4:fb:60:2c:a8:f7:e7:63:5c:b0:1c:62:af:01:3c:fe:4d:
         3c:0b:18:37:4c:25:fc:d0:b2:f6:b2:f1:c3:f4:0f:53:d6:1e:
         b5:fa:bc:d8:ad:dd:1c:f5:45:9f:af:fe:0a:01:79:92:9a:d8:
         71:db:37:f3:1e:bd:fb:c7:1e:0a:0f:97:2a:61:f3:7b:19:93:
         9c:a6:8a:69:cd:b0:f5:91:02:a5:1b:10:f4:80:5d:42:af:4e:
         82:12:30:3e:d3:a7:11:14:ce:50:91:04:80:d7:2a:03:ef:71:
         10:b8:db:a5
-----BEGIN CERTIFICATE-----
MIIFzTCCBLWgAwIBAgIBAjANBgkqhkiG9w0BAQUFADCCAWQxEzARBgoJkiaJk/Is
ZAEZFgNvcmcxFjAUBgoJkiaJk/IsZAEZFgZjaGVlc2UxDzANBgNVBAoMBkNoZWVz
ZTERMA8GA1UECgwIQ2hlZXNlIDIxFzAVBgNVBAsMDkNoZWVzZSBTZWN0aW9uMRkw
FwYDVQQLDBBDaGVlc2UgU2VjdGlvbiAyMRcwFQYDVQQDDA5TaW1wbGUgUm9vdCBD
QTEZMBcGA1UEAwwQU2ltcGxlIFJvb3QgQ0EgMjELMAkGA1UEBhMCRlIxCzAJBgNV
BAYTAlVTMREwDwYDVQQHDAhUT1VMT1VTRTENMAsGA1UEBwwETFlPTjETMBEGA1UE
CAwKUm9vdCBTdGF0ZTEVMBMGA1UECAwMUm9vdCBTdGF0ZSAyMR8wHQYJKoZIhvcN
AQkBFhByb290QHNpZ25pbmcuY29tMSAwHgYJKoZIhvcNAQkBFhFyb290MkBzaWdu
aW5nLmNvbTAeFw0xODEyMDYxMTEwMDlaFw0yODEyMDUxMTEwMDlaMIIBhDETMBEG
CgmSJomT8ixkARkWA29yZzEWMBQGCgmSJomT8ixkARkWBmNoZWVzZTEPMA0GA1UE
CgwGQ2hlZXNlMREwDwYDVQQKDAhDaGVlc2UgMjEfMB0GA1UECwwWU2ltcGxlIFNp
Z25pbmcgU2VjdGlvbjEhMB8GA1UECwwYU2ltcGxlIFNpZ25pbmcgU2VjdGlvbiAy
MRowGAYDVQQDDBFTaW1wbGUgU2lnbmluZyBDQTEcMBoGA1UEAwwTU2ltcGxlIFNp
Z25pbmcgQ0EgMjELMAkGA1UEBhMCRlIxCzAJBgNVBAYTAlVTMREwDwYDVQQHDAhU
T1VMT1VTRTENMAsGA1UEBwwETFlPTjEWMBQGA1UECAwNU2lnbmluZyBTdGF0ZTEY
MBYGA1UECAwPU2lnbmluZyBTdGF0ZSAyMSEwHwYJKoZIhvcNAQkBFhJzaW1wbGVA
c2lnbmluZy5jb20xIjAgBgkqhkiG9w0BCQEWE3NpbXBsZTJAc2lnbmluZy5jb20w
ggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQDDnZ9hFVc/eMznXSDiPi55
SsM6DCZAGNuHCIXC96+HExr/Z4q3K1inzIndd/9eJ2URgIKPr6CvJYbsok8gDhQV
FhLXdFrDmb07gchjb/yQFIbSOe6Hsv9tpWnaq1o6l80jN2pLumPNoanmeao3uNGQ
ySS16HD8Fa05lyhzR2b2InlasAODivHKrotQHsj6DZ92LgDCDnW8R1q22AXtWrxt
UFA2a6urafabG2x+qJ+yMzo8jG1eg84Xgp4QUaY57JhOULexqousu6FgG+oxO7gK
6mNBebXs7hnphY7zbZOA2phYokCTpVPrHSS2ZgfsWBBj5/puGGB0dhU5PPSVlX7f
AgMBAAGjZjBkMA4GA1UdDwEB/wQEAwIBBjASBgNVHRMBAf8ECDAGAQH/AgEAMB0G
A1UdDgQWBBQeUqLoVNU369WoHeTCBB034vdwAzAfBgNVHSMEGDAWgBQ2cDWq8PaT
soZdMnP5QVo/O8i8izANBgkqhkiG9w0BAQUFAAOCAQEAdvMWISdtoi7oGEmqVB74
Owf6ZVDYH6LPZGwV4A/IRrLXuA7NBTsG+93GLwGuvWnTu1VHqfblur5LRfsuPDPg
V9Q+jj4R8grxfQarBC6ldiDC26RoWjkAYiodwhKxkGaMNqj9g9Eb2iOnHVvmm0DE
eCXHt2t1Nc+7N0pP/H4yH4zPEtLJyJnZSlUKHqzetMt8v8T7YCyo9+djXLAcYq8B
PP5NPAsYN0wl/NCy9rLxw/QPU9Yetfq82K3dHPVFn6/+CgF5kprYcds38x69+8ce
Cg+XKmHzexmTnKaKac2w9ZECpRsQ9IBdQq9OghIwPtOnERTOUJEEgNcqA+9xELjb
pQ==
-----END CERTIFICATE-----
`

	minimalCheeseCrt = `-----BEGIN CERTIFICATE-----
MIIEQDCCAygCFFRY0OBk/L5Se0IZRj3CMljawL2UMA0GCSqGSIb3DQEBCwUAMIIB
hDETMBEGCgmSJomT8ixkARkWA29yZzEWMBQGCgmSJomT8ixkARkWBmNoZWVzZTEP
MA0GA1UECgwGQ2hlZXNlMREwDwYDVQQKDAhDaGVlc2UgMjEfMB0GA1UECwwWU2lt
cGxlIFNpZ25pbmcgU2VjdGlvbjEhMB8GA1UECwwYU2ltcGxlIFNpZ25pbmcgU2Vj
dGlvbiAyMRowGAYDVQQDDBFTaW1wbGUgU2lnbmluZyBDQTEcMBoGA1UEAwwTU2lt
cGxlIFNpZ25pbmcgQ0EgMjELMAkGA1UEBhMCRlIxCzAJBgNVBAYTAlVTMREwDwYD
VQQHDAhUT1VMT1VTRTENMAsGA1UEBwwETFlPTjEWMBQGA1UECAwNU2lnbmluZyBT
dGF0ZTEYMBYGA1UECAwPU2lnbmluZyBTdGF0ZSAyMSEwHwYJKoZIhvcNAQkBFhJz
aW1wbGVAc2lnbmluZy5jb20xIjAgBgkqhkiG9w0BCQEWE3NpbXBsZTJAc2lnbmlu
Zy5jb20wHhcNMTgxMjA2MTExMDM2WhcNMjEwOTI1MTExMDM2WjAzMQswCQYDVQQG
EwJGUjETMBEGA1UECAwKU29tZS1TdGF0ZTEPMA0GA1UECgwGQ2hlZXNlMIIBIjAN
BgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAskX/bUtwFo1gF2BTPNaNcTUMaRFu
FMZozK8IgLjccZ4kZ0R9oFO6Yp8Zl/IvPaf7tE26PI7XP7eHriUdhnQzX7iioDd0
RZa68waIhAGc+xPzRFrP3b3yj3S2a9Rve3c0K+SCV+EtKAwsxMqQDhoo9PcBfo5B
RHfht07uD5MncUcGirwN+/pxHV5xzAGPcc7On0/5L7bq/G+63nhu78zw9XyuLaHC
PM5VbOUvpyIESJHbMMzTdFGL8ob9VKO+Kr1kVGdEA9i8FLGl3xz/GBKuW/JD0xyW
DrU29mri5vYWHmkuv7ZWHGXnpXjTtPHwveE9/0/ArnmpMyR9JtqFr1oEvQIDAQAB
MA0GCSqGSIb3DQEBCwUAA4IBAQBHta+NWXI08UHeOkGzOTGRiWXsOH2dqdX6gTe9
xF1AIjyoQ0gvpoGVvlnChSzmlUj+vnx/nOYGIt1poE3hZA3ZHZD/awsvGyp3GwWD
IfXrEViSCIyF+8tNNKYyUcEO3xdAsAUGgfUwwF/mZ6MBV5+A/ZEEILlTq8zFt9dV
vdKzIt7fZYxYBBHFSarl1x8pDgWXlf3hAufevGJXip9xGYmznF0T5cq1RbWJ4be3
/9K7yuWhuBYC3sbTbCneHBa91M82za+PIISc1ygCYtWSBoZKSAqLk0rkZpHaekDP
WqeUSNGYV//RunTeuRDAf5OxehERb1srzBXhRZ3cZdzXbgR/
-----END CERTIFICATE-----
`

	completeCheeseCrt = `Certificate:
    Data:
        Version: 3 (0x2)
        Serial Number: 1 (0x1)
        Signature Algorithm: sha1WithRSAEncryption
        Issuer: DC=org, DC=cheese, O=Cheese, O=Cheese 2, OU=Simple Signing Section, OU=Simple Signing Section 2, CN=Simple Signing CA, CN=Simple Signing CA 2, C=FR, C=US, L=TOULOUSE, L=LYON, ST=Signing State, ST=Signing State 2/emailAddress=simple@signing.com/emailAddress=simple2@signing.com
        Validity
            Not Before: Dec  6 11:10:16 2018 GMT
            Not After : Dec  5 11:10:16 2020 GMT
        Subject: DC=org, DC=cheese, O=Cheese, O=Cheese 2, OU=Simple Signing Section, OU=Simple Signing Section 2, CN=*.cheese.org, CN=*.cheese.com, C=FR, C=US, L=TOULOUSE, L=LYON, ST=Cheese org state, ST=Cheese com state/emailAddress=cert@cheese.org/emailAddress=cert@scheese.com
        Subject Public Key Info:
            Public Key Algorithm: rsaEncryption
                RSA Public-Key: (2048 bit)
                Modulus:
                    00:de:77:fa:8d:03:70:30:39:dd:51:1b:cc:60:db:
                    a9:5a:13:b1:af:fe:2c:c6:38:9b:88:0a:0f:8e:d9:
                    1b:a1:1d:af:0d:66:e4:13:5b:bc:5d:36:92:d7:5e:
                    d0:fa:88:29:d3:78:e1:81:de:98:b2:a9:22:3f:bf:
                    8a:af:12:92:63:d4:a9:c3:f2:e4:7e:d2:dc:a2:c5:
                    39:1c:7a:eb:d7:12:70:63:2e:41:47:e0:f0:08:e8:
                    dc:be:09:01:ec:28:09:af:35:d7:79:9c:50:35:d1:
                    6b:e5:87:7b:34:f6:d2:31:65:1d:18:42:69:6c:04:
                    11:83:fe:44:ae:90:92:2d:0b:75:39:57:62:e6:17:
                    2f:47:2b:c7:53:dd:10:2d:c9:e3:06:13:d2:b9:ba:
                    63:2e:3c:7d:83:6b:d6:89:c9:cc:9d:4d:bf:9f:e8:
                    a3:7b:da:c8:99:2b:ba:66:d6:8e:f8:41:41:a0:c9:
                    d0:5e:c8:11:a4:55:4a:93:83:87:63:04:63:41:9c:
                    fb:68:04:67:c2:71:2f:f2:65:1d:02:5d:15:db:2c:
                    d9:04:69:85:c2:7d:0d:ea:3b:ac:85:f8:d4:8f:0f:
                    c5:70:b2:45:e1:ec:b2:54:0b:e9:f7:82:b4:9b:1b:
                    2d:b9:25:d4:ab:ca:8f:5b:44:3e:15:dd:b8:7f:b7:
                    ee:f9
                Exponent: 65537 (0x10001)
        X509v3 extensions:
            X509v3 Key Usage: critical
                Digital Signature, Key Encipherment
            X509v3 Basic Constraints: 
                CA:FALSE
            X509v3 Extended Key Usage: 
                TLS Web Server Authentication, TLS Web Client Authentication
            X509v3 Subject Key Identifier: 
                94:BA:73:78:A2:87:FB:58:28:28:CF:98:3B:C2:45:70:16:6E:29:2F
            X509v3 Authority Key Identifier: 
                keyid:1E:52:A2:E8:54:D5:37:EB:D5:A8:1D:E4:C2:04:1D:37:E2:F7:70:03

            X509v3 Subject Alternative Name: 
                DNS:*.cheese.org, DNS:*.cheese.net, DNS:*.cheese.com, IP Address:10.0.1.0, IP Address:10.0.1.2, email:test@cheese.org, email:test@cheese.net
    Signature Algorithm: sha1WithRSAEncryption
         76:6b:05:b0:0e:34:11:b1:83:99:91:dc:ae:1b:e2:08:15:8b:
         16:b2:9b:27:1c:02:ac:b5:df:1b:d0:d0:75:a4:2b:2c:5c:65:
         ed:99:ab:f7:cd:fe:38:3f:c3:9a:22:31:1b:ac:8c:1c:c2:f9:
         5d:d4:75:7a:2e:72:c7:85:a9:04:af:9f:2a:cc:d3:96:75:f0:
         8e:c7:c6:76:48:ac:45:a4:b9:02:1e:2f:c0:15:c4:07:08:92:
         cb:27:50:67:a1:c8:05:c5:3a:b3:a6:48:be:eb:d5:59:ab:a2:
         1b:95:30:71:13:5b:0a:9a:73:3b:60:cc:10:d0:6a:c7:e5:d7:
         8b:2f:f9:2e:98:f2:ff:81:14:24:09:e3:4b:55:57:09:1a:22:
         74:f1:f6:40:13:31:43:89:71:0a:96:1a:05:82:1f:83:3a:87:
         9b:17:25:ef:5a:55:f2:2d:cd:0d:4d:e4:81:58:b6:e3:8d:09:
         62:9a:0c:bd:e4:e5:5c:f0:95:da:cb:c7:34:2c:34:5f:6d:fc:
         60:7b:12:5b:86:fd:df:21:89:3b:48:08:30:bf:67:ff:8c:e6:
         9b:53:cc:87:36:47:70:40:3b:d9:90:2a:d2:d2:82:c6:9c:f5:
         d1:d8:e0:e6:fd:aa:2f:95:7e:39:ac:fc:4e:d4:ce:65:b3:ec:
         c6:98:8a:31
-----BEGIN CERTIFICATE-----
MIIGWjCCBUKgAwIBAgIBATANBgkqhkiG9w0BAQUFADCCAYQxEzARBgoJkiaJk/Is
ZAEZFgNvcmcxFjAUBgoJkiaJk/IsZAEZFgZjaGVlc2UxDzANBgNVBAoMBkNoZWVz
ZTERMA8GA1UECgwIQ2hlZXNlIDIxHzAdBgNVBAsMFlNpbXBsZSBTaWduaW5nIFNl
Y3Rpb24xITAfBgNVBAsMGFNpbXBsZSBTaWduaW5nIFNlY3Rpb24gMjEaMBgGA1UE
AwwRU2ltcGxlIFNpZ25pbmcgQ0ExHDAaBgNVBAMME1NpbXBsZSBTaWduaW5nIENB
IDIxCzAJBgNVBAYTAkZSMQswCQYDVQQGEwJVUzERMA8GA1UEBwwIVE9VTE9VU0Ux
DTALBgNVBAcMBExZT04xFjAUBgNVBAgMDVNpZ25pbmcgU3RhdGUxGDAWBgNVBAgM
D1NpZ25pbmcgU3RhdGUgMjEhMB8GCSqGSIb3DQEJARYSc2ltcGxlQHNpZ25pbmcu
Y29tMSIwIAYJKoZIhvcNAQkBFhNzaW1wbGUyQHNpZ25pbmcuY29tMB4XDTE4MTIw
NjExMTAxNloXDTIwMTIwNTExMTAxNlowggF2MRMwEQYKCZImiZPyLGQBGRYDb3Jn
MRYwFAYKCZImiZPyLGQBGRYGY2hlZXNlMQ8wDQYDVQQKDAZDaGVlc2UxETAPBgNV
BAoMCENoZWVzZSAyMR8wHQYDVQQLDBZTaW1wbGUgU2lnbmluZyBTZWN0aW9uMSEw
HwYDVQQLDBhTaW1wbGUgU2lnbmluZyBTZWN0aW9uIDIxFTATBgNVBAMMDCouY2hl
ZXNlLm9yZzEVMBMGA1UEAwwMKi5jaGVlc2UuY29tMQswCQYDVQQGEwJGUjELMAkG
A1UEBhMCVVMxETAPBgNVBAcMCFRPVUxPVVNFMQ0wCwYDVQQHDARMWU9OMRkwFwYD
VQQIDBBDaGVlc2Ugb3JnIHN0YXRlMRkwFwYDVQQIDBBDaGVlc2UgY29tIHN0YXRl
MR4wHAYJKoZIhvcNAQkBFg9jZXJ0QGNoZWVzZS5vcmcxHzAdBgkqhkiG9w0BCQEW
EGNlcnRAc2NoZWVzZS5jb20wggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIB
AQDed/qNA3AwOd1RG8xg26laE7Gv/izGOJuICg+O2RuhHa8NZuQTW7xdNpLXXtD6
iCnTeOGB3piyqSI/v4qvEpJj1KnD8uR+0tyixTkceuvXEnBjLkFH4PAI6Ny+CQHs
KAmvNdd5nFA10Wvlh3s09tIxZR0YQmlsBBGD/kSukJItC3U5V2LmFy9HK8dT3RAt
yeMGE9K5umMuPH2Da9aJycydTb+f6KN72siZK7pm1o74QUGgydBeyBGkVUqTg4dj
BGNBnPtoBGfCcS/yZR0CXRXbLNkEaYXCfQ3qO6yF+NSPD8VwskXh7LJUC+n3grSb
Gy25JdSryo9bRD4V3bh/t+75AgMBAAGjgeAwgd0wDgYDVR0PAQH/BAQDAgWgMAkG
A1UdEwQCMAAwHQYDVR0lBBYwFAYIKwYBBQUHAwEGCCsGAQUFBwMCMB0GA1UdDgQW
BBSUunN4oof7WCgoz5g7wkVwFm4pLzAfBgNVHSMEGDAWgBQeUqLoVNU369WoHeTC
BB034vdwAzBhBgNVHREEWjBYggwqLmNoZWVzZS5vcmeCDCouY2hlZXNlLm5ldIIM
Ki5jaGVlc2UuY29thwQKAAEAhwQKAAECgQ90ZXN0QGNoZWVzZS5vcmeBD3Rlc3RA
Y2hlZXNlLm5ldDANBgkqhkiG9w0BAQUFAAOCAQEAdmsFsA40EbGDmZHcrhviCBWL
FrKbJxwCrLXfG9DQdaQrLFxl7Zmr983+OD/DmiIxG6yMHML5XdR1ei5yx4WpBK+f
KszTlnXwjsfGdkisRaS5Ah4vwBXEBwiSyydQZ6HIBcU6s6ZIvuvVWauiG5UwcRNb
CppzO2DMENBqx+XXiy/5Lpjy/4EUJAnjS1VXCRoidPH2QBMxQ4lxCpYaBYIfgzqH
mxcl71pV8i3NDU3kgVi2440JYpoMveTlXPCV2svHNCw0X238YHsSW4b93yGJO0gI
ML9n/4zmm1PMhzZHcEA72ZAq0tKCxpz10djg5v2qL5V+Oaz8TtTOZbPsxpiKMQ==
-----END CERTIFICATE-----
`

	minimalCert = `-----BEGIN CERTIFICATE-----
MIIDGTCCAgECCQCqLd75YLi2kDANBgkqhkiG9w0BAQsFADBYMQswCQYDVQQGEwJG
UjETMBEGA1UECAwKU29tZS1TdGF0ZTERMA8GA1UEBwwIVG91bG91c2UxITAfBgNV
BAoMGEludGVybmV0IFdpZGdpdHMgUHR5IEx0ZDAeFw0xODA3MTgwODI4MTZaFw0x
ODA4MTcwODI4MTZaMEUxCzAJBgNVBAYTAkZSMRMwEQYDVQQIDApTb21lLVN0YXRl
MSEwHwYDVQQKDBhJbnRlcm5ldCBXaWRnaXRzIFB0eSBMdGQwggEiMA0GCSqGSIb3
DQEBAQUAA4IBDwAwggEKAoIBAQC/+frDMMTLQyXG34F68BPhQq0kzK4LIq9Y0/gl
FjySZNn1C0QDWA1ubVCAcA6yY204I9cxcQDPNrhC7JlS5QA8Y5rhIBrqQlzZizAi
Rj3NTrRjtGUtOScnHuJaWjLy03DWD+aMwb7q718xt5SEABmmUvLwQK+EjW2MeDwj
y8/UEIpvrRDmdhGaqv7IFpIDkcIF7FowJ/hwDvx3PMc+z/JWK0ovzpvgbx69AVbw
ZxCimeha65rOqVi+lEetD26le+WnOdYsdJ2IkmpPNTXGdfb15xuAc+gFXfMCh7Iw
3Ynl6dZtZM/Ok2kiA7/OsmVnRKkWrtBfGYkI9HcNGb3zrk6nAgMBAAEwDQYJKoZI
hvcNAQELBQADggEBAC/R+Yvhh1VUhcbK49olWsk/JKqfS3VIDQYZg1Eo+JCPbwgS
I1BSYVfMcGzuJTX6ua3m/AHzGF3Tap4GhF4tX12jeIx4R4utnjj7/YKkTvuEM2f4
xT56YqI7zalGScIB0iMeyNz1QcimRl+M/49au8ow9hNX8C2tcA2cwd/9OIj/6T8q
SBRHc6ojvbqZSJCO0jziGDT1L3D+EDgTjED4nd77v/NRdP+egb0q3P0s4dnQ/5AV
aQlQADUn61j3ScbGJ4NSeZFFvsl38jeRi/MEzp0bGgNBcPj6JHi7qbbauZcZfQ05
jECvgAY7Nfd9mZ1KtyNaW31is+kag7NsvjxU/kM=
-----END CERTIFICATE-----`
)

var next = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	_, err := w.Write([]byte("bar"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
})

func TestPassTLSClientCert_PEM(t *testing.T) {
	testCases := []struct {
		desc           string
		certContents   []string // set the request TLS attribute if defined
		config         dynamic.PassTLSClientCert
		expectedHeader string
	}{
		{
			desc: "No TLS, no option",
		},
		{
			desc:         "TLS, no option",
			certContents: []string{minimalCheeseCrt},
		},
		{
			desc:   "No TLS, with pem option true",
			config: dynamic.PassTLSClientCert{PEM: true},
		},
		{
			desc:           "TLS with simple certificate, with pem option true",
			certContents:   []string{minimalCheeseCrt},
			config:         dynamic.PassTLSClientCert{PEM: true},
			expectedHeader: getCleanCertContents([]string{minimalCert}),
		},
		{
			desc:           "TLS with complete certificate, with pem option true",
			certContents:   []string{minimalCheeseCrt},
			config:         dynamic.PassTLSClientCert{PEM: true},
			expectedHeader: getCleanCertContents([]string{minimalCheeseCrt}),
		},
		{
			desc:           "TLS with two certificate, with pem option true",
			certContents:   []string{minimalCert, minimalCheeseCrt},
			config:         dynamic.PassTLSClientCert{PEM: true},
			expectedHeader: getCleanCertContents([]string{minimalCert, minimalCheeseCrt}),
		},
	}

	for _, test := range testCases {
		tlsClientHeaders, err := New(context.Background(), next, test.config, "foo")
		require.NoError(t, err)

		res := httptest.NewRecorder()
		req := testhelpers.MustNewRequest(http.MethodGet, "http://example.com/foo", nil)

		if test.certContents != nil && len(test.certContents) > 0 {
			req.TLS = buildTLSWith(test.certContents)
		}

		tlsClientHeaders.ServeHTTP(res, req)

		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, http.StatusOK, res.Code, "Http Status should be OK")
			assert.Equal(t, "bar", res.Body.String(), "Should be the expected body")

			if test.expectedHeader != "" {
				expected := getCleanCertContents(test.certContents)
				assert.Equal(t, expected, req.Header.Get(xForwardedTLSClientCert), "The request header should contain the cleaned certificate")
			} else {
				assert.Empty(t, req.Header.Get(xForwardedTLSClientCert))
			}

			assert.Empty(t, res.Header().Get(xForwardedTLSClientCert), "The response header should be always empty")
		})
	}
}

func TestPassTLSClientCert_certInfo(t *testing.T) {
	minimalCheeseCertAllInfo := strings.Join([]string{
		`Subject="C=FR,ST=Some-State,O=Cheese"`,
		`Issuer="DC=org,DC=cheese,C=FR,C=US,ST=Signing State,ST=Signing State 2,L=TOULOUSE,L=LYON,O=Cheese,O=Cheese 2,OU=Simple Signing Section,OU=Simple Signing Section 2,CN=Simple Signing CA 2"`,
		`SerialNumber="481535886039632329873080491016862977516759989652"`,
		`NB="1544094636"`,
		`NA="1632568236"`,
	}, fieldSeparator)

	completeCertAllInfo := strings.Join([]string{
		`Subject="DC=org,DC=cheese,C=FR,C=US,ST=Cheese org state,ST=Cheese com state,L=TOULOUSE,L=LYON,O=Cheese,O=Cheese 2,OU=Simple Signing Section,OU=Simple Signing Section 2,CN=*.cheese.com"`,
		`Issuer="DC=org,DC=cheese,C=FR,C=US,ST=Signing State,ST=Signing State 2,L=TOULOUSE,L=LYON,O=Cheese,O=Cheese 2,CN=Simple Signing CA 2"`,
		`SerialNumber="1"`,
		`NB="1544094616"`,
		`NA="1607166616"`,
		`SAN="*.cheese.org,*.cheese.net,*.cheese.com,test@cheese.org,test@cheese.net,10.0.1.0,10.0.1.2"`,
	}, fieldSeparator)

	testCases := []struct {
		desc           string
		certContents   []string // set the request TLS attribute if defined
		config         dynamic.PassTLSClientCert
		expectedHeader string
	}{
		{
			desc: "No TLS, no option",
		},
		{
			desc:         "TLS, no option",
			certContents: []string{minimalCert},
		},
		{
			desc: "No TLS, with subject info",
			config: dynamic.PassTLSClientCert{
				Info: &dynamic.TLSClientCertificateInfo{
					Subject: &dynamic.TLSCLientCertificateSubjectDNInfo{
						CommonName:         true,
						Organization:       true,
						OrganizationalUnit: true,
						Locality:           true,
						Province:           true,
						Country:            true,
						SerialNumber:       true,
					},
				},
			},
		},
		{
			desc: "No TLS, with pem option false with empty subject info",
			config: dynamic.PassTLSClientCert{
				PEM: false,
				Info: &dynamic.TLSClientCertificateInfo{
					Subject: &dynamic.TLSCLientCertificateSubjectDNInfo{},
				},
			},
		},
		{
			desc:         "TLS with simple certificate, with all info",
			certContents: []string{minimalCheeseCrt},
			config: dynamic.PassTLSClientCert{
				Info: &dynamic.TLSClientCertificateInfo{
					NotAfter:     true,
					NotBefore:    true,
					Sans:         true,
					SerialNumber: true,
					Subject: &dynamic.TLSCLientCertificateSubjectDNInfo{
						CommonName:         true,
						Country:            true,
						DomainComponent:    true,
						Locality:           true,
						Organization:       true,
						OrganizationalUnit: true, // The simple subject has no OU
						Province:           true,
						SerialNumber:       true,
					},
					Issuer: &dynamic.TLSCLientCertificateIssuerDNInfo{
						CommonName:      true,
						Country:         true,
						DomainComponent: true,
						Locality:        true,
						Organization:    true,
						Province:        true,
						SerialNumber:    true,
					},
				},
			},
			expectedHeader: minimalCheeseCertAllInfo,
		},
		{
			desc:         "TLS with simple certificate, with some info",
			certContents: []string{minimalCheeseCrt},
			config: dynamic.PassTLSClientCert{
				Info: &dynamic.TLSClientCertificateInfo{
					NotAfter: true,
					Sans:     true,
					Subject: &dynamic.TLSCLientCertificateSubjectDNInfo{
						Organization: true,
					},
					Issuer: &dynamic.TLSCLientCertificateIssuerDNInfo{
						Country: true,
					},
				},
			},
			expectedHeader: `Subject="O=Cheese";Issuer="C=FR,C=US";NA="1632568236"`,
		},
		{
			desc:         "TLS with simple certificate, requesting non-existant info",
			certContents: []string{minimalCheeseCrt},
			config: dynamic.PassTLSClientCert{
				Info: &dynamic.TLSClientCertificateInfo{
					NotAfter: true,
					Sans:     true,
					Subject: &dynamic.TLSCLientCertificateSubjectDNInfo{
						Organization: true,
						// OrganizationalUnit is not set on this example certificate,
						// so even though it's requested, it will be absent.
						OrganizationalUnit: true,
					},
					Issuer: &dynamic.TLSCLientCertificateIssuerDNInfo{
						Country: true,
					},
				},
			},
			expectedHeader: `Subject="O=Cheese";Issuer="C=FR,C=US";NA="1632568236"`,
		},
		{
			desc:         "TLS with complete certificate, with all info",
			certContents: []string{completeCheeseCrt},
			config: dynamic.PassTLSClientCert{
				Info: &dynamic.TLSClientCertificateInfo{
					NotAfter:     true,
					NotBefore:    true,
					Sans:         true,
					SerialNumber: true,
					Subject: &dynamic.TLSCLientCertificateSubjectDNInfo{
						Country:            true,
						Province:           true,
						Locality:           true,
						Organization:       true,
						OrganizationalUnit: true,
						CommonName:         true,
						SerialNumber:       true,
						DomainComponent:    true,
					},
					Issuer: &dynamic.TLSCLientCertificateIssuerDNInfo{
						Country:         true,
						Province:        true,
						Locality:        true,
						Organization:    true,
						CommonName:      true,
						SerialNumber:    true,
						DomainComponent: true,
					},
				},
			},
			expectedHeader: completeCertAllInfo,
		},
		{
			desc:         "TLS with 2 certificates, with all info",
			certContents: []string{minimalCheeseCrt, completeCheeseCrt},
			config: dynamic.PassTLSClientCert{
				Info: &dynamic.TLSClientCertificateInfo{
					NotAfter:     true,
					NotBefore:    true,
					Sans:         true,
					SerialNumber: true,
					Subject: &dynamic.TLSCLientCertificateSubjectDNInfo{
						Country:            true,
						Province:           true,
						Locality:           true,
						Organization:       true,
						OrganizationalUnit: true,
						CommonName:         true,
						SerialNumber:       true,
						DomainComponent:    true,
					},
					Issuer: &dynamic.TLSCLientCertificateIssuerDNInfo{
						Country:         true,
						Province:        true,
						Locality:        true,
						Organization:    true,
						CommonName:      true,
						SerialNumber:    true,
						DomainComponent: true,
					},
				},
			},
			expectedHeader: strings.Join([]string{minimalCheeseCertAllInfo, completeCertAllInfo}, certSeparator),
		},
	}

	for _, test := range testCases {
		tlsClientHeaders, err := New(context.Background(), next, test.config, "foo")
		require.NoError(t, err)

		res := httptest.NewRecorder()
		req := testhelpers.MustNewRequest(http.MethodGet, "http://example.com/foo", nil)

		if test.certContents != nil && len(test.certContents) > 0 {
			req.TLS = buildTLSWith(test.certContents)
		}

		tlsClientHeaders.ServeHTTP(res, req)

		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, http.StatusOK, res.Code, "Http Status should be OK")
			assert.Equal(t, "bar", res.Body.String(), "Should be the expected body")

			if test.expectedHeader != "" {
				unescape, err := url.QueryUnescape(req.Header.Get(xForwardedTLSClientCertInfo))
				require.NoError(t, err)
				assert.Equal(t, test.expectedHeader, unescape, "The request header should contain the cleaned certificate")
			} else {
				assert.Empty(t, req.Header.Get(xForwardedTLSClientCertInfo))
			}

			assert.Empty(t, res.Header().Get(xForwardedTLSClientCertInfo), "The response header should be always empty")
		})
	}
}

func Test_sanitize(t *testing.T) {
	testCases := []struct {
		desc       string
		toSanitize []byte
		expected   string
	}{
		{
			desc: "Empty",
		},
		{
			desc:       "With a minimal cert",
			toSanitize: []byte(minimalCheeseCrt),
			expected: `MIIEQDCCAygCFFRY0OBk/L5Se0IZRj3CMljawL2UMA0GCSqGSIb3DQEBCwUAMIIB
hDETMBEGCgmSJomT8ixkARkWA29yZzEWMBQGCgmSJomT8ixkARkWBmNoZWVzZTEP
MA0GA1UECgwGQ2hlZXNlMREwDwYDVQQKDAhDaGVlc2UgMjEfMB0GA1UECwwWU2lt
cGxlIFNpZ25pbmcgU2VjdGlvbjEhMB8GA1UECwwYU2ltcGxlIFNpZ25pbmcgU2Vj
dGlvbiAyMRowGAYDVQQDDBFTaW1wbGUgU2lnbmluZyBDQTEcMBoGA1UEAwwTU2lt
cGxlIFNpZ25pbmcgQ0EgMjELMAkGA1UEBhMCRlIxCzAJBgNVBAYTAlVTMREwDwYD
VQQHDAhUT1VMT1VTRTENMAsGA1UEBwwETFlPTjEWMBQGA1UECAwNU2lnbmluZyBT
dGF0ZTEYMBYGA1UECAwPU2lnbmluZyBTdGF0ZSAyMSEwHwYJKoZIhvcNAQkBFhJz
aW1wbGVAc2lnbmluZy5jb20xIjAgBgkqhkiG9w0BCQEWE3NpbXBsZTJAc2lnbmlu
Zy5jb20wHhcNMTgxMjA2MTExMDM2WhcNMjEwOTI1MTExMDM2WjAzMQswCQYDVQQG
EwJGUjETMBEGA1UECAwKU29tZS1TdGF0ZTEPMA0GA1UECgwGQ2hlZXNlMIIBIjAN
BgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAskX/bUtwFo1gF2BTPNaNcTUMaRFu
FMZozK8IgLjccZ4kZ0R9oFO6Yp8Zl/IvPaf7tE26PI7XP7eHriUdhnQzX7iioDd0
RZa68waIhAGc+xPzRFrP3b3yj3S2a9Rve3c0K+SCV+EtKAwsxMqQDhoo9PcBfo5B
RHfht07uD5MncUcGirwN+/pxHV5xzAGPcc7On0/5L7bq/G+63nhu78zw9XyuLaHC
PM5VbOUvpyIESJHbMMzTdFGL8ob9VKO+Kr1kVGdEA9i8FLGl3xz/GBKuW/JD0xyW
DrU29mri5vYWHmkuv7ZWHGXnpXjTtPHwveE9/0/ArnmpMyR9JtqFr1oEvQIDAQAB
MA0GCSqGSIb3DQEBCwUAA4IBAQBHta+NWXI08UHeOkGzOTGRiWXsOH2dqdX6gTe9
xF1AIjyoQ0gvpoGVvlnChSzmlUj+vnx/nOYGIt1poE3hZA3ZHZD/awsvGyp3GwWD
IfXrEViSCIyF+8tNNKYyUcEO3xdAsAUGgfUwwF/mZ6MBV5+A/ZEEILlTq8zFt9dV
vdKzIt7fZYxYBBHFSarl1x8pDgWXlf3hAufevGJXip9xGYmznF0T5cq1RbWJ4be3
/9K7yuWhuBYC3sbTbCneHBa91M82za+PIISc1ygCYtWSBoZKSAqLk0rkZpHaekDP
WqeUSNGYV//RunTeuRDAf5OxehERb1srzBXhRZ3cZdzXbgR/`,
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			content := sanitize(test.toSanitize)

			expected := url.QueryEscape(strings.ReplaceAll(test.expected, "\n", ""))
			assert.Equal(t, expected, content, "The sanitized certificates should be equal")
		})
	}
}

func Test_getSANs(t *testing.T) {
	urlFoo := testhelpers.MustParseURL("my.foo.com")
	urlBar := testhelpers.MustParseURL("my.bar.com")

	testCases := []struct {
		desc     string
		cert     *x509.Certificate // set the request TLS attribute if defined
		expected []string
	}{
		{
			desc: "With nil",
		},
		{
			desc: "Certificate without Sans",
			cert: &x509.Certificate{},
		},
		{
			desc: "Certificate with all Sans",
			cert: &x509.Certificate{
				DNSNames:       []string{"foo", "bar"},
				EmailAddresses: []string{"test@test.com", "test2@test.com"},
				IPAddresses:    []net.IP{net.IPv4(10, 0, 0, 1), net.IPv4(10, 0, 0, 2)},
				URIs:           []*url.URL{urlFoo, urlBar},
			},
			expected: []string{"foo", "bar", "test@test.com", "test2@test.com", "10.0.0.1", "10.0.0.2", urlFoo.String(), urlBar.String()},
		},
	}

	for _, test := range testCases {
		sans := getSANs(test.cert)

		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			if len(test.expected) > 0 {
				for i, expected := range test.expected {
					assert.Equal(t, expected, sans[i])
				}
			} else {
				assert.Empty(t, sans)
			}
		})
	}
}

func getCleanCertContents(certContents []string) string {
	exp := regexp.MustCompile("-----BEGIN CERTIFICATE-----(?s)(.*)")

	var cleanedCertContent []string
	for _, certContent := range certContents {
		cert := sanitize([]byte(exp.FindString(certContent)))
		cleanedCertContent = append(cleanedCertContent, cert)
	}

	return strings.Join(cleanedCertContent, certSeparator)
}

func buildTLSWith(certContents []string) *tls.ConnectionState {
	var peerCertificates []*x509.Certificate

	for _, certContent := range certContents {
		peerCertificates = append(peerCertificates, getCertificate(certContent))
	}

	return &tls.ConnectionState{PeerCertificates: peerCertificates}
}

func getCertificate(certContent string) *x509.Certificate {
	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM([]byte(signingCA))
	if !ok {
		panic("failed to parse root certificate")
	}

	block, _ := pem.Decode([]byte(certContent))
	if block == nil {
		panic("failed to parse certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		panic("failed to parse certificate: " + err.Error())
	}

	return cert
}
