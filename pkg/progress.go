package degit

import "io"

// Progress is an optional hook for surfacing download progress.
// Implementations receive a copy of downloaded bytes via Write,
// learn the total length via Init (called once before the first Write,
// possibly again on a redirected response), and are notified of the
// end of the download via Finish.
//
// The pkg library never imports a progress bar UI library; consumers
// (e.g. the degit CLI) provide an adapter implementing this interface.
type Progress interface {
	io.Writer
	// Init is called once before bytes start flowing. total is the
	// response Content-Length, or -1 if the server did not advertise one.
	Init(total int64)
	// Finish is called after the last byte is written (success or error).
	Finish()
}
