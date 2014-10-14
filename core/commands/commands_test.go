package commands

import "testing"

var (
	IANA_MandatoryCommands = []string{"ABOR", "ACCT", "ALLO", "APPE", "CWD", "DELE", "FEAT", "HELP", "LIST", "MODE", "NLST", "NOOP", "OPTS", "PASS", "PASV", "PORT", "QUIT", "REIN", "REST", "REST+", "RETR", "RNFR", "RNTO", "SITE", "STAT", "STOR", "STRU", "TYPE", "USER"}
	IANA_OptionalCommands = []string{"ADAT", "ALGS", "AUTH", "AUTH+", "CCC", "CDUP", "CONF", "ENC", "EPRT", "EPSV", "HOST", "LANG", "MDTM", "MIC", "MKD", "MLSD", "MLST", "PBSZ", "PBSZ+", "PROT", "PROT+", "PWD", "RMD", "SIZE", "SMNT", "STOU", "SYST"}
	IANA_HistoricCommands = []string{"LPRT", "LPSV", "XCUP", "XCWD", "XMKD", "XPWD", "XRMD"}
)

func TestToStandardCommand(t *testing.T) {
	if ToStandardCommand("abor") != "ABOR" {
		t.Fatal("Invalid ToStandardCommand behaviour.")
	}
}

func TestObsoleteCommands(t *testing.T) {
	for _, c := range IANA_HistoricCommands {
		if !IsObsolete(c) {
			t.Fatal("Invalid command category.", c)
		}
	}
}

func TestObsoleteToCurrent(t *testing.T) {
	for _, c := range IANA_HistoricCommands {
		if !IsValid(ToStandardCommand(c)) {
			t.Fatal("Obsolete command has no mapping to any current command.", c)
		}
	}
}

func TestStandardCommands(t *testing.T) {
	for _, c := range IANA_MandatoryCommands {
		if !IsValid(c) {
			t.Fatal("Mandatory command does not exist in the commands definition.", c)
		}
	}

	for _, c := range IANA_OptionalCommands {
		if !IsValid(c) {
			t.Fatal("Optional command does not exist in commands definition.", c)
		}
	}
}



/* Scraper for: http://www.iana.org/assignments/ftp-commands-extensions/ftp-commands-extensions.xhtml */
//var rows = document.getElementById("table-ftp-commands-extensions-2").querySelectorAll('tr'),
//	commands = {all: []},
//	tagName = "TD",
//	i,
//	len = rows.length;
//
//for (i =0; i < len; i += 1) {
//	var command,
//		commandType;
//
//	if (rows[i].children[0].tagName != tagName) {
//		continue;
//	}
//
//	command = rows[i].children[0].innerText.trim();
//	commandType = rows[i].children[4].innerText.trim().toUpperCase();
//
//	if (commandType.length > 1) {
//		commandType = commandType.substr(0,1)
//	}
//
//	if (command.indexOf("-") > -1) {
//		continue;
//	}
//
//	if (!commands[commandType]) {
//		commands[commandType] = [];
//	}
//
//	commands[commandType].push(command);
//	commands.all.push(command);
//}
//
// commands.O.map(function(v,i){commands.O[i] = "\""+v+"\"";})
