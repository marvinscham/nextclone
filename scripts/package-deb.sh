#!/bin/sh
set -eu

version="${1:?version is required}"

mkdir -p package/DEBIAN package/usr/lib/nextclone package/usr/bin package/usr/share/applications

cp dist/nextclone-linux-amd64 package/usr/lib/nextclone/nextclone
cp dist/rclone-linux-amd64 package/usr/lib/nextclone/rclone
cp packaging/linux/nextclone package/usr/bin/nextclone
cp packaging/linux/nextclone.desktop package/usr/share/applications/nextclone.desktop

chmod 0755 package/usr/lib/nextclone/nextclone package/usr/lib/nextclone/rclone package/usr/bin/nextclone
chmod 0644 package/usr/share/applications/nextclone.desktop

cat > package/DEBIAN/control <<EOF
Package: nextclone
Version: ${version}
Section: utils
Priority: optional
Architecture: amd64
Maintainer: Nextclone maintainers
Depends: libc6, libgl1, libx11-6, libxcursor1, libxrandr2, libxinerama1, libxi6, libxxf86vm1, libxkbcommon0
Description: GUI for local-to-Nextcloud copy and sync jobs using rclone
 Nextclone provides a simple desktop dashboard for manual and scheduled
 independent local-to-Nextcloud jobs powered by a bundled rclone binary.
EOF

artifact_version="${GITHUB_REF_NAME:-${CI_COMMIT_TAG:-$version}}"
dpkg-deb --build package "dist/nextclone_${artifact_version}_linux_amd64.deb"
