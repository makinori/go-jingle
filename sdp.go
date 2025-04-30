package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/beevik/etree"
)

// https://github.com/jitsi/lib-jitsi-meet/blob/master/modules/sdp/SDP.js
// to convert between jingle and regular sdp

const (
	XEP_BUNDLE_MEDIA          = "urn:xmpp:jingle:apps:grouping:0"
	XEP_DTLS_SRTP             = "urn:xmpp:jingle:apps:dtls:0"
	XEP_ICE_UDP_TRANSPORT     = "urn:xmpp:jingle:transports:ice-udp:1"
	XEP_JINGLE                = "urn:xmpp:jingle:1"
	XEP_RAYO                  = "urn:xmpp:rayo:client:1"
	XEP_RTP_AUDIO             = "urn:xmpp:jingle:apps:rtp:audio"
	XEP_RTP_FEEDBACK          = "urn:xmpp:jingle:apps:rtp:rtcp-fb:0"
	XEP_RTP_HEADER_EXTENSIONS = "urn:xmpp:jingle:apps:rtp:rtp-hdrext:0"
	XEP_RTP_MEDIA             = "urn:xmpp:jingle:apps:rtp:1"
	XEP_RTP_VIDEO             = "urn:xmpp:jingle:apps:rtp:video"
	XEP_SCTP_DATA_CHANNEL     = "urn:xmpp:jingle:transports:dtls-sctp:1"
	XEP_SOURCE_ATTRIBUTES     = "urn:xmpp:jingle:apps:rtp:ssma:0"
)

type SDP struct {
	IsP2P bool

	Raw     string
	Session string

	RemoveTcpCandidates bool
	RemoveUdpCandidates bool

	FailICE bool
}

func adjustMsidSemantic(msid string, mediaType string, idx string) string {
	if mediaType == "audio" { // || !isChromiumBased || isEngineVersionGreaterThan(116)
		return msid
	}

	msidParts := strings.Split(msid, " ")

	if len(msidParts) == 2 {
		return msid
	}

	return fmt.Sprintf("%s %s-%s", msid, msid, idx)
}

func (this *SDP) FromJingle(jingle *etree.Element) {
	sessionID := time.Now().UnixMilli()

	this.Raw = "v=0\r\n" +
		fmt.Sprintf("o=- %d 2 IN IP4 0.0.0.0\r\n", sessionID) +
		"s=-\r\n" +
		"t=0 0\r\n"

	groups := jingle.FindElements(`group[@xmlns="` + XEP_BUNDLE_MEDIA + `"]`)

	if this.IsP2P && len(groups) > 0 {
		for _, group := range groups {
			var contents []string

			for _, content := range group.FindElements("content") {
				name := content.SelectAttr("name")
				if name != nil {
					contents = append(contents, name.Value)
				}
			}

			if len(contents) > 0 {

				semantics := group.SelectAttr("semantics")
				if semantics == nil {
					semantics = group.SelectAttr("type")
				}
				if semantics == nil {
					continue
				}

				this.Raw += "a=group:" + semantics.Value + " " +
					strings.Join(contents, " ") + "\r\n"
			}
		}
	}

	this.Session = this.Raw

	for _, content := range jingle.FindElements("content") {
		m := this.Jingle2Media(content)

		// TODO: line 221

		fmt.Println(m)
	}

}

