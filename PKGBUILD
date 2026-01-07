# Maintainer: Brice <b@bnema.dev>
pkgname=turtle-wow-launcher-git
pkgver=r2.795e83c
pkgrel=1
pkgdesc='A clean Go CLI wrapper for the Turtle WoW AppImage launcher on Linux'
arch=('x86_64')
url="https://github.com/bnema/turtle-wow-launcher"
license=('MIT')
makedepends=('go' 'git')
provides=('turtle-wow-launcher')
conflicts=('turtle-wow-launcher')
source=("$pkgname::git+$url.git")
sha256sums=('SKIP')

pkgver() {
  cd "$pkgname"
  printf "r%s.%s" "$(git rev-list --count HEAD)" "$(git rev-parse --short HEAD)"
}

prepare() {
  cd "$pkgname"
  mkdir -p build/
}

build() {
  cd "$pkgname"
  export CGO_CPPFLAGS="${CPPFLAGS}"
  export CGO_CFLAGS="${CFLAGS}"
  export CGO_CXXFLAGS="${CXXFLAGS}"
  export CGO_LDFLAGS="${LDFLAGS}"
  export GOFLAGS="-buildmode=pie -trimpath -ldflags=-linkmode=external -mod=readonly -modcacherw"
  go build -o build/turtle-wow .
}

check() {
  cd "$pkgname"
  go test ./...
}

package() {
  cd "$pkgname"
  install -Dm755 build/turtle-wow "$pkgdir/usr/bin/turtle-wow"
  install -Dm644 LICENSE "$pkgdir/usr/share/licenses/$pkgname/LICENSE"
}
