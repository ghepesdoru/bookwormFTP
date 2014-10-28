package commands

import (
	"fmt"
	"strings"
)

const (
	UnknownCommand = "Unknown"
)

var (
	ERR_InvalidCommandName = fmt.Errorf("Invalid command.")
)

/* Defines a map of known commands */
var KnownCommands map[string]bool = map[string]bool {
	"ABOR": true,	"ACCT": true, 	"ADAT": true, 	"ALGS": true, 	"ALLO": true, 	"APPE": true, 	"AUTH": true,
	"AUTH+": true, 	"CCC": true,	"CDUP": true, 	"CONF": true, 	"CWD": true, 	"DELE": true, 	"ENC": true,
	"EPRT": true,	"EPSV": true,	"FEAT": true, 	"HELP": true, 	"HOST": true,	"LANG": true, 	"LIST": true,
	"MDTM": true, 	"MIC": true,	"MKD": true,	"MLSD": true, 	"MLST": true, 	"MODE": true, 	"NLST": true,
	"NOOP": true,	"OPTS": true,	"OPTS_UTF8": true,				"PASS": true, 	"PASV": true, 	"PBSZ": true,
	"PBSZ+": true,	"PORT": true, 	"PROT": true, 	"PROT+": true,	"PWD": true, 	"QUIT": true,	"REIN": true,
	"REST": true,	"REST+": true, 	"RETR": true,	"RMD": true,	"RNFR": true,	"RNTO": true, 	"SITE": true,
	"SIZE": true, 	"SMNT": true, 	"STAT": true, 	"STOR": true,	"STOU": true,	"STRU": true, 	"SYST": true,
	"TYPE": true, 	"USER": true,
}

/* Defines a map of base commands, using the bool value to mark if the feature is mandatory */
var BaseCommands map[string]bool = map[string]bool {
	"ABOR": true,	"ACCT": true,	"ALLO": true,	"APPE": true,	"CDUP": false,	"CWD": true,	"DELE": true,
	"HELP": true,	"LIST": true,	"MKD": false,	"MODE": true,	"NLST": true,	"NOOP": true,	"PASS": true,
	"PASV": true,	"PORT": true,	"PWD": false,	"QUIT": true,	"REIN": true,	"REST": true,	"RETR": true,
	"RMD": false,	"RNFR": true,	"RNTO": true,	"SITE": true,	"SMNT": false,	"STAT": true,	"STOR": true,
	"STOU": false,	"STRU": true,	"SYST": false,	"TYPE": true,	"USER": true,
}

/* Defines a list of known obsolete commands */
var ObsoleteCommands map[string]bool = map[string]bool {
	"LPRT": true, "LPSV": true, "XCUP": true, "XCWD": true, "XMKD": true, "XPWD": true, "XRMD": true,
}

/* Maps historic commands to the standards compliant counterparts */
var ObsoleteToKnown map[string]string = map[string]string {
	"LPRT": "EPRT",	"LPSV": "EPSV",	"XCUP": "CDUP",	"XCWD": "CWD",	"XMKD": "MKD",	"XPWD": "PWD",	"XRMD": "RMD",
}

var RFC3659 = map[string]bool {
	/* Mapping of commands defined in RFC3659 -
	practically most servers implement FTP up to a specified standard version.
	Testing for feature existence outside the FEAT reply listings often ends finding
	non explicitly specified working features. */
	"MDTM": true,	"MLST": true,	"MLSD": true,	"REST+": true,	"SIZE": true, 	"TVFS": true,
}

/* Checks if the specified command represents a standard base feature */
func IsBase(feature string) bool {
	if f := ToStandardCommand(feature); IsValid(f) {
		_, ok := BaseCommands[feature]
		return ok
	}

	return false
}

/* Checks if the specified command represents a mandatory feature */
func IsMandatory(feature string) bool {
	if f := ToStandardCommand(feature); IsBase(f) {
		return BaseCommands[feature]
	}

	return false
}

/* Checks if the specified COMMAND is obsolete (marked as hist in the IANA FTP commands list) */
func IsObsolete(cmd string) bool {
	if ObsoleteCommands[strings.ToUpper(strings.TrimSpace(cmd))] {
		return true
	}

	return false
}

func IsRFC3659(cmd string) bool {
	if RFC3659[strings.ToUpper(strings.TrimSpace(cmd))] {
		return true
	}

	return false
}

/* Checks if the specified argument is a known valid COMMAND */
func IsValid(cmd string) bool {
	if KnownCommands[cmd] {
		return true
	}

	return false
}

/* Maps the specified command to it's current standard compliant counterpart */
func ToStandardCommand(cmd string) string {
	/* Convert the COMMAND to it's canonical form */
	cmd = strings.ToUpper(strings.TrimSpace(cmd))

	if IsValid(cmd) {
		return cmd
	} else if IsObsolete(cmd) {
		return ObsoleteToKnown[cmd]
	}

	return UnknownCommand
}

