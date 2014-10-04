package commands

import "strings"

const (
	CONST_UnknownCommand = "Unknown"
)

/* Defines a map of known commands */
var KnownCommands map[string]bool = map[string]bool {
	"ABOR": true,	"ACCT": true, 	"ADAT": true, 	"ALLO": true, 	"APPE": true, 	"AUTH": true, 	"CCC": true,
	"CDUP": true, 	"CONF": true, 	"CWD": true, 	"DELE": true, 	"ENC": true, 	"EPRT": true,	"EPSV": true,
	"FEAT": true, 	"HELP": true, 	"LANG": true, 	"LIST": true, 	"MDTM": true, 	"MIC": true, 	"MKD": true,
	"MLSD": true, 	"MLST": true, 	"MODE": true, 	"NLST": true, 	"NOOP": true,	"OPTS": true, 	"OPTS_UTF8": true,
	"PASS": true, 	"PASV": true, 	"PBSZ": true, 	"PORT": true, 	"PROT": true, 	"PWD": true, 	"QUIT": true,
	"REIN": true, 	"REST": true, 	"RETR": true,	"RMD": true, 	"RNFR": true, 	"RNTO": true, 	"SITE": true,
	"SIZE": true, 	"SMNT": true, 	"STAT": true, 	"STOR": true, 	"STOU": true, 	"STRU": true, 	"SYST": true,
	"TYPE": true, 	"USER": true,
}

/* Defines a list of known obsolete commands */
var ObsoleteCommands map[string]bool = map[string]bool {
	"LPRT": true, "LPSV": true, "XCUP": true, "XCWD": true, "XMKD": true, "XPWD": true, "XRMD": true,
}

/* Maps historic commands to the standards compliant counterparts */
var ObsoleteToKnown map[string]string = map[string]string {
	// TODO: Map obsolete commands to current commands!
}

/* Checks if the specified argument is a known valid COMMAND */
func IsValid(cmd string) bool {
	if KnownCommands[cmd] {
		return true
	}

	return false
}

/* Checks if the specified COMMAND is obsolete (marked as hist in the IANA FTP commands list) */
func IsObsolete(cmd string) bool {
	if ObsoleteCommands[cmd] {
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

	return CONST_UnknownCommand
}

