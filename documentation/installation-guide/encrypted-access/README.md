This section provides information about TLS-based encryption in OpenSearch service.

Topics covered in this section:
- [Common Information](#common-information)
- [Examples](#examples)
- [Certificate Renewal](#certificate-renewal)

# Common Information

You can enable `Transport Layer Security` (TLS) encryption for communication with OpenSearch.

OpenSearch uses TLS encryption across the cluster to verify authenticity of the servers and clients that connect. All (`transport`, `admin` and `rest`) layers of OpenSearch are always encrypted, you can only configure the way of certificates generation.

You can enable TLS encryption for OpenSearch DBaaS Adapter when `global.tls.enabled` and `dbaasAdapter.tls.enabled` parameters are set to `true`

# Examples

## Minimal example

Minimal parameters to enable TLS encryption for components of OpenSearch service are as follows:

```yaml
global:
  tls:
    enabled: true
    generateCerts:
      enabled: true
      certProvider: cert-manager
      clusterIssuerName: CLUSTER_ISSUER_NAME
```

where `CLUSTER_ISSUER_NAME` is the name of pre-created `ClusterIssuer` resource for certificates generation.

**Important**: Production environment requires `ClusterIssuer` for certificates generation.

## Full example

Full list of parameters to enable and configure TLS encryption for components of OpenSearch service is as follows:

```yaml
global:
  tls:
    enabled: true
    cipherSuites: []
    generateCerts:
      enabled: true
      certProvider: cert-manager
      durationDays: 365
      clusterIssuerName: ""

  disasterRecovery:
    tls:
      enabled: true
      secretName: ""
      cipherSuites: []
      subjectAlternativeName:
        additionalDnsNames: []
        additionalIpAddresses: []

opensearch:
   tls:
      enabled: true
      cipherSuites: []
      subjectAlternativeName:
         additionalDnsNames: []
         additionalIpAddresses: []
      transport:
         existingCertSecret:
         existingCertSecretCertSubPath: transport-crt.pem
         existingCertSecretKeySubPath: transport-key.pem
         existingCertSecretRootCASubPath: transport-root-ca.pem
      rest:
         existingCertSecret:
         existingCertSecretCertSubPath: rest-crt.pem
         existingCertSecretKeySubPath: rest-key.pem
         existingCertSecretRootCASubPath: rest-root-ca.pem
      admin:
         existingCertSecret:
         existingCertSecretCertSubPath: admin-crt.pem
         existingCertSecretKeySubPath: admin-key.pem
         existingCertSecretRootCASubPath: admin-root-ca.pem

curator:
   tls:
      enabled: true
      secretName: ""
      subjectAlternativeName:
         additionalDnsNames: []
         additionalIpAddresses: []
dbaasAdapter:
   tls:
      enabled: true
```

**Important**: Production environment requires `ClusterIssuer` for certificates generation.

# Certificate Renewal

`CertManager` automatically renews certificates. It calculates the time to renew a certificate based on the issued `X.509` certificate's duration and a `renewBefore` value. The `renewBefore` parameter specifies how long before expiry a certificate should be renewed. By default, the value of `renewBefore` parameter is `2/3` through the `X.509` certificate's duration. For more information, see [Cert Manager Renewal](https://cert-manager.io/docs/usage/certificate/#renewal). After certificate renewed by `CertManager` the secret contains new certificate, but running applications store previous certificate in pods. As `CertManager` generates new certificates before old expired the both certificates are valid for some time (`renewBefore`). OpenSearch service does not have any handlers for certificates secret changes, so you need to manually restart **all** OpenSearch service pods before old certificate is expired.