module gitlab.com/elixxir/client

go 1.13

require (
	github.com/golang-collections/collections v0.0.0-20130729185459-604e922904d3
	github.com/golang/protobuf v1.4.3
	github.com/gopherjs/gopherjs v0.0.0-20200217142428-fce0ec30dd00 // indirect
	github.com/magiconair/properties v1.8.4 // indirect
	github.com/mitchellh/mapstructure v1.3.3 // indirect
	github.com/pelletier/go-toml v1.8.1 // indirect
	github.com/pkg/errors v0.9.1
	github.com/smartystreets/assertions v1.0.1 // indirect
	github.com/spf13/afero v1.4.0 // indirect
	github.com/spf13/cast v1.3.1 // indirect
	github.com/spf13/cobra v1.0.0
	github.com/spf13/jwalterweatherman v1.1.0
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/spf13/viper v1.7.1
	gitlab.com/elixxir/bloomfilter v0.0.0-20200930191214-10e9ac31b228
	gitlab.com/elixxir/comms v0.0.4-0.20201217200754-6259dc49e6f1
	gitlab.com/elixxir/crypto v0.0.7-0.20201217200352-0ba771a66932
	gitlab.com/elixxir/ekv v0.1.3
	gitlab.com/elixxir/primitives v0.0.3-0.20201218201429-1968e786f56d
	gitlab.com/xx_network/comms v0.0.4-0.20201217200138-87075d5b4ffd
	gitlab.com/xx_network/crypto v0.0.5-0.20201217195719-cc31e1d1eee3
	gitlab.com/xx_network/primitives v0.0.4-0.20201216174909-808eb0fc97fc
	golang.org/x/crypto v0.0.0-20201016220609-9e8e0b390897
	golang.org/x/net v0.0.0-20201110031124-69a78807bb2b // indirect
	golang.org/x/sys v0.0.0-20201119102817-f84b799fce68 // indirect
	google.golang.org/genproto v0.0.0-20201119123407-9b1e624d6bc4 // indirect
	google.golang.org/grpc v1.33.2 // indirect
	google.golang.org/protobuf v1.25.0
	gopkg.in/ini.v1 v1.61.0 // indirect
)

replace google.golang.org/grpc => github.com/grpc/grpc-go v1.27.1
