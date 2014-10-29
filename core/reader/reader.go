package reader

import (
	"io"
	"bytes"
	"time"
)

type SIG_Status int
const (
	SIG_WaitingForData	SIG_Status = iota
	SIG_DataRead
	SIG_Done
)

const (
	MAX_RESPONSE_SIZE         = 24
	READING_ACCEPTED_FAILURES = 4
)

/* Read delay constant */
var (
	DELAY_READ             time.Duration = 100 * time.Millisecond
	DELAY_WAIT_FOR_READ    time.Duration = 100 * time.Millisecond
	ConnectionClosed               		 = "use of closed network connection"
	ERROR_EOF                            = io.EOF
	STATUS                               = map[SIG_Status]string{
		SIG_WaitingForData: "Waiting for data",
		SIG_DataRead:       "Data read",
		SIG_Done:           "Action completed successfully",
	}
)

/* Bookworm FTP reader */
type Reader struct {
	source		io.Reader
	destination	io.Writer
	buffer		*bytes.Buffer
	err			error
	status		SIG_Status
	sourceOk	bool
	active		bool
	outsource	bool
}

/* Instantiate a new Reader */
func NewReader(source io.Reader) (reader *Reader) {
	destination := bytes.NewBuffer([]byte{})
	reader = &Reader{source, nil, destination, nil, SIG_Done, true, false, false}
	reader.attachDestination(destination)
	return
}

/* Attach a destination writer to the current reader */
func (r *Reader) AttachDestination(w io.Writer) {
	r.attachDestination(w)
	r.outsource = true
}

/* Data getter */
func (r *Reader) Get() []byte {
	if !r.outsource {
		return r.getRaw(0, READING_ACCEPTED_FAILURES, true)
	}

	return []byte{}
}

/* Data get blocking for a longer time */
func (r *Reader) GetBlock() []byte {
	if !r.outsource {
		return r.getRaw(0, READING_ACCEPTED_FAILURES*READING_ACCEPTED_FAILURES, true)
	}

	return []byte{}
}

/* Last encountered error getter */
func (r *Reader) GetError() error {
	return r.err
}

/* Data getter. No extra delays */
func (r *Reader) GetNow() []byte {
	if !r.outsource {
		return r.getRaw(0, 0, true)
	}

	return []byte{}
}

/* Checks if the reader encountered any errors */
func (r *Reader) HasErrors() bool {
	return r.err == nil
}

/* Checks if the reader is active */
func (r *Reader) IsActive() bool {
	return r.active && r.sourceOk
}

/* Peek to the raw data contents at any given time without flushing the buffer */
func (r *Reader) Peek() []byte {
	if !r.outsource {
		return r.getRaw(0, READING_ACCEPTED_FAILURES, false)
	}

	return []byte{}
}

/* io.Reader interface implementation */
func (r *Reader) Read(p []byte) (n int, err error) {
	n, err = r.source.Read(p)
	if err != nil {
		if err == ERROR_EOF || ConnectionClosed == err.Error() {
			r.sourceOk = false
		}

		r.err = err
		r.StopReading()
	}

	return
}

/* Reset the internal buffer if the current reader is not outsourcing */
func (r *Reader) Reset() {
	if !r.outsource {
		r.buffer.Reset()
	}
}

/* Returns a human readable description of the current reader's status */
func (r *Reader) Status() string {
	if s, ok := STATUS[r.status]; ok {
		return s
	}

	return ""
}

/* Returns the code associated with the current status */
func (r *Reader) StatusCode() int {
	return int(r.status)
}

/* Stops the reader from reading any other input */
func (r *Reader) StopReading() {
	r.active = false
}

/* io.Writer interface implementation */
func (r *Reader) Write(p []byte) (n int, err error) {
	n, err = r.destination.Write(p)
	if err != nil {
		r.err = err
		r.StopReading()
	}

	return
}

/* Attach a new destination io.Writer to the current infinite reading process */
func (r *Reader) attachDestination(w io.Writer) {
	/* Stop reading if active */
	if r.IsActive() {
		r.StopReading()
	}

	/* Start listening for content */
	r.destination = w
	go r.listen()
	time.Sleep(DELAY_WAIT_FOR_READ)
}

/* Raw data getter, supports a number of failures and imposes delays */
func (r *Reader) getRaw(n int, READING_ACCEPTED_FAILURES int, flush bool) []byte {
	if n < READING_ACCEPTED_FAILURES {
		if r.status == SIG_WaitingForData {
			/* If the reader is waiting for data, offer it a delay of DELAY_WAIT_FOR_READ before restart */
			time.Sleep(DELAY_WAIT_FOR_READ)
			return r.getRaw(n+1, READING_ACCEPTED_FAILURES, flush)
		} else {
			/* If the reader has read some data, offer it a few more milliseconds, maybe the response
			spawns on multiple lines */
			time.Sleep(DELAY_READ)
			return r.getRaw(n+1, READING_ACCEPTED_FAILURES, flush)
		}
	} else if !r.outsource {
		data := r.buffer.Bytes()
		if flush {
			r.Reset()
		}

		return data
	}

	return []byte{}
}

/* Listen for data */
func (r *Reader) listen() {
	var data []byte = make([]byte, MAX_RESPONSE_SIZE)
	var n int
	var err error

	r.active = true

	for r.IsActive() {
		r.status = SIG_WaitingForData
		if n, err = r.Read(data); err == nil {
			r.status = SIG_DataRead
			n, err = r.Write(data[:n])
		}
	}
}
