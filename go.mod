module gitlab.com/elixxir/client

go 1.13

require (
	github.com/aws/aws-lambda-go v1.8.1 // indirect
	github.com/golang-collections/collections v0.0.0-20130729185459-604e922904d3
	github.com/golang/protobuf v1.4.3
	github.com/gopherjs/gopherjs v0.0.0-20200217142428-fce0ec30dd00 // indirect
	github.com/liyue201/goqr v0.0.0-20200803022322-df443203d4ea
	github.com/magiconair/properties v1.8.4 // indirect
	github.com/mitchellh/mapstructure v1.4.0 // indirect
	github.com/pelletier/go-toml v1.8.1 // indirect
	github.com/pkg/errors v0.9.1
	github.com/skip2/go-qrcode v0.0.0-20200617195104-da1b6568686e
	github.com/smartystreets/assertions v1.0.1 // indirect
	github.com/spf13/afero v1.5.1 // indirect
	github.com/spf13/cast v1.3.1 // indirect
	github.com/spf13/cobra v1.1.1
	github.com/spf13/jwalterweatherman v1.1.0
	github.com/spf13/viper v1.7.1
	gitlab.com/elixxir/bloomfilter v0.0.0-20200930191214-10e9ac31b228
	gitlab.com/elixxir/comms v0.0.4-0.20210409164117-ce218393248a
	gitlab.com/elixxir/crypto v0.0.7-0.20210409164001-5cb03c4d99b4
	gitlab.com/elixxir/ekv v0.1.5
	gitlab.com/elixxir/primitives v0.0.3-0.20210409190923-7bf3cd8d97e7
	gitlab.com/xx_network/comms v0.0.4-0.20210406210737-45d1e87d294a
	gitlab.com/xx_network/crypto v0.0.5-0.20210405224157-2b1f387b42c1
	gitlab.com/xx_network/primitives v0.0.4-0.20210402222416-37c1c4d3fac4
	golang.org/x/crypto v0.0.0-20210322153248-0c34fe9e7dc2
	golang.org/x/sys v0.0.0-20210403161142-5e06dd20ab57 // indirect
	google.golang.org/genproto v0.0.0-20210105202744-fe13368bc0e1 // indirect
	google.golang.org/grpc v1.34.0 // indirect
	google.golang.org/protobuf v1.26.0-rc.1
	gopkg.in/ini.v1 v1.62.0 // indirect
)

replace google.golang.org/grpc => github.com/grpc/grpc-go v1.27.1
