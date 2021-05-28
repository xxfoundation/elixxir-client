module gitlab.com/elixxir/client

go 1.13

require (
	github.com/golang-collections/collections v0.0.0-20130729185459-604e922904d3
	github.com/golang/protobuf v1.5.2
	github.com/gopherjs/gopherjs v0.0.0-20200217142428-fce0ec30dd00 // indirect
	github.com/magiconair/properties v1.8.4 // indirect
	github.com/mitchellh/mapstructure v1.4.0 // indirect
	github.com/pelletier/go-toml v1.8.1 // indirect
	github.com/pkg/errors v0.9.1
	github.com/smartystreets/assertions v1.0.1 // indirect
	github.com/spf13/afero v1.5.1 // indirect
	github.com/spf13/cast v1.3.1 // indirect
	github.com/spf13/cobra v1.1.1
	github.com/spf13/jwalterweatherman v1.1.0
	github.com/spf13/viper v1.7.1
	gitlab.com/elixxir/bloomfilter v0.0.0-20200930191214-10e9ac31b228
	gitlab.com/elixxir/comms v0.0.4-0.20210528193933-98d0d18be637
	gitlab.com/elixxir/crypto v0.0.7-0.20210526002540-1fb51df5b4b2
	gitlab.com/elixxir/ekv v0.1.5
	gitlab.com/elixxir/primitives v0.0.3-0.20210526002350-b9c947fec050
	gitlab.com/xx_network/comms v0.0.4-0.20210528192421-5931f64cbdc9
	gitlab.com/xx_network/crypto v0.0.5-0.20210526002149-9c08ccb202be
	gitlab.com/xx_network/primitives v0.0.4-0.20210525232109-3f99a04adcfd
	gitlab.com/xx_network/ring v0.0.3-0.20210527191221-ce3f170aabd5 // indirect
	golang.org/x/crypto v0.0.0-20210322153248-0c34fe9e7dc2
	golang.org/x/net v0.0.0-20210525063256-abc453219eb5
	google.golang.org/genproto v0.0.0-20210105202744-fe13368bc0e1 // indirect
	google.golang.org/protobuf v1.26.0
	gopkg.in/ini.v1 v1.62.0 // indirect
)

replace google.golang.org/grpc => github.com/grpc/grpc-go v1.27.1
