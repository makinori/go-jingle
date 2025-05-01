package jitsi

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/beevik/etree"
	"github.com/gorilla/websocket"
	"github.com/makinori/go-jitsi/jitsi/sdp"
)

type JitsiSessionInit struct {
	DidFirstOpen     bool
	DidAnonAuth      bool
	DidSecondOpen    bool
	DidIQBindAuth    bool
	DidIQSessionAuth bool
	DidEnableResume  bool

	ConferenceReqMsgID string
	DidConferenceReq   bool

	DidJingleInitiate bool
}

type JitsiSession struct {
	Host  string
	Room  string
	Nick  string
	Email string

	JID  string
	Init JitsiSessionInit
}

func (session *JitsiSession) handleMessage(conn *websocket.Conn, msg []byte) {
	fmt.Println("")
	fmt.Println("INCOMING: " + string(msg))

	doc := etree.NewDocument()

	err := doc.ReadFromBytes(msg)
	if err != nil {
		panic(err)
	}

	root := doc.Root()

	tag := root.Tag
	xmlns := root.SelectAttrValue("xmlns", "")

	init := &session.Init

	switch {

	case tag == "open" && xmlns == "urn:ietf:params:xml:ns:xmpp-framing":
		if !init.DidFirstOpen {
			init.DidFirstOpen = true
			sendAnonAuthSasl(conn)
		} else if !init.DidSecondOpen {
			init.DidSecondOpen = true
			sendIQBindAuth(conn)
		}

	case tag == "success" && xmlns == "urn:ietf:params:xml:ns:xmpp-sasl":
		if !init.DidAnonAuth {
			init.DidAnonAuth = true
			sendOpenFraming(conn)
		}

	case tag == "iq" && xmlns == "jabber:client":
		iqID := root.SelectAttrValue("id", "")
		iqType := root.SelectAttrValue("type", "")

		switch {

		case !init.DidIQBindAuth && iqID == "_bind_auth_2":
			if iqType != "result" {
				return
			}

			jidEl := root.FindElement("bind/jid")
			if jidEl == nil {
				return
			}

			init.DidIQBindAuth = true
			session.JID = jidEl.Text()
			fmt.Println("")
			fmt.Println("-----------------------------------------------------")
			fmt.Println("JID: " + session.JID)
			fmt.Println("-----------------------------------------------------")

			sendIQSessionAuth(conn)

		case !init.DidIQSessionAuth && iqID == "_session_auth_2":
			if iqType != "result" {
				return
			}

			init.DidIQSessionAuth = true

			sendEnableResume(conn)

		case !init.DidConferenceReq && iqID == init.ConferenceReqMsgID:
			conference := root.FindElement("conference")
			if conference == nil {
				return
			}

			ready := conference.SelectAttrValue("ready", "")
			if ready != "true" {
				return
			}

			init.DidConferenceReq = true

			sendPresence(conn, session.Room, session.JID, session.Nick, session.Email)

		case !init.DidJingleInitiate:
			jingle := root.FindElement("jingle")
			if jingle == nil {
				return
			}

			init.DidJingleInitiate = true

			doc.IndentTabs()
			doc.WriteToFile("jingle-incoming.xml")

			// TODO: continue from here

		}

	case tag == "enabled" && xmlns == "urn:xmpp:sm:3":
		if !init.DidEnableResume {
			init.DidEnableResume = true

			// sendDiscoRequest(conn, session.jid)
			init.ConferenceReqMsgID, _ = sendConferenceRequest(conn, session.Room)
		}

	}
}

func (session *JitsiSession) StartSession() {
	doc := etree.NewDocument()
	doc.ReadFromFile("jingle-incoming.xml")
	jingle := doc.FindElement("iq/jingle")

	sdp := sdp.SDP{
		IsP2P: false,
	}

	sdp.FromJingle(jingle)

	os.Exit(0)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{
		Scheme: "wss",
		Host:   session.Host,
		Path:   "/xmpp-websocket",
	}

	u.Query().Add("room", session.Room)

	log.Printf("connecting to %s", u.String())

	var dialer = &websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 45 * time.Second,
		Subprotocols:     []string{"xmpp"},
	}

	conn, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}

	defer conn.Close()

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				// log.Println("read:", err)
				continue
			}

			// log.Printf("recv: %s", message)

			if messageType != websocket.TextMessage {
				continue
			}

			session.handleMessage(conn, message)

		}
	}()

	// opening packet
	sendOpenFraming(conn)

	for {
		select {

		case <-done:
			return

		case <-interrupt:
			err := conn.WriteMessage(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			)

			if err != nil {
				log.Println("write close:", err)
				return
			}

			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return

		}
	}
}
