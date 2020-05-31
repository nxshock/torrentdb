pkgname=torrentdb
pkgver=0.0.1
pkgrel=0
pkgdesc="Torrent database"
arch=('x86_64' 'aarch64')
license=('MIT')
makedepends=('go')
options=("!strip")
backup=("etc/$pkgname.toml")
source=("git://github.com/nxshock/$pkgname.git")
sha256sums=('SKIP')

build() {
    cd "$srcdir/$pkgname"
    export GOFLAGS="-buildmode=pie -trimpath"
    go build -o $pkgname -ldflags "-s -w"
}

package() {
    cd "$srcdir/$pkgname"
    install -Dm755 "$pkgname"          "$pkgdir/usr/bin/$pkgname"
    install -Dm644 "$pkgname.toml"     "$pkgdir/etc/$pkgname.toml"
    install -Dm644 "$pkgname.service"  "$pkgdir/usr/lib/systemd/system/$pkgname.service"
    install -Dm644 "$pkgname.sysusers" "$pkgdir/usr/lib/sysusers.d/$pkgname.conf"

    cp -R "site/" "$pkgdir/usr/lib/$pkgname"
}
