package interfaces

type ClientError struct {
	Source  string
	Message string
	Trace   string
}

type ClientErrorReport func(source, message, trace string)
