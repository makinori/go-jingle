package main

import (
	"fmt"
	"strings"

	"github.com/beevik/etree"
)

type MLine struct {
	Media string
	Port  int
	Proto string
	Fmt   []string
}

func FilterSpecialChars(text string) string {
	// XXX Neither one of the falsy values (e.g. null, undefined, false,
	// "", etc.) "contain" special chars.
	// eslint-disable-next-line no-useless-escape
	// TODO: fix this
	return text
	// return text ? text.replace(/[\\\/\{,\}\+]/g, '') : text;
}

func BuildICEUfrag(frag string) string {
	return "a=ice-ufrag:" + frag
}

func BuildICEPwd(frag string) string {
	return "a=ice-pwd:" + frag
}

func BuildMLine(mline MLine) string {
	return fmt.Sprintf(
		"m=%s %d %s %s", mline.Media, mline.Port,
		mline.Proto, strings.Join(mline.Fmt, " "),
	)
}

func BuildRTPMap(el *etree.Element) string {
	line := fmt.Sprintf("a=rtpmap:%s %s/%s",
		el.SelectAttrValue("id", ""),
		el.SelectAttrValue("name", ""),
		el.SelectAttrValue("clockrate", ""),
	)

	channels := el.SelectAttr("channels")
	if channels != nil && channels.Value != "1" {
		line += fmt.Sprintf("/%s", channels.Value)
	}

	return line
}

func CandidateFromJingle(cand *etree.Element) string {
	line := "a=candidate:"

	line += cand.SelectAttrValue("foundation", "")
	line += " "
	line += cand.SelectAttrValue("component", "")
	line += " "

	protocol := cand.SelectAttrValue("protocol", "")

	// if firefox {
	// 	protocol = "tcp"
	// }

	line += protocol
	line += " "
	line += cand.SelectAttrValue("priority", "")
	line += " "
	line += cand.SelectAttrValue("ip", "")
	line += " "
	line += cand.SelectAttrValue("port", "")
	line += " "
	line += "typ"
	line += " " + cand.SelectAttrValue("type", "")
	line += " "

	switch cand.SelectAttrValue("type", "") {
	case "srflx", "prflx", "relay":
		relAddr := cand.SelectAttr("rel-addr")
		relPort := cand.SelectAttr("rel-port")
		if relAddr != nil && relPort != nil {
			line += "raddr"
			line += " "
			line += relAddr.Value
			line += " "
			line += "rport"
			line += " "
			line += relPort.Value
			line += " "
		}
	}

	if strings.ToLower(protocol) == "tcp" {
		line += "tcptype"
		line += " "
		line += cand.SelectAttrValue("tcptype", "")
		line += " "
	}

	line += "generation"
	line += " "
	line += cand.SelectAttrValue("generation", "")

	return line + "\r\n"
}
