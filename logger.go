package main

// import "github.com/corazawaf/coraza/v3/debuglog"

// type errWriter struct {
// 	debuglog.Logger
// }

// func (w errWriter) Write(p []byte) (n int, err error) {
// 	if len(p) == 0 {
// 		return 0, nil
// 	}
// 	w.Logger.Error().Msg(string(p))
// 	return len(p), nil
// }

// type outWriter struct {
// 	debuglog.Logger
// }

// func (w outWriter) Write(p []byte) (n int, err error) {
// 	if len(p) == 0 {
// 		return 0, nil
// 	}
// 	w.Logger.Debug().Msg(string(p))
// 	return len(p), nil
// }
