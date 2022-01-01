package clientlogs

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	version    = "0.20190618.0"
	installID  = "0ac28826-bd5c-4175-88a1-b2e12745e962"
	errStr     = "panic: assignment to entry in nil map"
	logsBefore = []byte(`
[kited] 2019/06/18 15:55:32.550122 app.go:149: listening at port: 46624
[kited] 2019/06/18 15:55:32.550233 app.go:150: machine ID: 8fad0ae918cd880075350bcdfe466c6c
[kited] 2019/06/18 15:55:32.550240 app.go:177: install ID: 0ac28826-bd5c-4175-88a1-b2e12745e962
[kited] 2019/06/18 15:55:32.550246 app.go:151: OS: darwin
[kited] 2019/06/18 15:55:32.550257 app.go:152: platform: darwin  10.14.5
[kited] 2019/06/18 15:55:32.550265 app.go:153: root dir: /Users/frank.xu/.kite
[kited] 2019/06/18 15:55:32.550273 app.go:154: log file: /Users/frank.xu/.kite/logs/client.log
[kited] 2019/06/18 15:55:32.550282 app.go:155: dev mode: false
[kited] 2019/06/18 15:55:32.550290 app.go:156: version: 0.20190618.0
[kited] 2019/06/18 15:55:32.550299 app.go:157: configuration: Release
[kited] 2019/03/29 11:48:56.215234 filemap.go:164: reading kite-resource-manager/builtin-stdlib/2.7/symgraphGraph/2019-02-08T19-02-39.blob
[kited] 2019/03/29 11:48:56.747121 filemap.go:164: reading kite-data/python-call-prob/2019-03-27_08-55-48-PM/serve/params.json
[kited] 2019/03/29 11:48:56.761856 component.go:283: error in componentmanager: Initialize of kitelocal failed with`)
	traceback = []byte(`panic: assignment to entry in nil map
[kited] 2019/03/29 11:48:56.762301 rollbar.go:95: panic: error in componentmanager: Initialize of kitelocal failed with panic: assignment to entry in nil map
	goroutine 17 [running, locked to thread]:
github.com/kiteco/kiteco/kite-golib/rollbar.PanicRecovery(0x100a9ee80, 0xc424dfe040, 0x0, 0x0, 0x0)
    /Users/hrysoula/go/src/github.com/kiteco/kiteco/kite-golib/rollbar/rollbar.go:94 +0x82
github.com/kiteco/kiteco/kite-go/client/component.(*Manager).panicRecovery(0xc42055cf60, 0x100c42b5f, 0xa, 0x1011fd980, 0xc420262300)
    /Users/hrysoula/go/src/github.com/kiteco/kiteco/kite-go/client/component/component.go:287 +0x308
panic(0x100b03680, 0x100d0ff50)
    /usr/local/go/src/runtime/panic.go:491 +0x283
github.com/kiteco/kiteco/kite-go/client/internal/kitelocal/internal/indexing.newManager(0x1012074c0, 0xc4265880c0, 0x10120b2e0, 0xc4201d9130, 0xc420262388)
    /Users/hrysoula/go/src/github.com/kiteco/kiteco/kite-go/client/internal/kitelocal/internal/indexing/manager.go:80 +0x5d
github.com/kiteco/kiteco/kite-go/client/internal/kitelocal/internal/indexing.NewContext(0xc4282de140, 0xc4265880c0, 0x10120b2e0, 0xc4201d9130, 0x0)
    /Users/hrysoula/go/src/github.com/kiteco/kiteco/kite-go/client/internal/kitelocal/internal/indexing/context.go:20 +0x4f
[kited] 2019/03/29 11:48:56.762362 rollbar.go:125: rollbar [error]: panic: error in componentmanager: Initialize of kitelocal failed with panic: assignment to entry in nil map []
	`)

	// --

	errStr2     = "runtime: VirtualAlloc of 8192 bytes failed with errno=1455"
	logsBefore2 = []byte(`
[kited] 2020/06/12 18:23:38.030340 app.go:177: machine ID: c5af31248f6d88e4bf6e98c81c4eb8d4
[kited] 2020/06/12 18:23:38.030351 app.go:178: install ID: b8b5059d-a7c0-462a-b938-1032b2ffe185
[kited] 2020/06/12 18:23:38.030359 app.go:179: OS: darwin
[kited] 2020/06/12 18:23:38.030416 app.go:180: CPU: vendor Intel, brand Intel(R) Core(TM) i7-4870HQ CPU @ 2.50GHz, family 6, model 70, 4 cores, 8 threads, flags: AESNI,AVX,AVX2,BMI1,BMI2,CLMUL,CMOV,CX16,ERMS,F16C,FMA3,HTT,IBPB,LZCNT,MMX,MMXEXT,NX,POPCNT,RDRAND,RDTSCP,SSE,SSE2,SSE3,SSE4.1,SSE4.2,SSSE3,STIBP,VMX
[kited] 2020/06/12 18:23:38.030428 app.go:181: platform: darwin Standalone Workstation
[kited] 2020/06/12 18:23:38.030440 app.go:182: root dir: /Users/hrysoula/.kite
[kited] 2020/06/12 18:23:38.030447 app.go:183: log file: /Users/hrysoula/.kite/logs/client.log
[kited] 2020/06/12 18:23:38.030456 app.go:184: dev mode: true
[kited] 2020/06/12 18:23:38.030464 app.go:185: version: 9999
[kited] 2020/06/12 18:23:38.030471 app.go:186: configuration: Debug
[kited] 2020/06/12 18:23:38.030494 app.go:193: proxy: environment
[kited] 2020/06/12 18:23:38.030581 app.go:211: debug build detected, setting ClientVersion to 1ce8ef6c7e
[kited] 2020/06/12 18:23:38.033249 client.go:166: Successfully loaded 1 licenses
[kited] 2020/06/12 18:23:38.039655 manager.go:170: using tf_threads value of 1
[kited] 2020/06/12 18:23:38.383943 track.go:162: tracker.Event: Index Build Filtered
[kited] 2020/06/12 18:23:38.383988 track.go:164: tracker.Event: disabled
[kited] 2020/06/12 18:23:38.384119 manager.go:385: walking root dir /Users/hrysoula
[kited] 2020/06/12 18:23:38.384842 manager.go:263: watching dir /Users/hrysoula
[kited] 2020/06/12 18:23:38.398695 app.go:306: using startup mode: ManualLaunch
[kited] 2020/06/12 18:23:38.398732 app.go:307: using startup channel:
[kited] 2020/06/12 18:23:38.520716 editorslist.go:19: Detected 2 running editors with 0 new entries, purged 0 entries
[kited] 2020/06/12 18:23:38.669473 autoinstall.go:84: no plugins installed automatically
[kited] 2020/06/12 18:23:38.764002 handlers.go:63: GET /clientapi/ping 200 2 86.013Âµs
[kited] 2020/06/12 18:23:38.765623 handlers.go:63: GET /clientapi/status?filename=%2FUsers%2Fhrysoula%2FDocuments%2Ftest%2Ftest.py 200 121 128.289Âµs
[kited] 2020/06/12 18:23:38.855375 http.go:32: starting processHTTP(https://staging.kite.com/)
[kited] 2020/06/12 18:23:38.855488 state.go:65: track: setting user ids User{id:0 email: installID:b8b5059d-a7c0-462a-b938-1032b2ffe185 machineID:c5af31248f6d88e4bf6e98c81c4eb8d4}
[kited] 2020/06/12 18:23:39.227763 manager.go:272: loaded builtin-stdlib==2.7 in 843.523026ms
[kited] 2020/06/12 18:23:40.046088 manager.go:272: loaded builtin-stdlib==3.7 in 818.08969ms
[kited] 2020/06/12 18:23:40.161105 handlers.go:63: POST /clientapi/metrics/counters 0 0 165.499Âµs
[kited] 2020/06/12 18:23:40.292441 handlers.go:63: POST /clientapi/metrics/counters 0 0 117.751Âµs
[kited] 2020/06/12 18:23:40.593386 handlers.go:63: GET /clientapi/settings/metrics_disabled 200 5 61.013Âµs
[kited] 2020/06/12 18:23:40.594928 handlers.go:63: GET /clientapi/settings/sidebar_on_top 200 4 50.055Âµs
[kited] 2020/06/12 18:23:40.595876 handlers.go:63: GET /clientapi/settings/proxy_mode 200 11 42.868Âµs
[kited] 2020/06/12 18:23:40.595984 handlers.go:63: GET /clientapi/settings/proxy_url 200 0 58.632Âµs
[kited] 2020/06/12 18:23:40.596073 handlers.go:63: GET /clientapi/systeminfo 200 36 82.352Âµs
[kited] 2020/06/12 18:23:40.630498 handlers.go:63: GET /clientapi/metrics/install_id 200 38 104.271Âµs
[kited] 2020/06/12 18:23:40.643719 handlers.go:63: GET /clientapi/metrics/id 200 115 150.138Âµs
[kited] 2020/06/12 18:23:40.649234 handlers.go:63: GET /clientapi/settings/setup_completed 200 4 33.275Âµs
[kited] 2020/06/12 18:23:40.654165 handlers.go:63: GET /clientapi/checkonline 200 15 57.511Âµs
[kited] 2020/06/12 18:23:40.657925 handlers.go:63: GET /clientapi/settings/have_shown_welcome 200 4 32.919Âµs
[kited] 2020/06/12 18:23:40.933886 handlers.go:63: GET /clientapi/kited_online 200 15 33.807Âµs
[kited] 2020/06/12 18:23:40.936546 handlers.go:63: GET /clientapi/settings/theme_default 404 29 46.347Âµs
[kited] 2020/06/12 18:23:40.937687 handlers.go:63: GET /clientapi/online 200 15 33.186Âµs
[kited] 2020/06/12 18:23:40.940017 handlers.go:63: GET /clientapi/settings/theme_default 404 29 25.802Âµs
[kited] 2020/06/04 12:08:28.473013 plugins.go:207: activated GoTick in plugin manager
[kited] 2020/06/04 12:08:40.849801 plugins.go:207: activated GoTick in plugin manager
[kited] 2020/06/04 12:09:06.136756 plugins.go:207: activated GoTick in plugin manager
[kited] 2020/06/04 12:09:18.423481 plugins.go:207: activated GoTick in plugin manager
[kited] 2020/06/04 12:09:27.366832 plugins.go:207: activated GoTick in plugin manager
[kited] 2020/06/04 12:09:43.611365 plugins.go:207: activated GoTick in plugin manager
[kited] 2020/06/04 12:09:53.395313 plugins.go:207: activated GoTick in plugin manager
[kited] 2020/06/04 12:10:17.420373 plugins.go:207: activated GoTick in plugin manager
[kited] 2020/06/04 12:10:21.629861 plugins.go:207: activated GoTick in plugin manager
[kited] 2020/06/04 12:10:40.830494 plugins.go:207: activated GoTick in plugin manager
[kited] 2020/06/04 12:10:52.092491 plugins.go:207: activated GoTick in plugin manager
[kited] 2020/06/04 12:10:54.436658 manager.go:272: loaded builtin-stdlib==2.7 in 6m43.7697015s
[kited] 2020/06/04 12:11:08.058327 plugins.go:207: activated GoTick in plugin manager
[kited] 2020/06/04 12:11:32.357582 plugins.go:207: activated GoTick in plugin manager
`)
	traceback2 = []byte(
		`runtime: VirtualAlloc of 8192 bytes failed with errno=1455
fatal error: out of memory

runtime stack:
runtime.throw(0x18aec58e, 0xd)
	c:/go/src/runtime/panic.go:1114 +0x79
runtime.sysUsed(0xc009ad8000, 0x2000)
	c:/go/src/runtime/mem_windows.go:83 +0x230
runtime.(*mheap).allocSpan(0x197dbba0, 0x1, 0xb00, 0x197f2688, 0x1fe)
	c:/go/src/runtime/mheap.go:1229 +0x3cb
runtime.(*mheap).alloc.func1()
	c:/go/src/runtime/mheap.go:871 +0x6b
runtime.systemstack(0x7ffd746c28c0)
	c:/go/src/runtime/asm_amd64.s:370 +0x6b
runtime.mstart()
	c:/go/src/runtime/proc.go:1041

goroutine 9 [running]:
runtime.systemstack_switch()
	c:/go/src/runtime/asm_amd64.s:330 fp=0xc00856bb70 sp=0xc00856bb68 pc=0x4684c0
runtime.(*mheap).alloc(0x197dbba0, 0x1, 0x48f9010b, 0x49029e28)
	c:/go/src/runtime/mheap.go:865 +0x88 fp=0xc00856bbc0 sp=0xc00856bb70 pc=0x428448
runtime.(*mcentral).grow(0x197ec4b0, 0x0)
	c:/go/src/runtime/mcentral.go:255 +0x80 fp=0xc00856bc00 sp=0xc00856bbc0 pc=0x41a3c0
runtime.(*mcentral).cacheSpan(0x197ec4b0, 0x49029e28)
	c:/go/src/runtime/mcentral.go:106 +0x2c3 fp=0xc00856bc48 sp=0xc00856bc00 pc=0x419ee3
runtime.(*mcache).refill(0x1e7507b0, 0xb)
	c:/go/src/runtime/mcache.go:138 +0x8c fp=0xc00856bc68 sp=0xc00856bc48 pc=0x4199ac
runtime.(*mcache).nextFree(0x1e7507b0, 0x10b, 0x0, 0x49029e28, 0x513401)
	c:/go/src/runtime/malloc.go:868 +0x8e fp=0xc00856bca0 sp=0xc00856bc68 pc=0x40e1de
runtime.mallocgc(0x40, 0x0, 0xc00894e900, 0x3e)
	c:/go/src/runtime/malloc.go:1036 +0x7d2 fp=0xc00856bd40 sp=0xc00856bca0 pc=0x40eb62
runtime.slicebytetostring(0x0, 0xc008ea578a, 0x3e, 0x477a4c, 0x0, 0x0)
	c:/go/src/runtime/string.go:102 +0xa6 fp=0xc00856bd70 sp=0xc00856bd40 pc=0x453066
encoding/gob.decString(0xc000b53d88, 0xc00894e920, 0x1885ea80, 0xc00894f4b0, 0x198)
	c:/go/src/encoding/gob/decode.go:399 +0xc4 fp=0xc00856be70 sp=0xc00856bd70 pc=0x7c2034
encoding/gob.(*Decoder).decodeStruct(0xc003126b00, 0xc00894f460, 0x189600e0, 0xc00894f4a0, 0x199)
	c:/go/src/encoding/gob/decode.go:471 +0xe9 fp=0xc00856bf40 sp=0xc00856be70 pc=0x7c29a9
encoding/gob.(*Decoder).decOpFor.func4(0xc005075640, 0xc00894e8e0, 0x189600e0, 0xc00894f4a0, 0x199)
	c:/go/src/encoding/gob/decode.go:860 +0x5b fp=0xc00856bf78 sp=0xc00856bf40 pc=0x7da35b
encoding/gob.decodeIntoValue(0xc00894e8e0, 0xc00894f480, 0x0, 0x189600e0, 0xc00894f4a0, 0x199, 0xc005075640, 0x188bb3e0, 0xc008938d88, 0x18b)
	c:/go/src/encoding/gob/decode.go:551 +0x6c fp=0xc00856bfb8 sp=0xc00856bf78 pc=0x7c38ac
encoding/gob.(*Decoder).decodeMap(0xc003126b00, 0x18d63000, 0x18950ec0, 0xc00894e8e0, 0x18950ec0, 0xc00d942a90, 0x195, 0x18b37e00, 0xc00894f480, 0x18d0fb60, ...)
	c:/go/src/encoding/gob/decode.go:574 +0x44d fp=0xc00856c0c0 sp=0xc00856bfb8 pc=0x7c3d6d
encoding/gob.(*Decoder).decOpFor.func2(0xc005075540, 0xc00894e8e0, 0x18950ec0, 0xc00d942a90, 0x195)
	c:/go/src/encoding/gob/decode.go:829 +0x9e fp=0xc00856c128 sp=0xc00856c0c0 pc=0x7da23e
encoding/gob.(*Decoder).decodeSingle(0xc003126b00, 0xc00894f440, 0x18950ec0, 0xc00d942a90, 0x195)
	c:/go/src/encoding/gob/decode.go:437 +0x124 fp=0xc00856c1c8 sp=0xc00856c128 pc=0x7c2754
encoding/gob.(*Decoder).decodeValue(0xc003126b00, 0xc000000042, 0x188fb660, 0xc00d942a90, 0x16)
	c:/go/src/encoding/gob/decode.go:1207 +0x206 fp=0xc00856c2b0 sp=0xc00856c1c8 pc=0x7ca2f6
encoding/gob.(*Decoder).DecodeValue(0xc003126b00, 0x188fb660, 0xc00d942a90, 0x16, 0x0, 0x0)
	c:/go/src/encoding/gob/decoder.go:213 +0x14f fp=0xc00856c308 sp=0xc00856c2b0 pc=0x7cb75f
encoding/gob.(*Decoder).Decode(0xc003126b00, 0x188fb660, 0xc00d942a90, 0x0, 0x0)
	c:/go/src/encoding/gob/decoder.go:188 +0x174 fp=0xc00856c370 sp=0xc00856c308 pc=0x7cb574
github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/resources/docs.Entities.Decode(0xc00893b260, 0x44de7260, 0xc00894e640, 0x0, 0x0)

goroutine 1 [runnable]:
github.com/kiteco/kiteco/kite-go/client/internal/client.(*Client).processHTTP(0xc0001e4300, 0x18d3c9e0, 0xc0012ac3c0, 0x18afc692, 0x17, 0x0, 0x0)
	D:/containers/containers/0000334m5v4/tmp/build/140ee389/gopath/src/github.com/kiteco/kiteco/kite-go/client/internal/client/http.go:131 +0x797
github.com/kiteco/kiteco/kite-go/client/internal/client.(*Client).Connect(0xc0001e4300, 0x18afc692, 0x17, 0x0, 0x0)
	D:/containers/containers/0000334m5v4/tmp/build/140ee389/gopath/src/github.com/kiteco/kiteco/kite-go/client/internal/client/client.go:286 +0x1a2
main.main()
	D:/containers/containers/0000334m5v4/tmp/build/140ee389/gopath/src/github.com/kiteco/kiteco/kite-go/client/cmds/kited/main.go:55 +0x191

goroutine 20 [chan receive, 945 minutes]:
github.com/kiteco/kiteco/vendor/github.com/rollbar/rollbar-go.NewAsyncTransport.func1(0xc00010b4a0)
	D:/containers/containers/0000334m5v4/tmp/build/140ee389/gopath/src/github.com/kiteco/kiteco/vendor/github.com/rollbar/rollbar-go/async_transport.go:51 +0x69
created by github.com/kiteco/kiteco/vendor/github.com/rollbar/rollbar-go.NewAsyncTransport
	D:/containers/containers/0000334m5v4/tmp/build/140ee389/gopath/src/github.com/kiteco/kiteco/vendor/github.com/rollbar/rollbar-go/async_transport.go:50 +0xe7

goroutine 5 [syscall, 945 minutes]:
os/signal.signal_recv(0x0)
	c:/go/src/runtime/sigqueue.go:147 +0xa3
os/signal.loop()
	c:/go/src/os/signal/signal_unix.go:23 +0x29
created by os/signal.Notify.func1
	c:/go/src/os/signal/signal.go:127 +0x4b

goroutine 22 [select, 41 minutes]:
github.com/kiteco/kiteco/kite-golib/telemetry.(*httpClientAPI).eventLoop(0xc0002e2000)
	D:/containers/containers/0000334m5v4/tmp/build/140ee389/gopath/src/github.com/kiteco/kiteco/kite-golib/telemetry/http.go:38 +0x106
created by github.com/kiteco/kiteco/kite-golib/telemetry.(*httpClientAPI).init
	D:/containers/containers/0000334m5v4/tmp/build/140ee389/gopath/src/github.com/kiteco/kiteco/kite-golib/telemetry/http.go:31 +0xba

goroutine 35 [runnable]:
github.com/kiteco/kiteco/kite-go/client/visibility.(*checker).loop(0xc000068cf0)
	D:/containers/containers/0000334m5v4/tmp/build/140ee389/gopath/src/github.com/kiteco/kiteco/kite-go/client/visibility/checker.go:80 +0x7f
created by github.com/kiteco/kiteco/kite-go/client/visibility.newChecker
	D:/containers/containers/0000334m5v4/tmp/build/140ee389/gopath/src/github.com/kiteco/kiteco/kite-go/client/visibility/checker.go:59 +0xb2

goroutine 36 [select]:
github.com/kiteco/kiteco/kite-go/client/internal/performance.loopLoadAvg(0xc0002ca420)
	D:/containers/containers/0000334m5v4/tmp/build/140ee389/gopath/src/github.com/kiteco/kiteco/kite-go/client/internal/performance/loadavg_windows.go:52 +0xf3
created by github.com/kiteco/kiteco/kite-go/client/internal/performance.init.0
	D:/containers/containers/0000334m5v4/tmp/build/140ee389/gopath/src/github.com/kiteco/kiteco/kite-go/client/internal/performance/loadavg_windows.go:39 +0x1d3

goroutine 37 [chan receive]:
github.com/kiteco/kiteco/kite-go/client/sysidle.(*checker).loop(0xc000bdab00)
	D:/containers/containers/0000334m5v4/tmp/build/140ee389/gopath/src/github.com/kiteco/kiteco/kite-go/client/sysidle/checker.go:51 +0x7e
created by github.com/kiteco/kiteco/kite-go/client/sysidle.newChecker
	D:/containers/containers/0000334m5v4/tmp/build/140ee389/gopath/src/github.com/kiteco/kiteco/kite-go/client/sysidle/checker.go:30 +0xa7

goroutine 6 [select, 945 minutes]:
github.com/kiteco/kiteco/kite-go/client/internal/clientapp.StartPort.func2(0x18d3ca20, 0xc0000940c8, 0xc0000644e0)
	D:/containers/containers/0000334m5v4/tmp/build/140ee389/gopath/src/github.com/kiteco/kiteco/kite-go/client/internal/clientapp/app.go:199 +0x11c
created by github.com/kiteco/kiteco/kite-go/client/internal/clientapp.StartPort
	D:/containers/containers/0000334m5v4/tmp/build/140ee389/gopath/src/github.com/kiteco/kiteco/kite-go/client/internal/clientapp/app.go:197 +0xfbf

goroutine 38 [runnable]:
github.com/kiteco/kiteco/kite-go/client/internal/metrics/livemetrics.(*cpuMetrics).loop(0xc0001e4240, 0x18d3c9e0, 0xc0000d2780)
	D:/containers/containers/0000334m5v4/tmp/build/140ee389/gopath/src/github.com/kiteco/kiteco/kite-go/client/internal/metrics/livemetrics/cpumetrics.go:147 +0x197
created by github.com/kiteco/kiteco/kite-go/client/internal/metrics/livemetrics.newCPUMetrics
	D:/containers/containers/0000334m5v4/tmp/build/140ee389/gopath/src/github.com/kiteco/kiteco/kite-go/client/internal/metrics/livemetrics/cpumetrics.go:38 +0xc2

goroutine 41090 [syscall, locked to thread]:
syscall.Syscall6(0x7ffd726b19c0, 0x5, 0x9f4, 0xc000ee9600, 0x200, 0xc0144ddd3c, 0x0, 0x0, 0x0, 0x0, ...)
	c:/go/src/runtime/syscall_windows.go:201 +0xf2
syscall.ReadFile(0x9f4, 0xc000ee9600, 0x200, 0x200, 0xc0144ddd3c, 0x0, 0x7ffff800000, 0x2)
	c:/go/src/syscall/zsyscall_windows.go:313 +0xd2
syscall.Read(0x9f4, 0xc000ee9600, 0x200, 0x200, 0xc0144ddda0, 0x44eea3, 0x50e592)
	c:/go/src/syscall/syscall_windows.go:344 +0x6f
internal/poll.(*FD).Read(0xc008909680, 0xc000ee9600, 0x200, 0x200, 0x0, 0x0, 0x0)
	c:/go/src/internal/poll/fd_windows.go:513 +0x221
os.(*File).read(...)
	c:/go/src/os/file_windows.go:220
os.(*File).Read(0xc00d942c60, 0xc000ee9600, 0x200, 0x200, 0x18d2a0e0, 0x197eebb0, 0xc0144ddea0)
`)

	//--

	errStr3     = "Exception 0xc0000005 0x0 0xc00e46a080 0x7ff9539e4720"
	logsBefore3 = []byte(`
[kited] 2020/06/12 18:23:38.030340 app.go:177: machine ID: c5af31248f6d88e4bf6e98c81c4eb8d4
[kited] 2020/06/12 18:23:38.030351 app.go:178: install ID: b8b5059d-a7c0-462a-b938-1032b2ffe185
[kited] 2020/06/12 18:23:38.030359 app.go:179: OS: darwin
[kited] 2020/06/12 18:23:38.030416 app.go:180: CPU: vendor Intel, brand Intel(R) Core(TM) i7-4870HQ CPU @ 2.50GHz, family 6, model 70, 4 cores, 8 threads, flags: AESNI,AVX,AVX2,BMI1,BMI2,CLMUL,CMOV,CX16,ERMS,F16C,FMA3,HTT,IBPB,LZCNT,MMX,MMXEXT,NX,POPCNT,RDRAND,RDTSCP,SSE,SSE2,SSE3,SSE4.1,SSE4.2,SSSE3,STIBP,VMX
[kited] 2020/06/12 18:23:38.030428 app.go:181: platform: darwin Standalone Workstation
[kited] 2020/06/12 18:23:38.030440 app.go:182: root dir: /Users/hrysoula/.kite
[kited] 2020/06/12 18:23:38.030447 app.go:183: log file: /Users/hrysoula/.kite/logs/client.log
[kited] 2020/06/12 18:23:38.030456 app.go:184: dev mode: true
[kited] 2020/06/12 18:23:38.030464 app.go:185: version: 9999
[kited] 2020/06/12 18:23:38.030471 app.go:186: configuration: DEBUG
[kited] 2020/06/12 18:23:38.030494 app.go:193: proxy: environment
`)
	traceback3 = []byte(
		`Exception 0xc0000005 0x0 0xc00e46a080 0x7ff9539e4720
PC=0x7ff9539e4720
signal arrived during external code execution

syscall.Syscall(0x7ff953575c70, 0x2, 0xc00e46a080, 0xc00d54c070, 0x0, 0x0, 0x0, 0x0)
	c:/go/src/runtime/syscall_windows.go:188 +0xe9
syscall.(*Proc).Call(0xc0009053c0, 0xc00d54c080, 0x2, 0x2, 0x0, 0x1, 0x8, 0x7)
	c:/go/src/syscall/dll_windows.go:173 +0x1f0
github.com/kiteco/kiteco/vendor/github.com/go-ole/go-ole.CLSIDFromProgID(0x18891fdd, 0xd, 0xea1ae3, 0x1878b9c0, 0xc011a40030)
	D:/containers/containers/0000334m5u7/tmp/build/140ee389/gopath/src/github.com/kiteco/kiteco/vendor/github.com/go-ole/go-ole/com.go:121 +0xbb
github.com/kiteco/kiteco/vendor/github.com/go-ole/go-ole.ClassIDFrom(0x18891fdd, 0xd, 0x18a846c0, 0x19393780, 0x1)
	D:/containers/containers/0000334m5u7/tmp/build/140ee389/gopath/src/github.com/kiteco/kiteco/vendor/github.com/go-ole/go-ole/utility.go:14 +0x40
github.com/kiteco/kiteco/vendor/github.com/go-ole/go-ole/oleutil.CreateObject(0x18891fdd, 0xd, 0xc011a40030, 0xc006693ae0, 0x0)
	D:/containers/containers/0000334m5u7/tmp/build/140ee389/gopath/src/github.com/kiteco/kiteco/vendor/github.com/go-ole/go-ole/oleutil/oleutil.go:16 +0x40
github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/process.readTarget(0xc012a9b700, 0x3e, 0x0, 0x0, 0x0, 0x0)
	D:/containers/containers/0000334m5u7/tmp/build/140ee389/gopath/src/github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/process/manager_windows.go:196 +0x72
github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/process.(*processManager).FilterStartMenuTargets(0x19391ac8, 0x188d87a8, 0xc00098ea00, 0x56, 0x200)
	D:/containers/containers/0000334m5u7/tmp/build/140ee389/gopath/src/github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/process/manager_windows.go:153 +0xa2
`)

	noCrash = []byte(
		`[kited] 2020/06/12 18:23:38.030281 app.go:176: listening at port: 46624
[kited] 2020/06/12 18:23:38.030340 app.go:177: machine ID: c5af31248f6d88e4bf6e98c81c4eb8d4
[kited] 2020/06/12 18:23:38.030351 app.go:178: install ID: b8b5059d-a7c0-462a-b938-1032b2ffe185
[kited] 2020/06/12 18:23:38.030359 app.go:179: OS: darwin
[kited] 2020/06/12 18:23:38.030416 app.go:180: CPU: vendor Intel, brand Intel(R) Core(TM) i7-4870HQ CPU @ 2.50GHz, family 6, model 70, 4 cores, 8 threads, flags: AESNI,AVX,AVX2,BMI1,BMI2,CLMUL,CMOV,CX16,ERMS,F16C,FMA3,HTT,IBPB,LZCNT,MMX,MMXEXT,NX,POPCNT,RDRAND,RDTSCP,SSE,SSE2,SSE3,SSE4.1,SSE4.2,SSSE3,STIBP,VMX
[kited] 2020/06/12 18:23:38.030428 app.go:181: platform: darwin Standalone Workstation
[kited] 2020/06/12 18:23:38.030440 app.go:182: root dir: /Users/hrysoula/.kite
[kited] 2020/06/12 18:23:38.030447 app.go:183: log file: /Users/hrysoula/.kite/logs/client.log
[kited] 2020/06/12 18:23:38.030456 app.go:184: dev mode: true
[kited] 2020/06/12 18:23:38.030464 app.go:185: version: 9999
[kited] 2020/06/12 18:23:38.030471 app.go:186: configuration: Debug
[kited] 2020/06/12 18:23:38.030494 app.go:193: proxy: environment
[kited] 2020/06/12 18:23:38.030581 app.go:211: debug build detected, setting ClientVersion to 1ce8ef6c7e
[kited] 2020/06/12 18:23:38.033249 client.go:166: Successfully loaded 1 licenses
[kited] 2020/06/12 18:23:38.039655 manager.go:170: using tf_threads value of 1
[kited] 2020/06/12 18:23:38.383943 track.go:162: tracker.Event: Index Build Filtered
[kited] 2020/06/12 18:23:38.383988 track.go:164: tracker.Event: disabled
[kited] 2020/06/12 18:23:38.384119 manager.go:385: walking root dir /Users/hrysoula
[kited] 2020/06/12 18:23:38.384842 manager.go:263: watching dir /Users/hrysoula
[kited] 2020/06/12 18:23:38.398695 app.go:306: using startup mode: ManualLaunch
[kited] 2020/06/12 18:23:38.398732 app.go:307: using startup channel:
[kited] 2020/06/12 18:23:38.520716 editorslist.go:19: Detected 2 running editors with 0 new entries, purged 0 entries
[kited] 2020/06/12 18:23:38.669473 autoinstall.go:84: no plugins installed automatically
[kited] 2020/06/12 18:23:38.764002 handlers.go:63: GET /clientapi/ping 200 2 86.013Âµs
[kited] 2020/06/12 18:23:38.765623 handlers.go:63: GET /clientapi/status?filename=%2FUsers%2Fhrysoula%2FDocuments%2Ftest%2Ftest.py 200 121 128.289Âµs
[kited] 2020/06/12 18:23:38.855375 http.go:32: starting processHTTP(https://staging.kite.com/)
[kited] 2020/06/12 18:23:38.855488 state.go:65: track: setting user ids User{id:0 email: installID:b8b5059d-a7c0-462a-b938-1032b2ffe185 machineID:c5af31248f6d88e4bf6e98c81c4eb8d4}
[kited] 2020/06/12 18:23:39.227763 manager.go:272: loaded builtin-stdlib==2.7 in 843.523026ms
[kited] 2020/06/12 18:23:40.046088 manager.go:272: loaded builtin-stdlib==3.7 in 818.08969ms
[kited] 2020/06/12 18:23:40.161105 handlers.go:63: POST /clientapi/metrics/counters 0 0 165.499Âµs
[kited] 2020/06/12 18:23:40.292441 handlers.go:63: POST /clientapi/metrics/counters 0 0 117.751Âµs
[kited] 2020/06/12 18:23:40.593386 handlers.go:63: GET /clientapi/settings/metrics_disabled 200 5 61.013Âµs
[kited] 2020/06/12 18:23:40.594928 handlers.go:63: GET /clientapi/settings/sidebar_on_top 200 4 50.055Âµs
[kited] 2020/06/12 18:23:40.595876 handlers.go:63: GET /clientapi/settings/proxy_mode 200 11 42.868Âµs
[kited] 2020/06/12 18:23:40.595984 handlers.go:63: GET /clientapi/settings/proxy_url 200 0 58.632Âµs
[kited] 2020/06/12 18:23:40.596073 handlers.go:63: GET /clientapi/systeminfo 200 36 82.352Âµs
[kited] 2020/06/12 18:23:40.630498 handlers.go:63: GET /clientapi/metrics/install_id 200 38 104.271Âµs
[kited] 2020/06/12 18:23:40.643719 handlers.go:63: GET /clientapi/metrics/id 200 115 150.138Âµs
[kited] 2020/06/12 18:23:40.649234 handlers.go:63: GET /clientapi/settings/setup_completed 200 4 33.275Âµs
[kited] 2020/06/12 18:23:40.654165 handlers.go:63: GET /clientapi/checkonline 200 15 57.511Âµs
[kited] 2020/06/12 18:23:40.657925 handlers.go:63: GET /clientapi/settings/have_shown_welcome 200 4 32.919Âµs
[kited] 2020/06/12 18:23:40.933886 handlers.go:63: GET /clientapi/kited_online 200 15 33.807Âµs
[kited] 2020/06/12 18:23:40.936546 handlers.go:63: GET /clientapi/settings/theme_default 404 29 46.347Âµs
[kited] 2020/06/12 18:23:40.937687 handlers.go:63: GET /clientapi/online 200 15 33.186Âµs
[kited] 2020/06/12 18:23:40.940017 handlers.go:63: GET /clientapi/settings/theme_default 404 29 25.802Âµs
[kited] 2020/06/12 18:23:40.940487 handlers.go:63: GET /clientapi/online 200 15 18.65Âµs
[kited] 2020/06/12 18:23:40.944371 handlers.go:63: GET /clientapi/settings/theme_default 404 29 36.627Âµs
[kited] 2020/06/12 18:23:40.945107 handlers.go:63: GET /clientapi/online 200 15 29.706Âµs
[kited] 2020/06/12 18:23:40.947069 handlers.go:63: GET /clientapi/settings/theme_default 404 29 34.935Âµs
[kited] 2020/06/12 18:23:40.948108 handlers.go:63: GET /clientapi/online 200 15 24.925Âµs
[kited] 2020/06/12 18:23:41.161550 handlers.go:63: GET /clientapi/user 401 31 222.036078ms
[kited] 2020/06/12 18:23:41.161596 handlers.go:63: GET /clientapi/user 401 31 211.085963ms
[kited] 2020/06/12 18:23:41.161556 handlers.go:63: GET /clientapi/user 401 31 224.737613ms
[kited] 2020/06/12 18:23:41.161572 handlers.go:63: GET /clientapi/user 401 31 214.597605ms
[kited] 2020/06/12 18:23:41.161577 handlers.go:63: GET /clientapi/user 401 31 217.572036ms
[kited] 2020/06/12 18:23:41.161585 handlers.go:63: GET /clientapi/license-info 200 80 568.365334ms
[kited] 2020/06/12 18:23:41.163589 http.go:142: received login event
[kited] 2020/06/12 18:23:41.163637 plugins.go:207: activated GoTick in plugin manager
[kited] 2020/06/12 18:23:41.163947 listener.go:36: connection dropped: failed to get reader: context canceled
[kited] 2020/06/12 18:23:41.167894 handlers.go:63: GET /clientapi/settings/theme_default 404 29 2.755ms
[kited] 2020/06/12 18:23:41.168185 handlers.go:63: GET /clientapi/online 200 15 150.766Âµs
[kited] 2020/06/12 18:23:41.171002 handlers.go:63: GET /clientapi/settings/theme_default 404 29 60.888Âµs
[kited] 2020/06/12 18:23:41.172999 handlers.go:63: GET /clientapi/online 200 15 31.767Âµs
[kited] 2020/06/12 18:23:41.184639 handlers.go:63: GET /clientapi/settings/theme_default 404 29 179.49Âµs
[kited] 2020/06/12 18:23:41.184718 handlers.go:63: GET /clientapi/online 200 15 154.001Âµs
[kited] 2020/06/12 18:23:41.186851 handlers.go:63: GET /clientapi/settings/theme_default 404 29 67.503Âµs
[kited] 2020/06/12 18:23:41.187040 handlers.go:63: GET /clientapi/online 200 15 72.93Âµs
[kited] 2020/06/12 18:23:41.204656 handlers.go:63: GET /clientapi/online 200 15 46.914Âµs
[kited] 2020/06/12 18:23:41.204913 handlers.go:63: GET /clientapi/settings/theme_default 404 29 480.711Âµs
[kited] 2020/06/12 18:23:41.206019 handlers.go:63: GET /clientapi/settings/theme_default 404 29 35.473Âµs
[kited] 2020/06/12 18:23:41.206708 handlers.go:63: GET /clientapi/online 200 15 25.649Âµs
[kited] 2020/06/12 18:23:41.217861 handlers.go:63: GET /clientapi/settings/theme_default 404 29 6.154611ms
[kited] 2020/06/12 18:23:41.240515 plugins.go:213: updating Kite_status flag for spyder settings: suboptimal = false
[kited] 2020/06/12 18:23:43.759516 handlers.go:63: GET /clientapi/ping 200 2 36.516Âµs
[kited] 2020/06/12 18:23:43.761247 handlers.go:63: GET /clientapi/status?filename=%2FUsers%2Fhrysoula%2FDocuments%2Ftest%2Ftest.py 200 121 99.229Âµs
[kited] 2020/06/12 18:23:44.775580 manager.go:354: identifying with id: 42
[kited] 2020/06/12 18:23:44.775630 handlers.go:63: GET /clientapi/user 200 177 3.557120965s
[kited] 2020/06/12 18:23:44.775656 handlers.go:63: GET /clientapi/user 200 177 3.603936644s
[kited] 2020/06/12 18:23:44.775630 handlers.go:63: GET /clientapi/user 200 177 3.588875652s
[kited] 2020/06/12 18:23:44.775707 handlers.go:63: GET /clientapi/user 200 177 3.591264902s
[kited] 2020/06/12 18:23:44.775630 handlers.go:63: GET /clientapi/user 200 177 3.569688792s
[kited] 2020/06/12 18:23:44.775637 handlers.go:63: GET /clientapi/user 200 177 3.582634792s
[kited] 2020/06/12 18:23:44.782481 handlers.go:63: GET /clientapi/settings/theme_default 404 29 235.836Âµs
[kited] 2020/06/12 18:23:44.782623 handlers.go:63: GET /clientapi/settings/theme_default 404 29 128.257Âµs
[kited] 2020/06/12 18:23:44.782546 handlers.go:63: GET /clientapi/online 200 15 249.453Âµs
[kited] 2020/06/12 18:23:44.782676 handlers.go:63: GET /clientapi/online 200 15 70.692Âµs
[kited] 2020/06/12 18:23:44.799296 handlers.go:63: GET /clientapi/online 200 15 13.047289ms
[kited] 2020/06/12 18:23:44.799580 handlers.go:63: GET /clientapi/settings/metrics_disabled 200 5 176.119Âµs
[kited] 2020/06/12 18:23:44.800957 handlers.go:63: GET /clientapi/settings/autosearch_default 200 4 47.563Âµs
[kited] 2020/06/12 18:23:44.801810 handlers.go:63: DELETE /clientapi/plugins/auto_installed 404 5 820.021Âµs
[kited] 2020/06/12 18:23:44.802319 handlers.go:63: GET /clientapi/online 200 15 54.653Âµs
[kited] 2020/06/12 18:23:44.803270 handlers.go:63: GET /clientapi/metrics/id 200 47 136.309Âµs
[kited] 2020/06/12 18:23:45.840839 autoinstall.go:84: no plugins installed automatically
[kited] 2020/06/12 18:23:47.455180 handlers.go:63: GET /clientapi/user 200 177 2.672385857s
[kited] 2020/06/12 18:23:47.455237 handlers.go:63: GET /clientapi/user 200 177 2.672618496s
[kited] 2020/06/12 18:23:47.455248 handlers.go:63: GET /clientapi/user 200 177 2.669180239s
[kited] 2020/06/12 18:23:47.457999 handlers.go:63: GET /clientapi/metrics/id 200 47 144.751Âµs
[kited] 2020/06/12 18:23:47.458417 handlers.go:63: GET /clientapi/metrics/id 200 47 319.74Âµs
[kited] 2020/06/12 18:23:47.460962 handlers.go:63: GET /clientapi/metrics/id 200 47 45.002Âµs
[kited] 2020/06/12 18:23:48.750661 handlers.go:63: GET /clientapi/ping 200 2 35.657Âµs
[kited] 2020/06/12 18:23:48.752124 handlers.go:63: GET /clientapi/status?filename=%2FUsers%2Fhrysoula%2FDocuments%2Ftest%2Ftest.py 200 121 56.894Âµs
[kited] 2020/06/12 18:23:50.721805 handlers.go:63: POST /clientapi/plugins 200 3193 5.936090646s
[kited] 2020/06/12 18:23:50.722501 handlers.go:63: GET /clientapi/metrics/id 200 47 55.331Âµs
[kited] 2020/06/12 18:23:53.746138 handlers.go:63: GET /clientapi/ping 200 2 50.952Âµs
[kited] 2020/06/12 18:23:53.747588 handlers.go:63: GET /clientapi/status?filename=%2FUsers%2Fhrysoula%2FDocuments%2Ftest%2Ftest.py 200 121 74.248Âµs
[kited] 2020/06/12 18:23:54.776949 loghelper.go:93: error uploading logs: error posting logs to https://staging.kite.com/clientlogs?filename=client.log.2020-04-28_11-17-14-PM.bak&installid=b8b5059d-a7c0-462a-b938-1032b2ffe185&machineid=c5af31248f6d88e4bf6e98c81c4eb8d4&platform=darwin: Post "https://staging.kite.com/clientlogs?filename=client.log.2020-04-28_11-17-14-PM.bak&installid=b8b5059d-a7c0-462a-b938-1032b2ffe185&machineid=c5af31248f6d88e4bf6e98c81c4eb8d4&platform=darwin": context deadline exceeded
[kited] 2020/06/12 18:23:58.743528 handlers.go:63: GET /clientapi/ping 200 2 46.928Âµs
[kited] 2020/06/12 18:23:58.745019 handlers.go:63: GET /clientapi/status?filename=%2FUsers%2Fhrysoula%2FDocuments%2Ftest%2Ftest.py 200 121 70.182Âµs
[kited] 2020/06/12 18:23:59.106931 handlers.go:63: GET /clientapi/license-info 200 80 14.304730669s
[kited] 2020/06/12 18:23:59.108492 handlers.go:63: GET /clientapi/metrics/id 200 47 123.02Âµs
[kited] 2020/06/12 18:23:59.109445 handlers.go:63: POST /clientapi/metrics/counters 0 0 60.719Âµs
[kited] 2020/06/12 18:23:59.110426 handlers.go:63: POST /clientapi/settings/autosearch_default 0 0 374.691Âµs
[kited] 2020/06/12 18:23:59.111067 handlers.go:63: GET /clientapi/online 200 15 32.102Âµs
[kited] 2020/06/12 18:24:00.643902 handlers.go:63: GET /clientapi/license-info 200 80 15.839979447s
[kited] 2020/06/12 18:24:00.645428 handlers.go:63: GET /clientapi/metrics/id 200 47 74.664Âµs
[kited] 2020/06/12 18:24:03.749260 handlers.go:63: GET /clientapi/ping 200 2 33.505Âµs
[kited] 2020/06/12 18:24:03.750769 handlers.go:63: GET /clientapi/status?filename=%2FUsers%2Fhrysoula%2FDocuments%2Ftest%2Ftest.py 200 121 69.911Âµs
[kited] 2020/06/12 18:24:08.765532 handlers.go:63: GET /clientapi/ping 200 2 59.539Âµs
[kited] 2020/06/12 18:24:08.766979 handlers.go:63: GET /clientapi/status?filename=%2FUsers%2Fhrysoula%2FDocuments%2Ftest%2Ftest.py 200 121 84.063Âµs
[kited] 2020/06/12 18:24:09.107501 loghelper.go:93: error uploading logs: error posting logs to https://staging.kite.com/clientlogs?filename=client.log.2020-06-01_02-20-43-PM.bak&installid=b8b5059d-a7c0-462a-b938-1032b2ffe185&machineid=c5af31248f6d88e4bf6e98c81c4eb8d4&platform=darwin: Post "https://staging.kite.com/clientlogs?filename=client.log.2020-06-01_02-20-43-PM.bak&installid=b8b5059d-a7c0-462a-b938-1032b2ffe185&machineid=c5af31248f6d88e4bf6e98c81c4eb8d4&platform=darwin": context deadline exceeded
[kited] 2020/06/12 18:24:13.755678 handlers.go:63: GET /clientapi/ping 200 2 65.82Âµs
[kited] 2020/06/12 18:24:13.757093 handlers.go:63: GET /clientapi/status?filename=%2FUsers%2Fhrysoula%2FDocuments%2Ftest%2Ftest.py 200 121 83.77Âµs
[kited] 2020/06/12 18:24:18.755734 handlers.go:63: GET /clientapi/ping 200 2 64.097Âµs
[kited] 2020/06/12 18:24:18.757195 handlers.go:63: GET /clientapi/status?filename=%2FUsers%2Fhrysoula%2FDocuments%2Ftest%2Ftest.py 200 121 72.606Âµs
[kited] 2020/06/12 18:24:20.220969 handlers.go:63: GET /clientapi/license-info 200 80 32.764006633s
[kited] 2020/06/12 18:24:20.222072 handlers.go:63: GET /clientapi/metrics/id 200 47 61.005Âµs
[kited] 2020/06/12 18:24:21.850656 handlers.go:63: GET /clientapi/license-info 200 80 34.390904899s
[kited] 2020/06/12 18:24:21.852383 handlers.go:63: GET /clientapi/metrics/id 200 47 126.323Âµs
[kited] 2020/06/12 18:24:21.853509 handlers.go:63: GET /clientapi/online 200 15 29.718Âµs
[kited] 2020/06/12 18:24:21.854376 handlers.go:63: GET /clientapi/online 200 15 24.201Âµs
[kited] 2020/06/12 18:24:21.855296 handlers.go:63: GET /clientapi/online 200 15 26.23Âµs
[kited] 2020/06/12 18:24:21.856400 handlers.go:63: GET /clientapi/online 200 15 29.524Âµs
[kited] 2020/06/12 18:24:21.857589 handlers.go:63: GET /clientapi/online 200 15 51.701Âµs
[kited] 2020/06/12 18:24:21.858504 handlers.go:63: GET /clientapi/settings/autosearch_default 200 4 55.528Âµs
[kited] 2020/06/12 18:24:21.859216 handlers.go:63: GET /clientapi/online 200 15 24.45Âµs
[kited] 2020/06/12 18:24:21.860205 handlers.go:63: GET /clientapi/online 200 15 36.556Âµs
[kited] 2020/06/12 18:24:21.860981 handlers.go:63: GET /clientapi/online 200 15 43.248Âµs
[kited] 2020/06/12 18:24:21.861781 handlers.go:63: GET /clientapi/online 200 15 21.859Âµs
[kited] 2020/06/12 18:24:21.862700 handlers.go:63: GET /clientapi/online 200 15 57.062Âµs
[kited] 2020/06/12 18:24:21.863423 handlers.go:63: GET /clientapi/online 200 15 16.176Âµs
[kited] 2020/06/12 18:24:21.864166 handlers.go:63: GET /clientapi/online 200 15 42.836Âµs
[kited] 2020/06/12 18:24:21.865169 handlers.go:63: GET /clientapi/online 200 15 16.955Âµs
[kited] 2020/06/12 18:24:21.865768 handlers.go:63: GET /clientapi/online 200 15 21.83Âµs
[kited] 2020/06/12 18:24:21.866500 handlers.go:63: GET /clientapi/online 200 15 16.74Âµs
[kited] 2020/06/12 18:24:21.867219 handlers.go:63: GET /clientapi/online 200 15 17.001Âµs
[kited] 2020/06/12 18:24:23.374698 handlers.go:63: GET /clientapi/online 200 15 42.079Âµs
[kited] 2020/06/12 18:24:23.387322 handlers.go:63: GET /clientapi/online 200 15 31.955Âµs
[kited] 2020/06/12 18:24:23.762791 handlers.go:63: GET /clientapi/ping 200 2 55.246Âµs
[kited] 2020/06/12 18:24:23.766453 handlers.go:63: GET /clientapi/status?filename=%2FUsers%2Fhrysoula%2FDocuments%2Ftest%2Ftest.py 200 121 83.903Âµs
[kited] 2020/06/12 18:24:26.386451 handlers.go:63: GET /clientapi/online 200 15 36.085Âµs
[kited] 2020/06/12 18:24:28.760175 handlers.go:63: GET /clientapi/ping 200 2 32.002Âµs
[kited] 2020/06/12 18:24:28.761510 handlers.go:63: GET /clientapi/status?filename=%2FUsers%2Fhrysoula%2FDocuments%2Ftest%2Ftest.py 200 121 60.213Âµs
[kited] 2020/06/12 18:24:29.388437 handlers.go:63: GET /clientapi/online 200 15 36.43Âµs
[kited] 2020/06/12 18:24:30.220919 loghelper.go:93: error uploading logs: error posting logs to https://staging.kite.com/clientlogs?filename=client.log.2020-06-03_01-14-37-AM.bak&installid=b8b5059d-a7c0-462a-b938-1032b2ffe185&machineid=c5af31248f6d88e4bf6e98c81c4eb8d4&platform=darwin: Post "https://staging.kite.com/clientlogs?filename=client.log.2020-06-03_01-14-37-AM.bak&installid=b8b5059d-a7c0-462a-b938-1032b2ffe185&machineid=c5af31248f6d88e4bf6e98c81c4eb8d4&platform=darwin": context deadline exceeded
[kited] 2020/06/12 18:24:30.374667 handlers.go:63: GET /clientapi/online 200 15 27.845Âµs
[kited] 2020/06/12 18:24:32.390661 handlers.go:63: GET /clientapi/online 200 15 41.878Âµs
[kited] 2020/06/12 18:24:33.766291 handlers.go:63:`)
)

