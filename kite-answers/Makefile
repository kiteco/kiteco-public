.PHONY: preview-app deploy-preview assets clean-assets

preview-app:
	cd web/preview && npm run clean && npm run update-renderer && npm install && npm run build

deploy-preview:
	env GOOS=linux GOARCH=amd64 go build github.com/kiteco/kiteco/kite-answers/go/cmds/preview-server
	ssh answers-preview.kite.com "sudo killall preview-server" || true
	scp preview-server answers-preview.kite.com:
	ssh answers-preview.kite.com "sudo ./preview-server </dev/null >preview-server.log 2>&1 &"
	rm preview-server

assets: go/cmds/preview-server/assets_bindata.go

clean-assets:
	rm go/cmds/preview-server/assets_bindata.go


preview-server:
	go build github.com/kiteco/kiteco/kite-answers/go/cmds/preview-server

go/cmds/preview-server/assets_bindata.go: web/preview/build PREVIEW.md
	rm -rf assets
	mkdir assets
	cp -r web/preview/build/* assets/
	cp PREVIEW.md assets/
	
	go-bindata-assetfs assets/... -o bindata_assetfs.go
	mv bindata_assetfs.go go/cmds/preview-server/assets_bindata.go
	
	rm -r assets
