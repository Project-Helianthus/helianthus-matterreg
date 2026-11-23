# Development attestation for the 0.7 Matter bridge

The 0.7 Matter bridge ships with development attestation, uncertified. It uses
the go-fabric test certificates for the 0.7 development cycle only; own
certificates come afterwards.

## What the bridge uses

- The bridge node takes the zero value of `node.Config.Attestation`, which is the
  go-fabric CSA test chain and test Certification Declaration. Both are built for
  the vendor and product in the configured product description
  (`attestation.BuildTestChainAt` and `attestation.BuildTestCertificationDeclaration`).
- The chain is rooted at the VID-less CSA test PAA (`TestPAANoVIDCert`).
- It is pairable only with a commissioner that ships the CSA test trust store.
- It is not shippable. No operator, product or certification statement follows
  from it.

## Boundaries

- Uncertified. The attestation is a development artifact, not a Matter
  certification, conformance result or interoperability claim.
- Development cycle only. Own certificates replace these test certificates after
  the 0.7 development cycle; this page does not define them.
- No secret material. The test chain is the go-fabric test material; this
  repository adds no private key, certificate, serial or device identifier.
- Recorded levels. A bench, hosted or hardware run that pairs with this
  attestation is a recorded level and blocks nothing (rule 12).

Support for the mapped endpoint is declared from the specification mapping
(see [gateway library composition](gateway-library-composition.md)); this page
adds no admission condition.
