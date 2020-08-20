module gitlab.com/elixxir/client

go 1.13

require (
	github.com/golang-collections/collections v0.0.0-20130729185459-604e922904d3
	github.com/golang/protobuf v1.4.2
	github.com/gopherjs/gopherjs v0.0.0-20200217142428-fce0ec30dd00 // indirect
	github.com/pelletier/go-toml v1.6.0 // indirect
	github.com/pkg/errors v0.9.1
	github.com/smartystreets/assertions v1.0.1 // indirect
	github.com/spf13/afero v1.2.2 // indirect
	github.com/spf13/cast v1.3.1 // indirect
	github.com/spf13/cobra v1.0.0
	github.com/spf13/jwalterweatherman v1.1.0
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/spf13/viper v1.6.2
	gitlab.com/elixxir/comms v0.0.0-20200813225502-e879259ca741
	gitlab.com/elixxir/crypto v0.0.0-20200820010132-199706584933
	gitlab.com/elixxir/ekv v0.0.0-20200729182028-159355ea5842
	gitlab.com/elixxir/primitives v0.0.0-20200812191102-31c01f08b4dc
	gitlab.com/xx_network/comms v0.0.0-20200818182121-732dd75b1947
	gitlab.com/xx_network/crypto v0.0.0-20200812183430-c77a5281c686
	gitlab.com/xx_network/primitives v0.0.0-20200812183720-516a65a4a9b2
	golang.org/x/crypto v0.0.0-20200728195943-123391ffb6de
	gopkg.in/ini.v1 v1.52.0 // indirect
)

replace google.golang.org/grpc => github.com/grpc/grpc-go v1.27.1
