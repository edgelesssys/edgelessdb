ertgo build -buildmode=c-archive -tags enclave -o $1 -ldflags "-X main.version=$2 -X main.gitCommit=`git rev-parse HEAD`"
