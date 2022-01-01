//go:generate bash -c "cd ../../../../../plugins/vim-plugin"
//go:generate go-bindata -o bindata.go -pkg vim -ignore=(\.git|test) -prefix ../../../../../plugins/ ../../../../../plugins/vim-plugin/...

package vim