func Test_FindCrash(t *testing.T) {
	logs := append(logsBefore, traceback...)

	foundVersion, err := findVersion(logs)
	assert.Nil(t, err)
	assert.Equal(t, version, foundVersion)

	foundID, err := findInstallID(logs)
	assert.Nil(t, err)
	assert.Equal(t, installID, foundID)

	foundErrStr, foundTraceback := findCrash(logs)
	assert.Equal(t, strings.TrimSpace(string(errStr)), strings.TrimSpace(foundErrStr))
	assert.Equal(t, strings.TrimSpace(string(traceback)), strings.TrimSpace(string(foundTraceback)))
}

func Test_HandleLogUpload(t *testing.T) {
	server, _, db := makeTestServer()
	defer db.Close()

	installid := "0"
	machineid := "1"
	filename := "testfile"

	vals := url.Values{}
	vals.Set("filename", filename)
	vals.Set("machineid", machineid)
	vals.Set("installid", installid)
	uploadURL := makeTestURL(server.URL, "logupload")
	ep, err := uploadURL.Parse("?" + vals.Encode())
	var buf bytes.Buffer
	resp, err := http.Post(ep.String(), "text/plain", &buf)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, resp.StatusCode, http.StatusOK)

	data := struct {
		URL string `json:"url"`
	}{}
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&data)
	require.NoError(t, err)
	key := fmt.Sprintf("%s/%s/%s/%s.gz", "dev", installid, machineid, filename)
	urlStr := fmt.Sprintf("%s/%s/%s", s3URL, clientLogBucket, key)
	require.Equal(t, urlStr, data.URL)
}