// Converts the content section from Jingle to a media section that can be appended to the SDP.
func (this *SDP) Jingle2Media(content *etree.Element) string {
	desc := content.FindElement("description")
	transport := content.FindElement(`transport[@xmlns="` + XEP_ICE_UDP_TRANSPORT + `"]`)

	var sdp string

	sctp := transport.FindElement(`sctpmap[@xmlns="` + XEP_SCTP_DATA_CHANNEL + `"]`)

	media := MLine{
		Media: desc.SelectAttrValue("media", ""),
	}

	mid := content.SelectAttrValue("name", "")

	media.Port = 9
	if content.SelectAttrValue("senders", "") == "rejected" {
		media.Port = 0
	}

	if len(transport.FindElements(`fingerprint[@xmlns="`+XEP_DTLS_SRTP+`"]`)) > 0 {
		media.Proto = "UDP/DTLS/SCTP"
	} else {
		media.Proto = "UDP/TLS/RTP/SAVPF"
	}

	if sctp != nil {
		sdp += fmt.Sprintf("m=application %d UDP/DTLS/SCTP webrtc-datachannel\r\n", media.Port)
		sdp += fmt.Sprintf("a=sctp-port:%s\r\n", sctp.SelectAttrValue("number", ""))
		sdp += "a=max-message-size:262144\r\n"
	} else {
		for _, payloadType := range desc.FindElements("payload-type") {
			id := payloadType.SelectAttr("id")
			if id != nil {
				media.Fmt = append(media.Fmt, id.Value)
			}
		}
		sdp += BuildMLine(media) + "\r\n"
	}

	sdp += "c=IN IP4 0.0.0.0\r\n"
	if sctp == nil {
		sdp += "a=rtcp:1 IN IP4 0.0.0.0\r\n"
	}

	if transport != nil {
		ufrag := transport.SelectAttr("ufrag")
		if ufrag != nil {
			sdp += BuildICEUfrag(ufrag.Value) + "\r\n"
		}

		pwd := transport.SelectAttr("pwd")
		if pwd != nil {
			sdp += BuildICEPwd(pwd.Value) + "\r\n"
		}

		for _, fingerprint := range transport.FindElements(
			`fingerprint[@xmlns="` + XEP_DTLS_SRTP + `"]`,
		) {
			sdp += fmt.Sprintf(
				"a=fingerprint:%s %s\r\n",
				fingerprint.SelectAttrValue("hash", ""), fingerprint.Text(),
			)

			setup := fingerprint.SelectAttr("setup")
			if setup != nil {
				sdp += fmt.Sprintf("a=setup:%s\r\n", setup.Value)
			}
		}
	}

	for _, candidate := range transport.FindElements("candidate") {
		protocol := candidate.SelectAttrValue("protocol", "")
		protocol = strings.ToLower(protocol)

		if (this.RemoveTcpCandidates && (protocol == "tcp" || protocol == "ssltcp")) ||
			(this.RemoveUdpCandidates && protocol == "udp") {
			break
		} else if this.FailICE {
			candidate.CreateAttr("ip", "1.1.1.1")
		}

		sdp += CandidateFromJingle(candidate)
	}

	switch content.SelectAttrValue("senders", "") {
	case "initiator":
		sdp += "a=sendonly\r\n"
	case "responder":
		sdp += "a=recvonly\r\n"
	case "none":
		sdp += "a=inactive\r\n"
	case "both":
		sdp += "a=sendrecv\r\n"
	}

	sdp += fmt.Sprintf("a=mid:%s\r\n", mid)

	// <description><rtcp-mux/></description>
	// see http://code.google.com/p/libjingle/issues/detail?id=309 -- no spec though
	// and http://mail.jabber.org/pipermail/jingle/2011-December/001761.html
	if desc.FindElement("rtcp-mux") != nil {
		sdp += "a=rtcp-mux\r\n"
	}

	for _, payloadType := range desc.FindElements("payload-type") {
		sdp += BuildRTPMap(payloadType) + "\r\n"

		parameters := payloadType.FindElements("parameter")
		if len(parameters) > 0 {
			sdp += "a=fmtp:" + payloadType.SelectAttrValue("id", "") + " "

			var paramatersLine []string
			for _, parameter := range parameters {
				name := parameter.SelectAttr("name")
				var parameterLine string
				if name != nil {
					parameterLine = name.Value + "="
				}
				parameterLine += parameter.SelectAttrValue("value", "")
				paramatersLine = append(paramatersLine, parameterLine)
			}

			sdp += strings.Join(paramatersLine, ";") + "\r\n"
		}

		sdp += RtcpFbFromJingle(payloadType, payloadType.SelectAttrValue("id", ""))
	}

	sdp += RtcpFbFromJingle(desc, "*")

	hdrExts := desc.FindElements(`rtp-hdrext[@xmlns="` + XEP_RTP_HEADER_EXTENSIONS + `"]`)
	for _, hdrExt := range hdrExts {
		sdp += fmt.Sprintf(
			"a=extmap:%s %s\r\n", hdrExt.SelectAttrValue("id", ""),
			hdrExt.SelectAttrValue("uri", ""),
		)
	}

	if desc.FindElement(`extmap-allow-mixed[@xmlns="`+XEP_RTP_HEADER_EXTENSIONS+`"]`) != nil {
		sdp += "a=extmap-allow-mixed\r\n"
	}

	ssrcGroups := desc.FindElements(`ssrc-group[@xmlns="` + XEP_SOURCE_ATTRIBUTES + `"]`)
	for _, ssrcGroup := range ssrcGroups {
		semantics := ssrcGroup.SelectAttrValue("semantics", "")

		ssrcs := []string{}
		for _, source := range ssrcGroup.FindElements("source") {
			ssrcs = append(ssrcs, source.SelectAttrValue("ssrc", ""))
		}

		if len(ssrcs) > 0 {
			sdp += fmt.Sprintf("a=ssrc-group:%s %s\r\n", semantics, strings.Join(ssrcs, ""))
		}
	}

	userSources := ""
	nonUserSources := ""

	for _, source := range desc.FindElements(`source[@xmlns="` + XEP_SOURCE_ATTRIBUTES + `"]`) {
		ssrc := source.SelectAttrValue("ssrc", "")
		isUserSource := true
		sourceStr := ""

		for _, parameter := range source.FindElements("parameter") {
			name := parameter.SelectAttrValue("name", "")
			value := parameter.SelectAttrValue("value", "")

			value = FilterSpecialChars(value)
			sourceStr += fmt.Sprintf("a=ssrc:%s %s", ssrc, name)

			if name == "msid" {
				value = adjustMsidSemantic(value, media.Media, mid)
			}

			if value != "" {
				sourceStr += ":" + value
			}

			sourceStr += "\r\n"

			if strings.Contains(value, "mixedmslabel") {
				isUserSource = false
			}
		}

		if isUserSource {
			userSources += sourceStr
		} else {
			nonUserSources += sourceStr
		}
	}

	// Append sources in the correct order, the mixedmslable m-line which has the JVB's SSRC for RTCP termination
	// is expected to be in the first m-line.
	sdp += nonUserSources + userSources

	fmt.Println("SDP:\n" + sdp)
	return sdp
}

func RtcpFbFromJingle(elem *etree.Element, payloadType string) string {
	var sdp string

	feedbackElementTrrInt := elem.FindElement(
		`rtcp-fb-trr-int[@xmlns=` + XEP_RTP_FEEDBACK + `]`,
	)

	if feedbackElementTrrInt != nil {
		sdp += "a=rtcp-fb:* trr-int "
		sdp += feedbackElementTrrInt.SelectAttrValue("value", "0")
		sdp += "\r\n"
	}

	feedbackElements := elem.FindElements(`rtcp-fb[@xmlns="` + XEP_RTP_FEEDBACK + `"]`)

	for _, fb := range feedbackElements {
		sdp += fmt.Sprintf("a=rtcp-fb:%s %s", payloadType, fb.SelectAttrValue("type", ""))
		subtype := fb.SelectAttr("subtype")
		if subtype != nil {
			sdp += " " + subtype.Value
		}
		sdp += "\r\n"
	}

	return sdp
}
