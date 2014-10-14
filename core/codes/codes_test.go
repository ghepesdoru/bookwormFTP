package codes

import (
	"testing"
)

var (
	Number = 123456
	NumberBytes = []byte{49, 50, 51, 52, 53, 54}
	IANA_Statuses = map[string][]int {
		"1": []int{100, 110, 120, 125, 150},
		"2": []int{200, 202, 211, 212, 213, 214, 215, 220, 221, 225, 226, 227, 228, 229, 230, 231, 232, 250, 257},
		"3": []int{300, 331, 332, 350},
		"4": []int{400, 421, 425, 426, 430, 434, 450, 451, 452},
		"5": []int{500, 501, 502, 503, 504, 530, 532, 550, 551, 552, 553},
		"6": []int{600, 631, 632, 633},
	}
)

func TestByteToNumber(t *testing.T) {
	if ToInt(NumberBytes) != Number {
		t.Fatal("Invalid byte to number conversion")
	}
}

func TestStandardStatuses(t *testing.T) {
	for statusType, statuses := range IANA_Statuses {
		for _, s := range statuses {
			if !IsValid(s) {
				t.Fatal("Standard status " + statusType +  "xy not recognised by current implementation.", s)
			}
		}
	}
}

/* Scraper for: http://en.wikipedia.org/wiki/List_of_FTP_server_return_codes */
//var rows = document.getElementsByClassName("wikitable")[2].rows,
//	statuses = {all: []},
//	tagName = "TD",
//	i,
//	len = rows.length;
//
//for (i =0; i < len; i += 1) {
//	var status,
//		statusType;
//
//	if (rows[i].children[0].tagName != tagName) {
//		continue;
//	}
//
//	status = rows[i].children[0].querySelector("code").innerText.trim();
//	status = status.length > 0 && status.indexOf(" Series") > -1 ? status.substr(0,status.indexOf(" Series")) : status;
//	status = parseInt(status, 10);
//
//	if (isNaN(status) || status > 700) {
//		continue;
//	}
//
//	statusType = Math.floor(status / 100);
//
//	if (!statuses[statusType]) {
//		statuses[statusType] = [];
//	}
//
//	statuses[statusType].push(status);
//	statuses.all.push(status);
//}
//
//statuses[2].join(", ")