func Test_ReadLogTail(t *testing.T) {
	logs := append(logsBefore, traceback...)

	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	_, err := gzw.Write(logs)
	require.NoError(t, err)
	err = gzw.Close()
	require.NoError(t, err)

	var buf2 bytes.Buffer
	tee := io.TeeReader(bytes.NewReader(buf.Bytes()), &buf2)

	_, server, db := makeTestServer()
	defer db.Close()

	cur, foundVersion, foundID, err := server.readLogTail(tee, &buf)
	assert.Nil(t, err)
	assert.Equal(t, version, foundVersion)
	assert.Equal(t, installID, foundID)

	foundErrStr, foundTraceback := findCrash(cur)
	assert.Equal(t, strings.TrimSpace(string(errStr)), strings.TrimSpace(foundErrStr))
	assert.True(t, bytes.Equal(bytes.TrimSpace(foundTraceback), bytes.TrimSpace(traceback)))
}

func Test_ReadLogTail2(t *testing.T) {
	maxLogRead = 10 << 10
	maxTracebackSize = 1 << 10
	defer func() {
		maxTracebackSize = defaultMaxTracebackSize
	}()
	logs := append(logsBefore2, traceback2...)

	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	_, err := gzw.Write(logs)
	require.NoError(t, err)
	err = gzw.Close()
	require.NoError(t, err)

	var buf2 bytes.Buffer
	tee := io.TeeReader(bytes.NewReader(buf.Bytes()), &buf2)

	_, server, db := makeTestServer()
	defer db.Close()

	cur, _, _, err := server.readLogTail(tee, &buf)
	assert.Nil(t, err)
	foundErrStr, foundTraceback := findCrash(cur)
	assert.Equal(t, strings.TrimSpace(string(errStr2)), strings.TrimSpace(foundErrStr))
	assert.True(t, bytes.Equal(bytes.TrimSpace(foundTraceback), bytes.TrimSpace(traceback2[:maxTracebackSize])))
}

