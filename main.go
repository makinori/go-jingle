package main

import "github.com/makinori/go-jitsi/jitsi"

func main() {
	session := jitsi.JitsiSession{
		Host: "jitsi.hotmilk.space",
		Room: "maki",
		Nick: "mikogo",
		// Email: "maki@hotmilk.space",
	}

	session.StartSession()
}
