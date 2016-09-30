go build -ldflags "-X main.date=`date -u +%Y%m%d-%H%M%S`"
env CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc go build -ldflags "-X main.date=`date -u +%Y%m%d-%H%M%S`" 

