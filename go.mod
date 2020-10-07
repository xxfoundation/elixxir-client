module gitlab.com/elixxir/client

go 1.13

require (
	github.com/golang-collections/collections v0.0.0-20130729185459-604e922904d3
	github.com/golang/protobuf v1.4.2
	github.com/magiconair/properties v1.8.4 // indirect
	github.com/mitchellh/mapstructure v1.3.3 // indirect
	github.com/pelletier/go-toml v1.8.1 // indirect
	github.com/pkg/errors v0.9.1
	github.com/spf13/afero v1.4.0 // indirect
	github.com/spf13/cobra v1.0.0
	github.com/spf13/jwalterweatherman v1.1.0
	github.com/spf13/viper v1.7.1
	gitlab.com/elixxir/comms v0.0.0-20201006010513-3353fb46569a
	gitlab.com/elixxir/crypto v0.0.0-20201006010428-67a8782d097e
	gitlab.com/elixxir/ekv v0.1.3
	gitlab.com/elixxir/primitives v0.0.0-20201007182231-b9aef2de8219
	gitlab.com/xx_network/comms v0.0.0-20200924225518-0c867207b1e6
	gitlab.com/xx_network/crypto v0.0.0-20200812183430-c77a5281c686
	gitlab.com/xx_network/primitives v0.0.0-20200915204206-eb0287ed0031
	golang.org/x/sys v0.0.0-20200923182605-d9f96fdee20d // indirect
	google.golang.org/protobuf v1.25.0
	gopkg.in/ini.v1 v1.61.0 // indirect
)

replace google.golang.org/grpc => github.com/grpc/grpc-go v1.27.1
