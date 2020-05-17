package messages

//go:generate ../bin/protoc -I ../proto/include/ --proto_path=$GOPATH/src:. --validate_out=lang=go:$GOPATH/src --twirp_out=$GOPATH/src --go_out=$GOPATH/src  ./telegram/telegram.proto
