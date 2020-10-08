package bindings

type Params interface {
	GetString(key string) (string, error)
	SetString(key, value string)
	GetInt(key string) (int, error)
	SetInt(key string, value int)
	GetFloat(key string) (float64, error)
	SetFloat(key, value string)
	GetParams(key string) (Params, error)
	SetParams(key string, p Params)
}
