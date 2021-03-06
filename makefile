preparefolders:
	if not exist "bin" mkdir "bin"
	if not exist "bin/windows" mkdir "bin/windows"
	if not exist "bin/linux-amd64" mkdir "bin/linux-amd64"
	if not exist "bin/linux-arm64" mkdir "bin/linux-arm64"
	if not exist "bin/linux-arm32" mkdir "bin/linux-arm32"
	if not exist "bin/linux-mipsle" mkdir "bin/linux-mipsle"

reset:
	if exist "gateserver_dev.exe" del "gateserver_dev.exe"
	if exist "gateserver_dev" del "gateserver_dev"
	if exist "gateserver_debug.exe" del "gateserver_debug.exe"
	if exist "gateserver_debug" del "gateserver_debug"
	if exist "gateserver.exe" del "gateserver.exe"
	if exist "gateserver_embedded.exe" del "gateserver_embedded.exe"
	if exist "gateserver" del "gateserver"
	if exist "gateserver_embedded" del "gateserver_embedded"
	set GOOS=windows
	set GOARCH=amd64
	go clean -cache

windev:
	set GOOS=windows
	set GOARCH=amd64
	go build -tags embedded,dev -o gateserver_windows_dev.exe

windevnc:
	set GOOS=windows
	set GOARCH=amd64
	go build -tags embedded,dev,newcache -o gateserver_windows_dev_nv.exe

pidev:
	set GOOS=linux
	set GOARCH=arm
	go build -tags embedded,dev -o gateserver_linux_arm_dev

arm64dev:
	set GOOS=linux
	set GOARCH=arm64
	go build -tags embedded,dev -o gateserver_linux_arm64_dev

arm64debug:
	set GOOS=linux
	set GOARCH=arm64
	go build -tags embedded,debug -o gateserver_linux_arm64_debug

windebug:
	set GOOS=windows
	set GOARCH=amd64
	go build -tags embedded,debug -o gateserver_windows_debug.exe

pidebug:
	set GOOS=linux
	set GOARCH=arm
	go build -tags embedded,debug -o gateserver_linux_arm_debug

linuxamddebug:
	set GOOS=linux
	set GOARCH=amd64
	go build -tags embedded,debug -o gateserver_linux_amd64_debug

mipsledebug:
	set GOOS=linux
	set GOARCH=mipsle
	set GOMIPS=softfloat
	go build -tags embedded,debug,newcache -o gateserver_linux_mipsle_debug
	set GOMIPS=

mipsledev:
	set GOOS=linux
	set GOARCH=mipsle
	set GOMIPS=softfloat
	go build -tags embedded,dev,newcache -o gateserver_mipsle_dev
	set GOMIPS=

windows:
	if exist "gateserver.exe" del "gateserver.exe"
	if exist "gateserver_embedded.exe" del "gateserver_embedded.exe"
	set GOOS=windows
	set GOARCH=amd64
	go build -a -gcflags=all="-l -B -wb=false" -ldflags="-w -s"
	go build -a -gcflags=all="-l -B -wb=false" -ldflags="-w -s" -tags embedded -o gateserver_embedded.exe
	if exist "./bin/windows" move gateserver.exe "./bin/windows/gateserver_nightly.exe"
	if exist "./bin/windows" move gateserver_embedded.exe "./bin/windows/gateserver_embedded_nightly.exe"

arm32:
	if exist "gateserver" del "gateserver"
	if exist "gateserver_embedded" del "gateserver_embedded"
	set GOOS=linux
	set GOARCH=arm
	go build -a -gcflags=all="-l -B -wb=false" -ldflags="-w -s"
	go build -a -gcflags=all="-l -B -wb=false" -ldflags="-w -s" -tags embedded -o gateserver_embedded
	if exist "./bin/linux-arm32" move gateserver "./bin/linux-arm32/gateserver_nightly"
	if exist "./bin/linux-arm32" move gateserver_embedded "./bin/linux-arm32/gateserver_embedded_nightly"

arm64:
	if exist "gateserver" del "gateserver"
	if exist "gateserver_embedded" del "gateserver_embedded"
	set GOOS=linux
	set GOARCH=arm64
	go build -a -gcflags=all="-l -B -wb=false" -ldflags="-w -s"
	go build -a -gcflags=all="-l -B -wb=false" -ldflags="-w -s" -tags embedded -o gateserver_embedded
	if exist "./bin/linux-arm64" move gateserver "./bin/linux-arm64/gateserver_nightly"
	if exist "./bin/linux-arm64" move gateserver_embedded "./bin/linux-arm64/gateserver_embedded_nightly"

amd64:
	if exist "gateserver" del "gateserver"
	if exist "gateserver_embedded" del "gateserver_embedded"
	set GOOS=linux
	set GOARCH=amd64
	go build -a -gcflags=all="-l -B -wb=false" -ldflags="-w -s"
	go build -a -gcflags=all="-l -B -wb=false" -ldflags="-w -s" -tags embedded -o gateserver_embedded
	if exist "./bin/linux-amd64" move gateserver "./bin/linux-amd64/gateserver_nightly"
	if exist "./bin/linux-amd64" move gateserver_embedded "./bin/linux-amd64/gateserver_embedded_nightly"

mipsle:
	if exist "gateserver" del "gateserver"
	if exist "gateserver_embedded" del "gateserver_embedded"
	set GOOS=linux
	set GOARCH=mipsle
	set GOMIPS=softfloat
	go build -a
	go build -a -tags embedded,newcache -o gateserver_embedded
	if exist "./bin/linux-mipsle" move gateserver "./bin/linux-mipsle/gateserver_nightly"
	if exist "./bin/linux-mipsle" move gateserver_embedded "./bin/linux-mipsle/gateserver_embedded_nightly"
	set GOMIPS=

all: preparefolders arm32 arm64 amd64 mipsle windows

debug: arm64debug windebug pidebug linuxamddebug