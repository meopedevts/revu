#!/usr/bin/env bash
# Empacota build/bin/revu em tarball linux-amd64 + sha256, gera dist/.
# Chamado pelo @semantic-release/exec durante o ciclo de release.
#
# Uso: bash scripts/package-release.sh <version>
#   <version> — string sem prefixo "v" (ex.: "0.2.0"). Tag final = "v<version>".

set -euo pipefail

VERSION="${1:?usage: package-release.sh <version>}"
STAGE="revu-v${VERSION}-linux-amd64"
DIST="dist"
BIN="build/bin/revu"

if [ ! -x "${BIN}" ]; then
  echo "::error::binary not found at ${BIN} — run 'task release' first" >&2
  exit 1
fi

mkdir -p "${DIST}"
rm -rf "${DIST}/${STAGE}" "${DIST}/${STAGE}.tar.gz" "${DIST}/${STAGE}.tar.gz.sha256"

mkdir -p "${DIST}/${STAGE}"
cp "${BIN}" "${DIST}/${STAGE}/"
cp README.md "${DIST}/${STAGE}/"
if [ -f LICENSE ]; then
  cp LICENSE "${DIST}/${STAGE}/"
fi

tar -C "${DIST}" -czf "${DIST}/${STAGE}.tar.gz" "${STAGE}"
(cd "${DIST}" && sha256sum "${STAGE}.tar.gz" > "${STAGE}.tar.gz.sha256")

echo "Packaged: ${DIST}/${STAGE}.tar.gz"
ls -lh "${DIST}/${STAGE}.tar.gz" "${DIST}/${STAGE}.tar.gz.sha256"