func Test_ReadLogTail3(t *testing.T) {
	maxLogRead = 1 << 10
	maxTracebackSize = 1 << 10
	defer func() {
		maxTracebackSize = defaultMaxTracebackSize
	}()
	logs := append(logsBefore3, traceback3...)

	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	_, err := gzw.Write(logs)
	require.NoError(t, err)
	err = gzw.Close()
	require.NoError(t, err)

	var buf2 bytes.Buffer
	tee := io.TeeReader(bytes.NewReader(buf.Bytes()), &buf2)

	_, server, db := makeTestServer()
	defer db.Close()

	cur, _, _, err := server.readLogTail(tee, &buf)
	assert.Nil(t, err)
	foundErrStr, foundTraceback := findCrash(cur)
	assert.Equal(t, strings.TrimSpace(string(errStr3)), strings.TrimSpace(foundErrStr))
	assert.True(t, bytes.Equal(bytes.TrimSpace(foundTraceback), bytes.TrimSpace(traceback3[:maxTracebackSize])))
}

func Test_ReadLogTailEmpty(t *testing.T) {
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	_, err := gzw.Write([]byte{})
	require.NoError(t, err)
	err = gzw.Close()
	require.NoError(t, err)

	var buf2 bytes.Buffer
	tee := io.TeeReader(bytes.NewReader(buf.Bytes()), &buf2)

	_, server, db := makeTestServer()
	defer db.Close()

	cur, version, id, err := server.readLogTail(tee, &buf)
	assert.Nil(t, err)
	assert.Equal(t, "", version)
	assert.Equal(t, "", id)

	foundErrStr, foundTraceback := findCrash(cur)
	assert.Equal(t, "", foundErrStr)
	assert.Len(t, bytes.TrimSpace(foundTraceback), 0)
}

