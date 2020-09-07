module gitlab.com/elixxir/client

go 1.13

replace google.golang.org/grpc => github.com/grpc/grpc-go v1.27.1

require (
	github.com/golang-collections/collections v0.0.0-20130729185459-604e922904d3
	github.com/golang/protobuf v1.4.2
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.0.0
	github.com/spf13/jwalterweatherman v1.1.0
	github.com/spf13/viper v1.7.1
	gitlab.com/elixxir/comms v0.0.0-20200903181126-c92d7a304999
	gitlab.com/elixxir/crypto v0.0.0-20200907171019-008a9d4aa264
	gitlab.com/elixxir/ekv v0.1.1
	gitlab.com/elixxir/primitives v0.0.0-20200907165319-16ed0124890b
	gitlab.com/xx_network/comms v0.0.0-20200825213037-f58fa7c0a641
	gitlab.com/xx_network/crypto v0.0.0-20200812183430-c77a5281c686
	gitlab.com/xx_network/primitives v0.0.0-20200812183720-516a65a4a9b2
	golang.org/x/crypto v0.0.0-20200820211705-5c72a883971a
	google.golang.org/protobuf v1.25.0
)
