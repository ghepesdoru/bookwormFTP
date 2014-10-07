package codes

/**
Replay codes definition
*/
const (
	/* 1xy - The requested action is being initiated, expect another reply before proceeding with a new command. */
	PositivePreliminary			 	= 100
	RestartMarkerReplay 			= 110
	ServiceReadyInNMinutes 			= 120
	DataConnectionAlreadyOpen 		= 125
	FileStatusOk 					= 150
	/* 2xy - The requested action has been successfully completed. */
	PositiveCompletion				= 200
	CommandNotImplemented 			= 202
	SystemStatus 					= 211
	DirectoryStatus 				= 212
	FileStatus 						= 213
	HelpMessage 					= 214
	NAMEType 						= 215
	Ready 							= 220
	ClosingControlConnection 		= 221
	DataConnectionOpen 				= 225
	DataConnectionClose 			= 226
	PassiveMode 					= 227
	LongPassiveMode 				= 228
	ExtendedPassiveMode 			= 229
	UserLoggedIn 					= 230
	UserLogOff_Termination 			= 231
	UserLogOff_QueuedTermination 	= 232
	FileActionOk 					= 250
	Pathname		 				= 257
	/* 3xy - The command has been accepted, but the requested action is on hold, pending receipt of
	further information. */
	PositiveIntermediate			= 300
	UserNameOk 						= 331
	AccountForLogin 				= 332
	FileActionPending 				= 350
	/* 4xy - The command was not accepted and the requested action did not take place, but the error
	condition is temporary and the action may be requested again. */
	TransientNegativeCompletion		= 400
	ServiceNotAvailable 			= 421
	DataConnectionFail 				= 425
	ConnectionClose					= 426
	InvalidAuthenticationData		= 430
	UnavailableHost					= 434
	ActionNotTaken					= 450
	ProcessingError					= 451
	ActionFailure					= 452
	/* 5xy - Syntax error, command unrecognized and the requested action did not take place.
	This may include errors such as command line too long. */
	PermanentNegativeCompletion		= 500
	SyntaxError						= 501
	NotImplemented					= 502
	BadSequence						= 503
	WrongParameters					= 504
	NotLoggedIn						= 530
	AuthenticationRequired			= 532
	FileUnavailable					= 550
	ActionAborted					= 551
	FileActionAborted				= 552
	BadFileName						= 553
	/* 6xy - Replies regarding confidentiality and integrity */
	ProtectedReply					= 600
	ProtectedIntegrity				= 631
	ProtectedConfAndIntegrity		= 632
	ProtectedConfidentiality		= 633
	/* 10xyz - Common Winsock Error Codes */
	ConnectionPeerReset				= 10054
	ConnectionFailed				= 10060
	ConnectionRefused				= 10061
	DirectoryNotEmpty				= 10066
	ServerFull						= 10068
)

/* Defines a map of known reply codes */
var KnownStatusCodes map[int]bool = map[int]bool {
	110: true, 120: true, 125: true, 150: true,
	200: true, 202: true, 211: true, 212: true, 213: true, 214: true, 215: true, 220: true, 221: true, 225: true,
	226: true, 227: true, 228: true, 229: true, 230: true, 231: true, 232: true, 250: true, 257: true,
	331: true, 332: true, 350: true,
	421: true, 425: true, 426: true, 430: true, 434: true, 450: true, 451: true, 452: true,
	500: true, 501: true, 502: true, 503: true, 504: true, 530: true, 532: true, 550: true, 551: true, 552: true, 553: true,
	631: true, 632: true, 633: true,
	10000: true, 10054: true, 10060: true, 10061: true, 10066: true, 10068: true,
}

/* Map of byte values to numeric counterparts */
var Numbers map[byte]int = map[byte]int {
	48: 0, 49: 1, 50: 2, 51: 3, 52: 4, 53: 5, 54: 6, 55: 7, 56: 8, 57: 9,
}

/* Checks if the given argument is a known valid status code */
func IsValid (status int) bool {
	if KnownStatusCodes[status] {
		return true
	}

	return false
}

/* Checks if the given argument can represent a valid number */
func ByteIsNumber (n byte) bool {
	_, ok := Numbers[n]
	return ok
}

/* Converts a slice of bytes into an int, breaking at first non numeric byte - only working for unsigned */
func ToInt (n []byte) int {
	var nr int = -1

	for _, v := range n {
		if !ByteIsNumber(v) {
			break
		} else {
			i, _ := Numbers[v]

			if nr == -1 {
				nr = i
			} else {
				nr = nr * 10 + i
			}
		}
	}

	return nr
}
