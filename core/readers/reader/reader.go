package reader

import (
	"net"
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
)

/* Bookworm FTP reader control connection */
type Reader struct {
	connection 			net.Conn
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
func NewReader(conn net.Conn) (reader *Reader) {
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
	return r.getRaw(0)
}

/* Reads any new content from the moment of calling the function (flushes any old content) */
func (r *Reader) Read() []byte {
	/* Reset content buffer */
	r.controlChannel <- SIG_Reset
	<- r.controlChannel

	return r.Get()
}

/* Checks if the reader encountered any errors */
func (r *Reader) HasErrors() bool {
	return r.lastError == nil
}

/* Last encountered error getter */
func (r *Reader) GetError() error {
	return *r.lastError
}

/* Raw data getter, supports a number of failures and imposes delays */
func (r *Reader) getRaw(n int) []byte {
	if n < READING_ACCEPTED_FAILURES {
		if r.status == SIG_WaitingForData {
			/* If the reader is waiting for data, offer it a delay of DELAY_WAIT_FOR_READ before restart */
			time.Sleep(DELAY_WAIT_FOR_READ)
			return r.getRaw(n + 1)
		} else {
			/* If the reader has read some data, offer it a few more milliseconds, maybe the response
			spawns on multiple lines */
			time.Sleep(DELAY_READ)
			return r.getRaw(n + 1)
		}
	} else {
		data := append(r.rawData, []byte{}...)
		r.controlChannel <- SIG_Reset
		<- r.controlChannel
		return data
	}
}

/* Listen for control, data and error channels */
func (r *Reader) listen() {
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
						for r.connAvailable && r.active {
							data := make([]byte, MAX_RESPONSE_SIZE)

							/* The reader will wait forever for data at this point, notify */
							r.controlChannel <- SIG_WaitingForData
							<- r.controlChannel

							/* Read incoming data */
							n, err := r.connection.Read(data)

							if err != nil {
								/* Break the loop on error */
								r.errorChannel <- err
								return
							} else if n > 0 {
								/* Forward data throw data channel */
								r.dataChannel <- data

								/* The reader read some data or encountered an error, notify */
								r.controlChannel <- SIG_DataRead
								<- r.controlChannel
							}
						}
					}();
				}

				/* Notify the end of the current action */
				r.controlChannel <- SIG_Done
				break

			case SIG_WaitingForData:
				/* The reading go routine is waiting for data, this might take forever, set a flag to sync data getter */
				r.status = SIG_WaitingForData
				r.controlChannel <- SIG_Done
				break

			case SIG_DataRead:
				r.status = SIG_DataRead
				r.controlChannel <- SIG_Done
				break

			case SIG_StopReading:
				/* Stop the reading go routine */
				r.active = false
				break

			case SIG_Reset:
				/* Empty the raw read lines buffer */
				r.rawData = []byte{}
				r.controlChannel <- SIG_Done
				break

			default:
				fmt.Println("Uncaught signal: ", sig)
				break
			}
			break

		/* Data read */
		case data := <- r.dataChannel:
			/* Append the newly read data to the raw data keeper */
			r.rawData = append(r.rawData, data...)
			break;

		/* Data channel encountered an error. Stop reading, and remember the error! */
		case err := <- r.errorChannel:
			r.lastError = &err

			if err == ERROR_EOF {
				r.connAvailable = false
			} else if err == ERROR_ConnectionClosed {
				r.connAvailable = false
			}

			/* Notify the reader that no reading is taking place anymore */
			go func(){
				r.controlChannel <- SIG_StopReading
			}()

			break;
		}
	}
}
