CGO_ENABLED=0 GOOS=linux  GOARCH=amd64 go build  -o ./abrForLinux
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build  -o ./abrForMac_amd
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build  -o ./abrForMac_arm