package errors

import "strings"

type quickErr string

func (q quickErr) Error() string {
	return string(q)
}

func Error(msg string) error {
	return quickErr(msg)
}

func ErrorWith(errMsgs ...string) error {
	var errMsgBuilder strings.Builder
	for _, msg := range errMsgs {
		errMsgBuilder.WriteString(msg)
		errMsgBuilder.WriteByte(';')
	}
	return Error(errMsgBuilder.String())
}
