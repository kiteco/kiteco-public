## Generation of ProtoBuf interface files (.pb.go)

The script `proto.sh` regenerates all the .pb.go files automatically. 

It needs `protoc` and `protoc-gen-go` to be availble in your path. 

### Install on Ubuntu
- Download from https://github.com/protocolbuffers/protobuf/releases the release you want. We are currently using 3.11.4
- Unzip its content either in `~/.local` or anywhere accessible in your path (the bin folder needs to be in Path variable)
- Check that your version of `protoc-gen-go` is also up to date. For that:
    - If you have already `protoc-gen-go` in your path, you might need to update it, if not go directly to the 3rd point 
    - Delete `$GOROOT/src/github.com/golang/protobuf/protoc-gen-go` folder and the executable protoc-gen-go from your go bin folder
    - Do `go get github.com/golang/protobuf/protoc-gen-go`
    - Doc taken from https://pkg.go.dev/github.com/golang/protobuf/protoc-gen-go?tab=doc
    
### Generating protobuf files
Once you have `protoc` and `protoc-gen-go` installed, you can run the script `proto.sh` from this folder to generate protobuf file. 

You should get warning like : 
```
2020/05/18 14:57:24 WARNING: Deprecated use of the 'import_path' command-line argument. In "tensorflow_serving/util/class_registration_test.proto", please specify:
	option go_package = "serving";
A future release of protoc-gen-go will no longer support the 'import_path' argument.
See https://developers.google.com/protocol-buffers/docs/reference/go-generated#package for more information.
```   

That's expected and currently a bit of a mess to fix. 


#### ProtoPackageIsVersion4 required
Also, make sure the generated source file require version 4 (or newer?) 
```
const _ = proto.ProtoPackageIsVersion4
```
And not version 3. If you get the line `const _ = proto.ProtoPackageIsVersion3` (notice the 3 at the end), you need to upgrade your version of `protoc-gen-go`.


### Updating GRPC package
This shouldn't be required anymore, the current vendored version is good enough for us. 

In case grpc needs to be updated again: 
- Clean from godeps.json file all references to grpc folder
- Remove grpc folder from vendor subdirectory
- Go to `$GOPATH/go/src/google.golang.org/grpc`, fetch origin and checkout on the branch you need
- Run `make save-deps` from kiteco (with absolute path, save-deps doesn't work when using a path containing a symbolic link)
