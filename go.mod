module gitlab.com/elixxir/client

go 1.13

require (
	github.com/golang-collections/collections v0.0.0-20130729185459-604e922904d3
	github.com/golang/protobuf v1.4.2
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.0.0
	github.com/spf13/jwalterweatherman v1.1.0
	github.com/spf13/viper v1.6.2
	gitlab.com/elixxir/comms v0.0.0-20200925154004-716284cff8cb
	gitlab.com/elixxir/crypto v0.0.0-20200921195205-bca0178268ec
	gitlab.com/elixxir/ekv v0.1.2-0.20200917221437-9f9630da030a
	gitlab.com/elixxir/primitives v0.0.0-20200915190719-f4586ec93f50
	gitlab.com/xx_network/comms v0.0.0-20200925191822-08c0799a24a6
	gitlab.com/xx_network/crypto v0.0.0-20200812183430-c77a5281c686
	gitlab.com/xx_network/primitives v0.0.0-20200812183720-516a65a4a9b2
	golang.org/x/sys v0.0.0-20200918174421-af09f7315aff // indirect
	google.golang.org/protobuf v1.25.0
)

replace google.golang.org/grpc => github.com/grpc/grpc-go v1.27.1
