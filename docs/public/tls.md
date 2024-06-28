This section provides information about TLS-based encryption in OpenSearch service.

[[_TOC_]]

# Common Information

You can configure `Transport Layer Security` (TLS) encryption for communication with OpenSearch.

OpenSearch uses TLS encryption across the cluster to verify authenticity of the servers and clients that connect.
The `transport` and `admin` layers of OpenSearch are always encrypted, you can only configure the way of certificates generation.

By default, the `rest` layer is also TLS encrypted as recommended option from OpenSearch Team,
because OpenSearch does not support some types of security configurations on REST layer without encryption.
To disable it you can put the property `opensearch.tls.enabled` to `false`, but pay attention on [Migration Guide](/docs/public/installation.md#migration-to-opensearch-2x-opensearch-service-1xx).

You can enable TLS encryption for OpenSearch DBaaS Adapter when `global.tls.enabled` and `dbaasAdapter.tls.enabled` parameters are set to `true`
and `dbaasAdapter.dbaasAggregatorRegistrationAddress` contains `https` address.

**Important:** By default OpenSearch is deployed with self-signed certificates for development purposes. To integration with Cert Manager please follow the example sections below.

# SSL Configuration using CertManager

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

# SSL Configuration using parameters with manually generated Certificates

You can automatically generate TLS-based secrets using Helm by specifying certificates in deployment parameters. For example, to generate `opensearch-drd-tls-secret` :

1. Following certificates should be generated in BASE64 format:

   ```yaml
    ca.crt: ${ROOT_CA_CERTIFICATE}
    tls.crt: ${CERTIFICATE}
    tls.key: ${PRIVATE_KEY}
   ```

   Where:
   * `${ROOT_CA_CERTIFICATE}` is the root CA in BASE64 format.
   * `${CERTIFICATE}` is the certificate in BASE64 format.
   * `${PRIVATE_KEY}` is the private key in BASE64 format.

2. Specify the certificates and other deployment parameters:

   ```yaml
    global:
      tls:
        enabled: true
        cipherSuites: []
        generateCerts:
          enabled: false
          certProvider: dev
          clusterIssuerName: ""

      disasterRecovery:
        tls:
          enabled: true
          certificates:
            crt: LS0tLS1CRUdJTiBSU0E...  
            key: LS0tLS1CRUdJTiBSU0EgUFJJV...
            ca: LS0tLS1CRUdJTiBSU0E...
          secretName: "opensearch-drd-tls-secret"
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
          certificates:
            crt: LS0tLS1CRUdJTiBSU0E...  
            key: LS0tLS1CRUdJTiBSU0EgUFJJV...
            ca: LS0tLS1CRUdJTiBSU0E...
        rest:
          certificates:
            crt: LS0tLS1CRUdJTiBSU0E...  
            key: LS0tLS1CRUdJTiBSU0EgUFJJV...
            ca: LS0tLS1CRUdJTiBSU0E...
        admin:
          certificates:
            crt: LS0tLS1CRUdJTiBSU0E...  
            key: LS0tLS1CRUdJTiBSU0EgUFJJV...
            ca: LS0tLS1CRUdJTiBSU0E...

    curator:
      tls:
        enabled: true
        certificates:
          crt: LS0tLS1CRUdJTiBSU0E...  
          key: LS0tLS1CRUdJTiBSU0EgUFJJV...
          ca: LS0tLS1CRUdJTiBSU0E...
        secretName: "opensearch-curator-tls-secret"
        subjectAlternativeName:
          additionalDnsNames: []
          additionalIpAddresses: []
    dbaasAdapter:
      tls:
        enabled: true
        certificates:
          crt: LS0tLS1CRUdJTiBSU0E...  
          key: LS0tLS1CRUdJTiBSU0EgUFJJV...
          ca: LS0tLS1CRUdJTiBSU0E...
        secretName: "dbaas-opensearch-adapter-tls-secret"
   ```

**Pay attention**, when you upgrade OpenSearch from non TLS installation to TLS with manually specified
certificates you need to delete `<fullname>-admin-certs`, `<fullname>-admin-certs` and `<fullname>-admin-certs`
secrets manually before upgrade, where `<fullname>` is the OpenSearch full name.

# Certificate Renewal

`CertManager` automatically renews certificates.
It calculates the time to renew a certificate based on the issued `X.509` certificate's duration and a `renewBefore` value.
The `renewBefore` parameter specifies how long before expiry a certificate should be renewed.
By default, the value of `renewBefore` parameter is `2/3` through the `X.509` certificate's duration.
For more information, see [Cert Manager Renewal](https://cert-manager.io/docs/usage/certificate/#renewal).
After certificate renewed by `CertManager` the secret contains new certificate, but running applications store previous certificate in pods.
As `CertManager` generates new certificates before old expired the both certificates are valid for some time (`renewBefore`).
OpenSearch service does not have any handlers for certificates secret changes, so you need to manually restart **all** OpenSearch service pods before old certificate is expired.

# Certificate Import On Client Side

When certificate is generated by Cert Manager you need to import Cert Manager's cluster CA certificate
to Application Deployer to make it possible to add that certificate to every application namespace `defaultsslcertificate` secret.
