package jitsi

import (
	"fmt"
	"strings"

	"github.com/beevik/etree"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

func marshalAndSend(conn *websocket.Conn, doc *etree.Document) error {
	msg, err := doc.WriteToBytes()
	if err != nil {
		return err
	}

	fmt.Println("")
	fmt.Println("OUTGOING: " + string(msg))

	conn.WriteMessage(websocket.TextMessage, msg)
	return nil
}

// <open to="meet.jitsi" version="1.0" xmlns="urn:ietf:params:xml:ns:xmpp-framing"/>

func sendOpenFraming(conn *websocket.Conn) error {
	doc := etree.NewDocument()

	open := doc.CreateElement("open")
	open.CreateAttr("to", "meet.jitsi")
	open.CreateAttr("version", "1.0")
	open.CreateAttr("xmlns", "urn:ietf:params:xml:ns:xmpp-framing")

	return marshalAndSend(conn, doc)
}

// <auth mechanism="ANONYMOUS" xmlns="urn:ietf:params:xml:ns:xmpp-sasl"/>

func sendAnonAuthSasl(conn *websocket.Conn) error {
	doc := etree.NewDocument()

	auth := doc.CreateElement("auth")
	auth.CreateAttr("mechanism", "ANONYMOUS")
	auth.CreateAttr("xmlns", "urn:ietf:params:xml:ns:xmpp-sasl")

	return marshalAndSend(conn, doc)
}

// <iq id="_bind_auth_2" type="set" xmlns="jabber:client">
// <bind xmlns="urn:ietf:params:xml:ns:xmpp-bind"/>
// </iq>

func sendIQBindAuth(conn *websocket.Conn) error {
	doc := etree.NewDocument()

	iq := doc.CreateElement("iq")
	iq.CreateAttr("id", "_bind_auth_2")
	iq.CreateAttr("type", "set")
	iq.CreateAttr("xmlns", "jabber:client")

	bind := iq.CreateElement("bind")
	bind.CreateAttr("xmlns", "urn:ietf:params:xml:ns:xmpp-bind")

	return marshalAndSend(conn, doc)
}

// <iq id="_session_auth_2" type="set" xmlns="jabber:client">
// <session xmlns="urn:ietf:params:xml:ns:xmpp-session"/>
// </iq>

func sendIQSessionAuth(conn *websocket.Conn) error {
	doc := etree.NewDocument()

	iq := doc.CreateElement("iq")
	iq.CreateAttr("id", "_session_auth_2")
	iq.CreateAttr("type", "set")
	iq.CreateAttr("xmlns", "jabber:client")

	session := iq.CreateElement("session")
	session.CreateAttr("xmlns", "urn:ietf:params:xml:ns:xmpp-session")

	return marshalAndSend(conn, doc)
}

// <enable resume="true" xmlns="urn:xmpp:sm:3"/>

func sendEnableResume(conn *websocket.Conn) error {
	doc := etree.NewDocument()

	iq := doc.CreateElement("enable")
	iq.CreateAttr("resume", "true")
	iq.CreateAttr("xmlns", "urn:xmpp:sm:3")

	return marshalAndSend(conn, doc)
}

/*
func sendDiscoRequest(conn *websocket.Conn, myJid string) error {
	// <iq id="b7f7d0ac-e294-419a-8abf-448bdf3f5c1b:sendIQ" to="meet.jitsi"
	//     type="get" xmlns="jabber:client">
	// <services xmlns="urn:xmpp:extdisco:2"/>
	// </iq>

	{
		doc := etree.NewDocument()

		iq := doc.CreateElement("iq")
		iq.CreateAttr("id", uuid.NewString()+":sendIQ")
		iq.CreateAttr("to", "meet.jitsi")
		iq.CreateAttr("type", "get")
		iq.CreateAttr("xmlns", "jabber:client")

		services := iq.CreateElement("services")
		services.CreateAttr("xmlns", "urn:xmpp:extdisco:2")

		err := marshalAndSend(conn, doc)
		if err != nil {
			return err
		}
	}

	// <iq from="10d34bda-1591-4c2b-9450-20d0987855aa@meet.jitsi/cM3HspdrTCR8"
	//     id="73266b7f-a39e-4de8-ba38-c2fe0f69c67a:sendIQ"
	//     to="meet.jitsi" type="get" xmlns="jabber:client">
	// <query xmlns="http://jabber.org/protocol/disco#info"/>
	// </iq>

	{
		doc := etree.NewDocument()

		iq := doc.CreateElement("iq")
		iq.CreateAttr("from", myJid)
		iq.CreateAttr("id", uuid.NewString()+":sendIQ")
		iq.CreateAttr("to", "meet.jitsi")
		iq.CreateAttr("type", "get")
		iq.CreateAttr("xmlns", "jabber:client")

		query := iq.CreateElement("query")
		query.CreateAttr("xmlns", "http://jabber.org/protocol/disco#info")

		err := marshalAndSend(conn, doc)
		if err != nil {
			return err
		}
	}

	return nil
}
*/

// <iq id="d2ddec97-4ed4-4773-b167-3bb95adf8284:sendIQ" to="focus.meet.jitsi"
//     type="set" xmlns="jabber:client">
// <conference machine-uid="b78a8c7633664e2eb576c27ee1c16295"
//     room="maki@muc.meet.jitsi" xmlns="http://jitsi.org/protocol/focus">
// <property name="startAudioMuted" value="10"/>
// <property name="startVideoMuted" value="10"/>
// <property name="rtcstatsEnabled" value="false"/>
// <property name="visitors-version" value="1"/>
// </conference>
// </iq>

func sendConferenceRequest(conn *websocket.Conn, room string) (string, error) {
	doc := etree.NewDocument()

	msgID := uuid.NewString() + ":sendIQ"

	iq := doc.CreateElement("iq")
	iq.CreateAttr("id", msgID)
	iq.CreateAttr("to", "focus.meet.jitsi")
	iq.CreateAttr("type", "set")
	iq.CreateAttr("xmlns", "jabber:client")

	conference := iq.CreateElement("conference")
	conference.CreateAttr(
		"machine-uid", strings.ReplaceAll(uuid.NewString(), "-", ""),
	)
	conference.CreateAttr("room", room+"@muc.meet.jitsi")
	conference.CreateAttr("xmlns", "http://jitsi.org/protocol/focus")

	for _, pair := range [][]string{
		{"startAudioMuted", "10"},
		{"startVideoMuted", "10"},
		{"rtcstatsEnabled", "false"},
		{"visitors-version", "1"},
	} {
		property := conference.CreateElement("property")
		property.CreateAttr("name", pair[0])
		property.CreateAttr("value", pair[1])
	}

	return msgID, marshalAndSend(conn, doc)
}

// <presence to="maki@muc.meet.jitsi/addf937b" xmlns="jabber:client">
// <x xmlns="http://jabber.org/protocol/muc"/>
// <stats-id>Everett-JDI</stats-id>
// <c hash="sha-1" node="https://jitsi.org/jitsi-meet" ver="W+eXiSPXEn8io7DtaLMtt3J13E4="
//     xmlns="http://jabber.org/protocol/caps"/>
// <SourceInfo>{}</SourceInfo>
// <jitsi_participant_codecList>av1,vp9,vp8,h264</jitsi_participant_codecList>
// <email>maki@hotmilk.space</email>
// <nick xmlns="http://jabber.org/protocol/nick">Maki</nick>
// </presence>

func sendPresence(
	conn *websocket.Conn, room string, myJid string,
	nick string, email string,
) error {
	doc := etree.NewDocument()

	presence := doc.CreateElement("presence")
	presence.CreateAttr("to", room+"@muc.meet.jitsi/"+myJid[:8])
	presence.CreateAttr("xmlns", "jabber:client")

	x := presence.CreateElement("x")
	x.CreateAttr("xmlns", "http://jabber.org/protocol/muc")

	statsID := presence.CreateElement("stats-id")
	statsID.SetText("Everett-JDI")

	c := presence.CreateElement("c")
	c.CreateAttr("hash", "sha-1")
	c.CreateAttr("node", "https://jitsi.org/jitsi-meet")
	c.CreateAttr("ver", "W+eXiSPXEn8io7DtaLMtt3J13E4=")
	c.CreateAttr("xmlns", "http://jabber.org/protocol/caps")

	sourceInfo := presence.CreateElement("SourceInfo")
	sourceInfo.SetText("{}")

	jitsiParticipantCodecList := presence.CreateElement("jitsi_participant_codecList")
	jitsiParticipantCodecList.SetText("av1,vp9,vp8,h264")

	if email != "" {
		emailEl := presence.CreateElement("email")
		emailEl.SetText(email)
	}

	nickEl := presence.CreateElement("nick")
	nickEl.CreateAttr("xmlns", "http://jabber.org/protocol/nick")
	nickEl.SetText(nick)

	return marshalAndSend(conn, doc)
}
