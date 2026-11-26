# Security policy

## Reporting a vulnerability

Report a vulnerability privately at <https://github.com/UnstoppableMango/tdl/security/advisories/new>.
Never report one in a public issue, discussion, or pull request.

Say what is affected, how to reproduce it, and what an attacker gains.
A first response comes within 14 days.

## Disclosure

Disclosure is coordinated through a GitHub security advisory.
The advisory stays private while a fix is built, and is published once a release carrying the fix is out.
A reporter who wants credit is named in it.

## Supported versions

Only the latest release is supported.
A fix ships as a new release, not as a patch to an older one.

## Verifying a download

Every release asset is attested, and `SHA256SUMS` lists each one's checksum:

```shell
sha256sum --ignore-missing -c SHA256SUMS
gh attestation verify "$asset" --repo UnstoppableMango/tdl
```

The README's [release assets](README.md#release-assets) section downloads and verifies one end to end.