func Test_ReadLogTailNoCrash(t *testing.T) {
	maxLogRead = 1 << 10
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	_, err := gzw.Write(noCrash)
	require.NoError(t, err)
	err = gzw.Close()
	require.NoError(t, err)

	var buf2 bytes.Buffer
	tee := io.TeeReader(bytes.NewReader(buf.Bytes()), &buf2)

	_, server, db := makeTestServer()
	defer db.Close()

	cur, _, _, err := server.readLogTail(tee, &buf)
	assert.Nil(t, err)

	foundErrStr, foundTraceback := findCrash(cur)
	assert.Equal(t, "", foundErrStr)
	assert.Len(t, bytes.TrimSpace(foundTraceback), 0)
}

func Test_CrashTypes(t *testing.T) {
	errTypes := [][]byte{
		[]byte(`panic: assignment to entry in nil map`),
		[]byte(`runtime: VirtualAlloc of 8192 bytes failed with errno=1455`),
		[]byte(`Exception 0xc0000005 0x0 0xc00e46a080 0x7ff9539e4720`),
		[]byte(`fatal error: unexpected signal during runtime execution`),
		[]byte(`goroutine 3 [running]:`),
	}

	for _, e := range errTypes {
		foundErrStr, foundTraceback := findCrash(e)
		assert.Equal(t, string(e), foundErrStr)
		assert.True(t, bytes.Equal(bytes.TrimSpace(foundTraceback), bytes.TrimSpace(e)))
	}
}
