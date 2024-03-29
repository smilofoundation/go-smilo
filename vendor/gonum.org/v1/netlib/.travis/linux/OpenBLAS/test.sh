set -ex

go env
go get -d -t -v ./...
export CGO_LDFLAGS="-L/usr/lib -lopenblas"
go test -a -v ./...
if [[ $TRAVIS_SECURE_ENV_VARS = "true" ]]; then bash -c "$GOPATH/src/gonum.org/v1/netlib/.travis/test-coverage.sh"; fi

# travis compiles commands in script and then executes in bash.  By adding
# set -e we are changing the travis build script's behavior, and the set
# -e lives on past the commands we are providing it.  Some of the travis
# commands are supposed to exit with non zero status, but then continue
# executing.  set -x makes the travis log files extremely verbose and
# difficult to understand.
# 
# see travis-ci/travis-ci#5120
set +ex
