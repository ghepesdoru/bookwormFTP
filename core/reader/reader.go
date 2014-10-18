package reader

import (
	"fmt"
	"time"
	"io"
)

const (
	SIG_StartReading = iota
	SIG_StopReading
	SIG_WaitingForData
	SIG_DataRead
	SIG_Reset
	SIG_Process
	SIG_ProcessingFinished
	SIG_Done
	SIG_Quit
)

const (
	MAX_RESPONSE_SIZE = 1024
	READING_ACCEPTED_FAILURES = 4
)

/* Read delay constant */
var (
	DELAY_READ time.Duration = 50 * time.Millisecond
	DELAY_WAIT_FOR_READ time.Duration = 150 * time.Millisecond
	ERROR_ConnectionClosed = fmt.Errorf("use of closed network connection")
	ERROR_EOF	= io.EOF
	STATUS = map[int]string {
		SIG_WaitingForData: "Waiting for data",
		SIG_DataRead: "Data read",
		SIG_Done: "Action completed successfully",
	}
)

/* Bookworm FTP reader control connection */
type Reader struct {
	connection 			io.Reader
	controlChannel 		chan int
	errorChannel		chan error
	dataChannel			chan []byte
	rawData				[]byte
	lastError			*error
	connAvailable		bool
	active				bool
	status				int
}

/* Instantiate a new Reader and initialize the channel selection */
func NewReader(conn io.Reader) (reader *Reader) {
	/* Instantiate a new Reader */
	reader = &Reader{conn, make(chan int), make(chan error), make(chan []byte), []byte{}, nil, true, false, SIG_Done}

	/* Initialize channel listening for the current Reader */
	go reader.listen()

	time.Sleep(DELAY_READ)

	/* Put the reader into reading status */
	reader.controlChannel <- SIG_StartReading
	<- reader.controlChannel

	return
}

/* Data getter */
func (r *Reader) Get() []byte {
	return r.getRaw(0, READING_ACCEPTED_FAILURES, true)
}

/* Data get blocking for a longer time */
func (r *Reader) GetBlock() []byte {
	return r.getRaw(0, READING_ACCEPTED_FAILURES * READING_ACCEPTED_FAILURES, true)
}

/* Last encountered error getter */
func (r *Reader) GetError() error {
	return *r.lastError
}

/* Data getter. No extra delays */
func (r *Reader) GetNow() []byte {
	return r.getRaw(0, 0, true)
}

/* Checks if the reader encountered any errors */
func (r *Reader) HasErrors() bool {
	return r.lastError == nil
}

/* Checks if the reader is active */
func (r *Reader) IsActive() bool {
	return r.active && r.connAvailable
}

/* Peek to the raw data contents at any given time without flushing the buffer */
func (r *Reader) Peek() []byte {
	return r.getRaw(0, READING_ACCEPTED_FAILURES, false)
}

/* Reads any new content from the moment of calling the function (flushes any old content) */
func (r *Reader) Read() []byte {
	/* Reset content buffer */
	r.controlChannel <- SIG_Reset
	<- r.controlChannel
	return r.GetBlock()
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
	return r.status
}

/* Stops the reader from reading any other input, as soon as possible */
func (r *Reader) StopReading() {
	r.connAvailable = false
}

/* Raw data getter, supports a number of failures and imposes delays */
func (r *Reader) getRaw(n int, READING_ACCEPTED_FAILURES int, flush bool) []byte {
	if n < READING_ACCEPTED_FAILURES {
		if r.status == SIG_WaitingForData {
			/* If the reader is waiting for data, offer it a delay of DELAY_WAIT_FOR_READ before restart */
			time.Sleep(DELAY_WAIT_FOR_READ)
			return r.getRaw(n + 1, READING_ACCEPTED_FAILURES, flush)
		} else {
			/* If the reader has read some data, offer it a few more milliseconds, maybe the response
			spawns on multiple lines */
			time.Sleep(DELAY_READ)
			return r.getRaw(n + 1, READING_ACCEPTED_FAILURES, flush)
		}
	} else {
		data := append(r.rawData, []byte{}...)
		if flush {
			r.controlChannel <- SIG_Reset
			<- r.controlChannel
		}
		return data
	}
}

/* Listen for control, data and error channels */
func (r *Reader) listen() {
	var n int
	var err error
	var data []byte

	for {
		select {
		/* Signal received */
		case sig := <- r.controlChannel:
			switch sig {
			case SIG_StartReading:
				/* Start a go routine that will infinitely read raw data from the server, up to the connection closing */
				if r.connAvailable && !r.active {
					r.active = true

					go func() {
//						fmt.Println("Connection opened")
						for r.connAvailable && r.active {
							data = make([]byte, MAX_RESPONSE_SIZE)

							/* The reader will wait forever for data at this point, notify */
							r.controlChannel <- SIG_WaitingForData
							<- r.controlChannel

							/* Read incoming data */
							n, err = r.connection.Read(data)

							if err != nil {
								/* Break the loop on error */
								r.errorChannel <- err
								<- r.errorChannel
							} else if n > 0 {
								/* Forward data throw data channel */
								r.dataChannel <- data[:n]

								/* The reader read some data or encountered an error, notify */
								r.controlChannel <- SIG_DataRead
								<- r.controlChannel
							}
						}
//						fmt.Println("Connection closed or otherwise unavailable")
					}();
				}

				/* Notify the end of the current action */
				r.controlChannel <- SIG_Done

			case SIG_WaitingForData:
				/* The reading go routine is waiting for data, this might take forever, set a flag to sync data getter */
				r.status = SIG_WaitingForData
				r.controlChannel <- SIG_Done

			case SIG_DataRead:
				r.status = SIG_DataRead
				r.controlChannel <- SIG_Done

			case SIG_StopReading:
				/* Stop the reading go routine */
				r.active = false
				r.controlChannel <- SIG_Done

			case SIG_Reset:
				/* Empty the raw read lines buffer */
				r.rawData = []byte{}
				r.controlChannel <- SIG_Done

			default:
		}

		/* Data read */
		case data := <- r.dataChannel:
			/* Append the newly read data to the raw data keeper */
			r.rawData = append(r.rawData, data...)

		/* Data channel encountered an error. Stop reading, and remember the error! */
		case err := <- r.errorChannel:
			r.lastError = &err

			if err == ERROR_EOF {
				r.StopReading()
			} else if err == ERROR_ConnectionClosed {
				r.StopReading()
			}

			r.errorChannel <- nil
		}
	}
}